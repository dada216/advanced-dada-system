package plugin

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// Service is the interface that we're exposing as a plugin.
type Service interface {
	RunTask(args map[string]string) (string, error)
}

// ServiceRPC is an implementation that translates an RPC call to the Service interface
type ServiceRPC struct{ Impl Service }

func (p *ServiceRPC) RunTask(args map[string]string, resp *string) error {
	result, err := p.Impl.RunTask(args)
	*resp = result
	return err
}

// ServiceRPCClient is an implementation that translates the Service interface to an RPC call
type ServiceRPCClient struct{ client *rpc.Client }

func (m *ServiceRPCClient) RunTask(args map[string]string) (string, error) {
	var resp string
	err := m.client.Call("Plugin.RunTask", args, &resp)
	return resp, err
}

// ServicePlugin is the implementation of plugin.Plugin so we can serve/consume this
type ServicePlugin struct {
	Impl Service
}

func (p *ServicePlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ServiceRPC{Impl: p.Impl}, nil
}

func (ServicePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ServiceRPCClient{client: c}, nil
}

// HandshakeConfig is a common handshake that is shared by plugin and host.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ADS_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "advanced-dada-system-v3",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"service": &ServicePlugin{},
}
