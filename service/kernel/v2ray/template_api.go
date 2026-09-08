package v2ray

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/go-leo/slicex"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/kernel/coreObj"
	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

func (t *Template) SetAPI(serverData *ServerData) (port int, err error) {
	// find a valid port
	config := configure.GetPortsNotNil()
	if config.Api.Port != 0 {
		port = config.Api.Port
	} else {
		for {
			if l, err := net.Listen("tcp4", "127.0.0.1:0"); err == nil {
				port = l.Addr().(*net.TCPAddr).Port
				_ = l.Close()
				break
			}
			time.Sleep(30 * time.Millisecond)
		}
	}
	services := []string{
		"LoggerService",
	}
	services = slicex.Uniq(append(services, config.Api.Services...))
	// observatory
	if serverData != nil {
		// Iterate over constructed outbounds to find balancer groups.
		// t.outNames() returns server tags (e.g. 「『group』_0」) as keys when they're in a group,
		// but serverData.OutboundName2ServerObjs is keyed by group name (e.g. "group").
		// So we need to walk t.Outbounds and collect balancer groups explicitly.
		for _, o := range t.Outbounds {
			if len(o.Balancers) == 0 {
				continue
			}

			// o.Tag is the group tag.
			groupTag := o.Tag
			// t.Outbounds only contains outbounds created by SetOutbound.
			// For balancer outbounds, o.Balancers is already populated by template_outbound.go
			// with the exact server outbound tags that exist in t.Outbounds.
			// Reuse that list directly so the selector always matches actual outbounds.
			selector := make([]string, len(o.Balancers))
			copy(selector, o.Balancers)
			if len(selector) == 0 {
				log.Warn("template_api: skipping balancer group %q because no server outbound tags were available (empty Balancers list)", groupTag)
				continue
			}
			log.Debug("template_api: reusing %d server tags from o.Balancers for group %q: %v", len(selector), groupTag, selector)

			// Derive the outbound name (key into serverData) from the group tag.
			// t.serverInfoMap stores info keyed by group tag (o.Tag), and serverInfo.OutboundName
			// is the actual outbound name used in settings maps (may differ from extracted group name).
			// Look up outbound name from serverInfoMap; fall back to extracted name if not found.
			var outboundName string
			if info, ok := t.serverInfoMap[groupTag]; ok && info != nil {
				outboundName = info.OutboundName
				log.Debug("template_api: resolved outbound name %q for balancer group %q via serverInfoMap", outboundName, groupTag)
			} else {
				log.Warn("template_api: group tag %q not found in serverInfoMap, falling back to extracted name %q (settings lookup may use different key)", groupTag, extractGroupName(groupTag))
				outboundName = extractGroupName(groupTag)
			}

			//TODO: random, leastload
			strategy := serverData.OutboundName2Setting[outboundName].Type
			interval, err := time.ParseDuration(serverData.OutboundName2Setting[outboundName].ProbeInterval)
			if err != nil {
				log.Warn("observatory: invalid duration %q for outbound group %q (key %q), using default 10s: %v",
					serverData.OutboundName2Setting[outboundName].ProbeInterval, groupTag, outboundName, err)
				interval = 10 * time.Second
			}

			t.Routing.Balancers = append(t.Routing.Balancers, coreObj.Balancer{
				Tag:      groupTag,
				Selector: selector,
				Strategy: coreObj.BalancerStrategy{
					Type: strategy.String(),
					Settings: &coreObj.StrategySettings{
						ObserverTag: groupTag,
					},
				},
			})

			if strings.ToLower(strategy.String()) == "leastping" {
				probeUrl := serverData.OutboundName2Setting[outboundName].ProbeURL
				if _, err := url.Parse(probeUrl); err != nil {
					log.Warn("observatory: invalid probe URL %q for outbound group %q (key %q), using default %q: %v",
						probeUrl, groupTag, outboundName, "https://gstatic.com/generate_204", err)
					probeUrl = "https://gstatic.com/generate_204"
				}

				// v2raya_core always uses MultiObservatory: one observer per balancer group.
				if t.MultiObservatory == nil {
					t.MultiObservatory = &coreObj.MultiObservatory{}
				}
				t.MultiObservatory.Observers = append(t.MultiObservatory.Observers, coreObj.ObservatoryItem{
					Tag: groupTag,
					Settings: coreObj.Observatory{
						SubjectSelector: selector,
						PingConfig: &coreObj.PingConfig{
							Destination: probeUrl,
							Interval:    interval.String(),
						},
						// Keep legacy fields for backward compatibility with older custom cores.
						ProbeURL:      probeUrl,
						ProbeInterval: interval.String(),
					},
				})
			}
		}
		if t.MultiObservatory != nil || t.Observatory != nil {
			// v2raya_core supports ObservatoryService via the v2ray-compat gRPC path.
			if t.Variant == where.V2rayaCore {

				var observatoryTags []string
				for _, o := range t.Outbounds {
					if len(o.Balancers) > 0 {
						observatoryTags = append(observatoryTags, o.Tag)
					}
				}
				t.ApiCloses = append(t.ApiCloses, ObservatoryProducer(port, observatoryTags))
			}
		}
	}
	t.API = &coreObj.APIObject{
		Tag:      "api-out",
		Services: services,
	}

	t.Inbounds = append(t.Inbounds, coreObj.Inbound{
		Port:     port,
		Protocol: "dokodemo-door",
		Listen:   "127.0.0.1",
		Settings: &coreObj.InboundSettings{
			Address: "127.0.0.1",
		},
		Tag: "api-in",
	})
	t.Routing.Rules = append(t.Routing.Rules, coreObj.RoutingRule{
		Type:        "field",
		InboundTag:  []string{"api-in"},
		OutboundTag: "api-out",
	})
	t.ApiPort = port
	return port, nil
}

// extractGroupName derives the raw group name from a GroupWrapper(tag).
// GroupWrapper wraps with 「『...』」, so this strips those markers.
// If the tag is not wrapped, it is returned as-is.
func extractGroupName(groupTag string) string {
	start := strings.Index(groupTag, "『")
	end := strings.Index(groupTag, "』")
	if start == -1 || end == -1 || end <= start {
		return groupTag
	}
	return groupTag[start+1 : end]
}
