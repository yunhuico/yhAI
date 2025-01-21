package tunnel

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/consts"
)

type (
	TunnelClient interface {
		RegisterHTTP(port int, subdomain string)
		Start(ctx context.Context) error
		Close()
	}

	frpClient struct {
		user       string
		subdomains map[string]int
		server     string
		token      string

		service *client.Service
	}
)

var _ TunnelClient = (*frpClient)(nil)

func NewTunnelClient(user, server, token string) *frpClient {
	return &frpClient{
		user:       user,
		subdomains: make(map[string]int),
		token:      token,
		server:     server,
	}
}

func (c *frpClient) RegisterHTTP(port int, subdomain string) {
	subdomain = strings.ToLower(subdomain)
	c.subdomains[subdomain] = port
}

func (c *frpClient) Close() {
	c.service.Close()
}

func (c *frpClient) Start(ctx context.Context) error {
	if err := c.init(); err != nil {
		return err
	}

	return c.service.Run()
}

func (c *frpClient) init() error {
	if len(c.subdomains) == 0 {
		return fmt.Errorf("should register at least one subdomain")
	}

	cfg := config.GetDefaultClientConf()
	cfg.LogLevel = "warn"
	cfg.ServerAddr = c.server
	ipStr, portStr, err := net.SplitHostPort(c.server)
	if err != nil {
		return fmt.Errorf("invalid tunnel server: %w", err)
	}
	cfg.ServerAddr = ipStr
	cfg.ServerPort, err = strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid tunnel server: %w", err)
	}
	cfg.Protocol = "tcp"
	cfg.Token = c.token
	cfg.User = c.user

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("frp config error: %v", err)
	}

	service, err := client.NewService(cfg, c.buildHTTPProxyConf(), nil, "")
	if err != nil {
		return fmt.Errorf("create frp service error: %w", err)
	}
	c.service = service
	return nil
}

func (c *frpClient) buildHTTPProxyConf() (proxyConfs map[string]config.ProxyConf) {
	proxyConfs = make(map[string]config.ProxyConf, len(c.subdomains))
	for subdomain, port := range c.subdomains {
		proxyName := c.user + "." + subdomain
		conf := &config.HTTPProxyConf{
			BaseProxyConf: config.BaseProxyConf{
				ProxyName: proxyName,
				ProxyType: consts.HTTPProxy,
				LocalSvrConf: config.LocalSvrConf{
					LocalIP:   "127.0.0.1",
					LocalPort: port,
				},
			},
			DomainConf: config.DomainConf{
				SubDomain: subdomain,
			},
		}
		proxyConfs[proxyName] = conf
	}
	return
}
