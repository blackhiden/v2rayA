package v2ray

import (
	"reflect"
	"testing"

	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/kernel/coreObj"
	"github.com/v2rayA/v2rayA/kernel/serverObj"
	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
)

type loadBalanceTestServer struct {
	name string
}

func (s *loadBalanceTestServer) Configuration(info serverObj.PriorInfo) (serverObj.Configuration, error) {
	return serverObj.Configuration{
		CoreOutbound: coreObj.OutboundObject{
			Tag:      info.Tag,
			Protocol: "freedom",
		},
		UDPSupport: true,
	}, nil
}

func (s *loadBalanceTestServer) ExportToURL() string  { return "test://" + s.name }
func (s *loadBalanceTestServer) NeedPluginPort() bool { return false }
func (s *loadBalanceTestServer) ProtoToShow() string  { return "test" }
func (s *loadBalanceTestServer) GetProtocol() string  { return "test" }
func (s *loadBalanceTestServer) GetSecurity() string  { return "" }
func (s *loadBalanceTestServer) GetHostname() string  { return s.name + ".example" }
func (s *loadBalanceTestServer) GetPort() int         { return 443 }
func (s *loadBalanceTestServer) GetName() string      { return s.name }
func (s *loadBalanceTestServer) SetName(name string)  { s.name = name }

func newLoadBalanceServerData(enabled bool) *ServerData {
	infos := []serverInfo{
		{Info: &loadBalanceTestServer{name: "A"}, OutboundName: "PROXY"},
		{Info: &loadBalanceTestServer{name: "B"}, OutboundName: "PROXY"},
		{Info: &loadBalanceTestServer{name: "C"}, OutboundName: "PROXY"},
	}
	link2ServerObj := make(map[string]serverObj.ServerObj)
	link2ServerInfos := make(map[string][]*serverInfo)
	for i := range infos {
		link := infos[i].Info.ExportToURL()
		link2ServerObj[link] = infos[i].Info
		link2ServerInfos[link] = append(link2ServerInfos[link], &infos[i])
	}
	return &ServerData{
		ServerInfos: infos,
		OutboundName2ServerInfos: map[string][]*serverInfo{
			"PROXY": {&infos[0], &infos[1], &infos[2]},
		},
		Link2ServerObj:    link2ServerObj,
		Link2ServerInfos:  link2ServerInfos,
		OutboundName2ServerObjs: map[string][]serverObj.ServerObj{
			"PROXY": {&loadBalanceTestServer{name: "A"}, &loadBalanceTestServer{name: "B"}, &loadBalanceTestServer{name: "C"}},
		},
		OutboundName2Setting: map[string]configure.OutboundSetting{
			"PROXY": {
				LoadBalanceEnabled:  enabled,
				LoadBalanceStrategy: configure.LoadBalanceStrategyRoundRobin,
			},
		},
	}
}

func newLoadBalanceTemplate() *Template {
	return &Template{
		Setting:     configure.NewSetting(),
		Variant:     where.V2rayaCore,
		CoreVersion: "test",
	}
}

func TestResolveOutboundsCreatesOneRoundRobinBalancerPerEnabledGroup(t *testing.T) {
	template := newLoadBalanceTemplate()
	data := newLoadBalanceServerData(true)

	if _, _, err := template.resolveOutbounds(data); err != nil {
		t.Fatalf("resolveOutbounds returned an error: %v", err)
	}
	if len(template.balancerGroups) != 1 {
		t.Fatalf("expected one balancer group, got %d", len(template.balancerGroups))
	}

	group := template.balancerGroups[0]
	if group.Tag != "『PROXY』" || group.OutboundName != "PROXY" {
		t.Fatalf("unexpected group metadata: %#v", group)
	}
	wantSelector := []string{"『PROXY』_0", "『PROXY』_1", "『PROXY』_2"}
	if !reflect.DeepEqual(group.Selector, wantSelector) {
		t.Fatalf("expected selector %#v, got %#v", wantSelector, group.Selector)
	}

	template.setBalancerConfiguration(data, 0)
	if len(template.Routing.Balancers) != 1 {
		t.Fatalf("expected one generated routing balancer, got %d", len(template.Routing.Balancers))
	}
	balancer := template.Routing.Balancers[0]
	if balancer.Tag != group.Tag || balancer.Strategy.Type != configure.LoadBalanceStrategyRoundRobin {
		t.Fatalf("unexpected balancer: %#v", balancer)
	}
	if !reflect.DeepEqual(balancer.Selector, wantSelector) {
		t.Fatalf("expected generated selector %#v, got %#v", wantSelector, balancer.Selector)
	}

	for _, outbound := range template.Outbounds {
		if outbound.Protocol == "" {
			t.Fatalf("invalid empty-protocol outbound was generated for tag %q", outbound.Tag)
		}
	}
	template.Routing.Rules = append(template.Routing.Rules, coreObj.RoutingRule{OutboundTag: group.OutboundName})
	template.setGroupRouting()
	rule := template.Routing.Rules[len(template.Routing.Rules)-1]
	if rule.OutboundTag != "" || rule.BalancerTag != group.Tag {
		t.Fatalf("expected rule to target balancer %q, got %#v", group.Tag, rule)
	}
}

func TestResolveOutboundsKeepsDisabledGroupSingleServer(t *testing.T) {
	template := newLoadBalanceTemplate()
	data := newLoadBalanceServerData(false)

	if _, _, err := template.resolveOutbounds(data); err != nil {
		t.Fatalf("resolveOutbounds returned an error: %v", err)
	}
	if len(template.balancerGroups) != 0 {
		t.Fatalf("disabled group must not create balancers: %#v", template.balancerGroups)
	}
	if len(template.Outbounds) != 3 { // selected server, direct, block
		t.Fatalf("expected one active group member plus direct/block, got %d outbounds", len(template.Outbounds))
	}
	if template.Outbounds[0].Tag != "PROXY" {
		t.Fatalf("expected disabled group to use ordinary outbound tag PROXY, got %q", template.Outbounds[0].Tag)
	}
}

func TestActiveProxyOutboundNameUsesSelectedGroupOrConfiguredFallback(t *testing.T) {
	template := newLoadBalanceTemplate()
	data := newLoadBalanceServerData(true)

	if _, _, err := template.resolveOutbounds(data); err != nil {
		t.Fatalf("resolveOutbounds returned an error: %v", err)
	}
	if got, want := template.activeProxyOutboundName("PROXY"), "『PROXY』"; got != want {
		t.Fatalf("active balancer group = %q, want %q", got, want)
	}
	if got, want := template.activeProxyOutboundName("missing"), "『PROXY』"; got != want {
		t.Fatalf("missing active group fallback = %q, want %q", got, want)
	}
}

func TestDefaultProxyOutboundNameKeepsNamedGroupTargets(t *testing.T) {
	template := newLoadBalanceTemplate()
	data := newLoadBalanceServerData(true)
	if _, _, err := template.resolveOutbounds(data); err != nil {
		t.Fatalf("resolveOutbounds returned an error: %v", err)
	}
	if got, want := template.defaultProxyOutboundName(configure.DefaultOutboundName, "PROXY"), "『PROXY』"; got != want {
		t.Fatalf("reserved proxy route target = %q, want %q", got, want)
	}
	if got, want := template.defaultProxyOutboundName("direct", "PROXY"), "direct"; got != want {
		t.Fatalf("named route target = %q, want %q", got, want)
	}
}
