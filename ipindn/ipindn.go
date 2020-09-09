package ipindn

import (
	"net"
	"strings"
	"context"

	"github.com/miekg/dns"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"
)

type IPinDN struct {
	Next   plugin.Handler
	Fall fall.F

	// Origins is a list of zones we are authoritative for
	Origins []string
	// TTL sets the response TTL
	TTL uint32
}

// ServeDNS implements the middleware.Handler interface.
func (p IPinDN) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()

	zone := plugin.Zones(p.Origins).Matches(qname)
	if zone == "" {
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}

	if p.tryAnswerIP(w, r) {
		return dns.RcodeSuccess, nil
	}

	// Only on NXDOMAIN we will fallthrough.
	if p.Fall.Through(qname) {
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}

	// no fallthru so fail it
	return dns.RcodeServerFailure, nil
}

// Name implements the Handler interface.
func (p IPinDN) Name() string { return pluginName }

func (p *IPinDN) tryAnswerIP(w dns.ResponseWriter, r *dns.Msg) bool {
	if len(r.Question) <= 0 {
		return false
	}

	var rrs []dns.RR

	for i := 0; i < len(r.Question); i++ {
		question := r.Question[i]
		if question.Qclass != dns.ClassINET {
			continue
		}

		if question.Qtype == dns.TypeA || question.Qtype == dns.TypeAAAA {
			ip := p.parseIP(&question)
			if ip == nil {
				//log.Debugf("Try parsed IP from %s is nil", question.Name)
				continue
			}

			// support both ipv4 and ipv6
			if ip4 := ip.To4(); ip4 != nil {
				//log.Debugf("Parsed IP from %s as IPv4", question.Name)
				rrs = append(rrs, &dns.A{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    p.TTL,
					},
					A: ip,
				})
			} else {
				//log.Debugf("Parsed IP from %s as IPv6", question.Name)
				rrs = append(rrs, &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    p.TTL,
					},
					AAAA: ip,
				})
			}
		}
	}

	if len(rrs) > 0 {
		//log.Debugf("Answered with %d rrs", len(rrs))
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.Answer = rrs
		w.WriteMsg(m)
		return true
	}
	
	return false
}

func (p *IPinDN) parseIP(question *dns.Question) net.IP {
	//log.Debugf("Parse query for %s", question.Name)
	const minIPv4Len = 7
	const minIPv6Len = 15

	for _, domain := range p.Origins {
		if strings.HasSuffix(strings.ToLower(question.Name), domain) == false {
			//log.Debugf("Query %s skip handling by %s", question.Name, domain)
			continue
		}

		// `foo.172-32-22-12.example.com.` in `domain example.com.` with subdomain `foo.172-32-22-12.`
		subdomain := question.Name[:len(question.Name)-len(domain)]
		//log.Debugf("Query %s in domain %s with subdomain %s", question.Name, domain, subdomain)

		// do a quick check so we can be lazy sometimes
		if len(subdomain) < minIPv4Len {
			//log.Debugf("Query %s is too short to be IP", question.Name)
			return nil
		}

		// ["foo", "172-32-22-12", ""]
		subdomainParts := strings.Split(subdomain, ".")
		if len(subdomainParts) < 2 || subdomainParts[len(subdomainParts)-1] != "" {
			//log.Debugf("Query %s is not IPinDN format", question.Name)
			return nil
		}

		subdomainRoot := subdomainParts[len(subdomainParts)-2]
		if subdomainRoot == "" || strings.HasSuffix(subdomainRoot, "-") {
			//log.Debugf("Subdomain root ends with - or empty: '%s'", question.Name)
			return nil
		}

		if len(subdomainRoot) < minIPv4Len {
			return nil
		}

		// support ipv6
		if len(subdomainRoot) >= minIPv6Len {
			//log.Debugf("Try parse subdomain root as IPv6: '%s'", question.Name)
			return net.ParseIP(strings.Replace(subdomainRoot, "-", ":", -1))
		}
		
		//log.Debugf("Try parse subdomain root as IPv4: '%s'", question.Name)
		return net.ParseIP(strings.Replace(subdomainRoot, "-", ".", -1))
	}

	log.Errorf("Query for %s does not end with any registered zones", question.Name)
	return nil
}
