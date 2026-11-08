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
		"StatsService",
	}
	services = slicex.Uniq(append(services, config.Api.Services...))
	// observatory
	if serverData != nil {
		t.setBalancerConfiguration(serverData, port)
	}
	t.API = &coreObj.APIObject{
		Tag:      "api-out",
		Services: services,
	}
	t.Stats = &struct{}{}
	if t.Policy == nil {
		t.Policy = &coreObj.Policy{}
	}
	t.Policy.System.StatsOutboundUplink = true
	t.Policy.System.StatsOutboundDownlink = true
	t.ApiCloses = append(t.ApiCloses, TrafficProducer(port))

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

// setBalancerConfiguration adds routing balancers and any strategy-specific
// observatory configuration for enabled proxy groups.
func (t *Template) setBalancerConfiguration(serverData *ServerData, apiPort int) {
	if serverData == nil {
		return
	}
	for _, group := range t.balancerGroups {
		groupTag := group.Tag
		selector := append([]string(nil), group.Selector...)
		if len(selector) == 0 {
			log.Warn("template_api: skipping balancer group %q because no server outbound tags were available", groupTag)
			continue
		}
		outboundName := group.OutboundName

		outboundSetting := serverData.OutboundName2Setting[outboundName]
		strategy := outboundSetting.LoadBalanceStrategy

		t.Routing.Balancers = append(t.Routing.Balancers, coreObj.Balancer{
			Tag:      groupTag,
			Selector: selector,
			Strategy: coreObj.BalancerStrategy{
				Type: strategy,
			},
		})

		if strings.ToLower(strategy) == "leastping" {
			interval, err := time.ParseDuration(outboundSetting.ProbeInterval)
			if err != nil {
				log.Warn("observatory: invalid duration %q for outbound group %q (key %q), using default 10s: %v",
					outboundSetting.ProbeInterval, groupTag, outboundName, err)
				interval = 10 * time.Second
			}
			probeUrl := outboundSetting.ProbeURL
			if _, err := url.Parse(probeUrl); err != nil {
				log.Warn("observatory: invalid probe URL %q for outbound group %q (key %q), using default %q: %v",
					probeUrl, groupTag, outboundName, "https://gstatic.com/generate_204", err)
				probeUrl = "https://gstatic.com/generate_204"
			}

			// v2raya_core uses one observer per strategy that requires latency data.
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
			for _, group := range t.balancerGroups {
				observatoryTags = append(observatoryTags, group.Tag)
			}
			t.ApiCloses = append(t.ApiCloses, ObservatoryProducer(apiPort, observatoryTags))
		}
	}
}
