package ipindn

import (
	"strconv"

	"github.com/caddyserver/caddy"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

const (
	pluginName = "ipindn"
)

var log = clog.NewWithPlugin(pluginName)

func init() {
	caddy.RegisterPlugin(pluginName, caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	ipdn, err := parseConfig(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		ipdn.Next = next
		return ipdn
	})

	return nil
}

func parseConfig(c *caddy.Controller) (IPinDN, error) {
	ipdn := IPinDN{
		TTL: 30,
	}

	i := 0
	for c.Next() {
		// this plugin is single instance only
		if i > 0 {
			return ipdn, plugin.Error(pluginName, plugin.ErrOnce)
		}
		i++

		args := c.RemainingArgs()
		if len(args) < 1 {
			return ipdn, plugin.Error(pluginName, c.Err("at least 1 zone must be specified"))
		}

		origins := make([]string, len(c.ServerBlockKeys))
		copy(origins, c.ServerBlockKeys)
		origins = args

		for i := range origins {
			origins[i] = plugin.Host(origins[i]).Normalize()
			log.Infof("register zone %s", origins[i])
		}
		ipdn.Origins = origins

		for c.NextBlock() {
			switch c.Val() {
			case "fallthrough":
				ipdn.Fall.SetZonesFromArgs(c.RemainingArgs())
			case "ttl":
				remaining := c.RemainingArgs()
				if len(remaining) < 1 {
					return ipdn, c.Errf("ttl must be followed by an integer between 1-65535")
				}
				ttl, err := strconv.Atoi(remaining[0])
				if err != nil {
					return ipdn, c.Errf("ttl should be an integer between 1-65535")
				}
				if ttl < 1 || ttl > 65535 {
					return ipdn, c.Errf("ttl out of range (expect 1-65535)")
				}
				ipdn.TTL = uint32(ttl)
			default:
				return ipdn, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	return ipdn, nil
}