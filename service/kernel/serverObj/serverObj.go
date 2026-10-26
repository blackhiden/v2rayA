package serverObj

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/v2rayA/v2rayA/common"
	"github.com/v2rayA/v2rayA/conf"
	"github.com/v2rayA/v2rayA/kernel/coreObj"
	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
)

var ErrInvalidParameter = fmt.Errorf("malformed server link")

type ServerObj interface {
	Configuration(info PriorInfo) (c Configuration, err error)
	ExportToURL() string
	NeedPluginPort() bool
	ProtoToShow() string
	GetProtocol() string
	GetSecurity() string
	GetHostname() string
	GetPort() int
	GetName() string
	SetName(name string)
}

type Configuration struct {
	CoreOutbound            coreObj.OutboundObject
	ExtraOutbounds          []coreObj.OutboundObject
	PluginChain             string // The first is a server plugin, and the others are client plugins. Split by ",".
	UDPSupport              bool
	PluginManagerServerLink string
}

type PriorInfo struct {
	Variant     where.Variant
	CoreVersion string
	Tag         string
	PluginPort  int
	Backend     string // effective backend: "v2ray" or "" (daeuniverse/system default)
}

// BackendGetter is implemented by server types that support backend selection.
type BackendGetter interface {
	GetBackend() string
}

// EncryptionTypes returns the encryption and transport-security modes configured
// for a server. A server can expose more than one value: for example, a VMess
// server may use both a VMess cipher and TLS or Reality.
func EncryptionTypes(server ServerObj) []string {
	switch server := server.(type) {
	case *V2Ray:
		values := make([]string, 0, 2)
		if server.TLS == "" {
			values = append(values, "none")
		} else {
			values = append(values, server.TLS)
		}
		if server.Protocol == "vmess" && server.Security != "" {
			values = append(values, server.Security)
		}
		return uniqueEncryptionTypes(values)
	case *Shadowsocks:
		return uniqueEncryptionTypes([]string{server.Cipher})
	case *ShadowsocksR:
		return uniqueEncryptionTypes([]string{server.Cipher})
	case *Trojan:
		return []string{"tls"}
	case *Tuic, *Hysteria2, *Juicity:
		return []string{"tls"}
	case *AnyTLS:
		return []string{"anytls"}
	case *HTTP:
		if strings.EqualFold(server.Protocol, "https") {
			return []string{"tls"}
		}
		return []string{"none"}
	case *SOCKS:
		return []string{"none"}
	case *WireGuard:
		return []string{"wireguard"}
	case *Plugin:
		return []string{"unknown"}
	default:
		return []string{"unknown"}
	}
}

func uniqueEncryptionTypes(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return []string{"unknown"}
	}
	return result
}

func (info *PriorInfo) PluginObj() coreObj.OutboundObject {
	return coreObj.OutboundObject{
		Tag:      info.Tag,
		Protocol: "socks",
		Settings: coreObj.Settings{
			Servers: []coreObj.Server{
				{
					Address: "127.0.0.1",
					Port:    info.PluginPort,
				},
			}},
	}
}

type FromLinkCreator func(link string) (ServerObj, error)
type EmptyCreator func() (ServerObj, error)

var fromLinkCreators = make(map[string]FromLinkCreator)
var emptyCreators = make(map[string]EmptyCreator)

func FromLinkRegister(name string, creator FromLinkCreator) {
	fromLinkCreators[name] = creator
}
func EmptyRegister(name string, creator EmptyCreator) {
	emptyCreators[name] = creator
}
func New(name string) (ServerObj, error) {
	if creator, ok := emptyCreators[name]; ok {
		return creator()
	} else if pm := conf.GetEnvironmentConfig().PluginManager; pm != "" {
		// we do not support to override build-in protocols
		creator := emptyCreators[PluginManagerScheme]
		return creator()
	} else {
		return nil, fmt.Errorf("%q links are not supported; supported schemes: vmess, vless, ss, trojan, trojan-go, socks5, http, https, http-proxy, https-proxy, hysteria2, hy2, tuic, juicity, anytls, wireguard", name)
	}
}
func NewFromLink(name string, link string) (ServerObj, error) {
	if name == "ssr" || name == "shadowsocksr" {
		return nil, common.Coded("LINK_UNSUPPORTED_SCHEME", fmt.Errorf("%s", ssrUnsupported), map[string]interface{}{"scheme": name})
	}
	if creator, ok := fromLinkCreators[name]; ok {
		obj, err := creator(link)
		var coded *common.CodedError
		if errors.As(err, &coded) {
			return nil, err
		}
		if err != nil {
			return nil, common.Coded("LINK_MALFORMED", err, map[string]interface{}{
				"protocol": name,
				"detail":   strings.TrimPrefix(err.Error(), ErrInvalidParameter.Error()+": "),
			})
		}
		return obj, nil
	} else if pm := conf.GetEnvironmentConfig().PluginManager; pm != "" {
		// we do not support to override build-in protocols
		creator := fromLinkCreators[PluginManagerScheme]
		return creator(link)
	} else {
		return nil, common.Coded("LINK_UNSUPPORTED_SCHEME", fmt.Errorf("%q links are not supported; supported schemes: vmess, vless, ss, trojan, trojan-go, socks5, http, https, http-proxy, https-proxy, hysteria2, hy2, tuic, juicity, anytls, wireguard", name), map[string]interface{}{"scheme": name})
	}
}

func setValue(values *url.Values, key string, value string) {
	if value == "" {
		return
	}
	values.Set(key, value)
}
