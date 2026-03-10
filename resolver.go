// @license
// Copyright (C) 2019  Dinko Korunic
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/dkorunic/dnstrace/cache"
	"github.com/dkorunic/dnstrace/hints"
	"github.com/fatih/color"
	"github.com/miekg/dns"
)

const (
	defaultDNSTimeout = 2000 * time.Millisecond
	maxQueryDepth     = 20
)

var (
	roots                *hints.Hints
	aCache               *cache.Cache
	ErrParseClientSubnet = errors.New("failure to parse client-subnet IP")
	ErrIdMismatch        = errors.New("id mismatch")
	ErrResolve           = errors.New("unable to resolve")
)

// isRetryable returns true when the exchange result warrants a fallback retry:
// either the response was truncated or a UDP read timed out.
func isRetryable(r *dns.Msg, err error) bool {
	if r != nil && r.Truncated {
		return true
	}

	return err != nil &&
		strings.HasPrefix(err.Error(), "read udp") &&
		strings.HasSuffix(err.Error(), "i/o timeout")
}

// exchangeWithFallback sends a DNS query and, if the result is retryable,
// upgrades first to EDNS (if not already enabled) and then to TCP.
// It never mutates the global *edns or *fallback flags.
func exchangeWithFallback(c *dns.Client, m *dns.Msg, addr string) (*dns.Msg, time.Duration, error) {
	r, rtt, err := c.Exchange(m, addr)

	if !isRetryable(r, err) || !*fallback {
		return r, rtt, err
	}

	// Stage 1: upgrade to EDNS if not already active.
	// Check via IsEdns0() rather than the *edns flag, because an OPT record
	// may already be present (e.g. when *client != "" forces one regardless).
	// Clone the message before mutating Extra so the caller's dns.Msg is not affected.
	if m.IsEdns0() == nil {
		color.Red("! Answer truncated, retrying with EDNS enabled and 4096 bytes as advertised payload size")

		m = m.Copy()
		o := new(dns.OPT)
		o.Hdr.Name = "."
		o.Hdr.Rrtype = dns.TypeOPT
		o.SetUDPSize(dns.DefaultMsgSize)
		m.Extra = append(m.Extra, o)

		r, rtt, err = c.Exchange(m, addr)
	}

	// Stage 2: upgrade to TCP if still retryable.
	if isRetryable(r, err) {
		color.Red("! Answer truncated, retrying with TCP")

		c.Net = "tcp"
		r, rtt, err = c.Exchange(m, addr)
	}

	return r, rtt, err
}

func doDNSQuery(qname string, qtype uint16, nsIP, nsLabel, zone string, rttIn time.Duration, sub bool, depth int) (time.Duration, error) {
	if depth > maxQueryDepth {
		return rttIn, fmt.Errorf("%w: max query depth exceeded for %q/%v", ErrResolve, qname, dns.TypeToString[qtype])
	}

	fmt.Printf("Querying about %q/%v @ %v(%v, %q authority)\n", qname,
		dns.TypeToString[qtype], nsIP, nsLabel, zone)

	// Check cache first
	if qtype == dns.TypeA {
		if v, ok := aCache.Get(qname); ok {
			fmt.Printf("Answer RTT: 0ms (cached from previous responses)\n")
			if sub {
				color.Green("Final response (sub-query):")
			} else {
				color.Green("Final response:")
			}
			for _, rr := range v {
				color.Green("%v", rr)
			}

			return rttIn, nil
		}
	}

	m := new(dns.Msg)
	m.Id = dns.Id()
	m.RecursionDesired = *recurse
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{Name: qname, Qtype: qtype, Qclass: dns.ClassINET}

	c := new(dns.Client)
	c.Timeout = defaultDNSTimeout
	if *tcp {
		c.Net = "tcp"
	}

	// Enable EDNS and 4096 buffer size; using Client-Subnet also requires EDNS
	if *edns || *client != "" {
		o := new(dns.OPT)
		o.Hdr.Name = "."
		o.Hdr.Rrtype = dns.TypeOPT
		o.SetUDPSize(dns.DefaultMsgSize)

		// Set EDNS Client-Subnet
		if *client != "" {
			e := &dns.EDNS0_SUBNET{
				Code:          dns.EDNS0SUBNET,
				Address:       net.ParseIP(*client),
				Family:        1, // IP4
				SourceNetmask: net.IPv4len * 8,
			}

			if e.Address == nil {
				return rttIn, fmt.Errorf("%w: %v", ErrParseClientSubnet, *client)
			}

			if e.Address.To4() == nil {
				e.Family = 2 // IP6
				e.SourceNetmask = net.IPv6len * 8
			}
			o.Option = append(o.Option, e)
		}

		m.Extra = append(m.Extra, o)
	}

	addr := net.JoinHostPort(nsIP, strconv.Itoa(int(*port)))

	r, rtt, err := exchangeWithFallback(c, m, addr)
	if err != nil {
		return rttIn, err
	}

	if r == nil {
		return rttIn, fmt.Errorf("%w: nil response from %v(%v)", ErrResolve, nsIP, nsLabel)
	}

	// Response ID mismatch
	if r.Id != m.Id {
		return rttIn, fmt.Errorf("%w", ErrIdMismatch)
	}

	fmt.Printf("Answer RTT: %v from %v\n", rtt, nsIP)

	// Update A cache from A type RRs in ANSWER section
	for _, rr := range r.Answer {
		if rr.Header().Rrtype == dns.TypeA {
			if a, ok := rr.(*dns.A); ok {
				rrn := strings.ToLower(rr.Header().Name)
				aCache.Add(rrn, a)
			}
		}
	}

	// Update A cache from A type RRs in ADDITIONAL section
	for _, rr := range r.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			if a, ok := rr.(*dns.A); ok {
				rrn := strings.ToLower(rr.Header().Name)
				aCache.Add(rrn, a)
			}
		}
	}

	// Process ANSWER section
	if len(r.Answer) > 0 {
		// Process ANSWER RR subset matching query name and query type
		rrTypeSub := getRRset(r.Answer, qname, qtype)
		if len(rrTypeSub) > 0 {
			if sub {
				color.Green("Final response (sub-query):")
			} else {
				color.Green("Final response:")
			}
			for _, rr := range rrTypeSub {
				color.Green("%v", rr)
			}

			return rttIn + rtt, nil
		}

		// Process ANSWER RR subset matching query name and CNAME
		rrCnameSub := getRRset(r.Answer, qname, dns.TypeCNAME)
		if len(rrCnameSub) > 0 {
			color.Green("Got CNAME in response:")
			for _, rr := range rrCnameSub {
				if cn, ok := rr.(*dns.CNAME); ok {
					color.Green("%v", rr)
					color.Cyan("~ Following CNAME: %q/CNAME -> %q", rr.Header().Name, cn.Target)

					// In-zone CNAME: continue with the current nameserver.
					// Exclude the root zone (".") because every name is technically
					// in-zone there, but root servers only delegate — never answer.
					if zone != "." && strings.HasSuffix(cn.Target, "."+zone) {
						fmt.Printf("\n")

						return doDNSQuery(cn.Target, qtype, nsIP, nsLabel, zone, rtt+rttIn, sub, depth+1)
					}

					// Out-of-zone CNAME target: restart from root nameservers
					nsLabel, nsIP, _ = roots.GetRand()
					color.Yellow("~ Out of zone CNAME target, sub-query will restart from \".\"")
					fmt.Printf("\n")

					return doDNSQuery(cn.Target, qtype, nsIP, nsLabel, ".", rtt+rttIn, sub, depth+1)
				}
			}
		}

		// End of response processing by default
		return rttIn + rtt, nil
	}

	// Process AUTHORITY section — try every NS record before giving up
	var firstNsErr error

	for _, ns := range r.Ns {
		t, ok := ns.(*dns.NS)
		if !ok {
			continue
		}

		// Normalise the NS hostname to a fully-qualified name so cache
		// lookups match regardless of trailing-dot presence in the response.
		nextNs := dns.Fqdn(t.Ns)
		nsRtt := rtt
		resolved := false

		// Attempt to match NS entries from AUTHORITY with A records in ADDITIONAL section
		if v, ok := aCache.GetRand(nextNs); ok {
			color.Cyan("+ Matched delegated NS and glue in additional section: %v(%v)",
				v.A, v.Header().Name)

			nextNs = v.A.String()
			resolved = true
		}

		// No glue available; resolve the NS hostname to an A record via sub-query
		if !resolved && !*ignoresub {
			color.Yellow("- No NS/glue match, we need extra lookups for %v", t.Ns)
			fmt.Printf("\n")

			rLabel, rIP, _ := roots.GetRand()

			rtt2, err := doDNSQuery(dns.Fqdn(t.Ns), dns.TypeA, rIP, rLabel, ".", 0, true, depth+1)
			if err != nil {
				if firstNsErr == nil {
					firstNsErr = err
				}

				continue
			}

			if v, ok := aCache.GetRand(nextNs); ok {
				nextNs = v.A.String()
				nsRtt += rtt2
			} else {
				nsErr := fmt.Errorf("%w: %q/A (following authority NS)", ErrResolve, t.Ns)
				if firstNsErr == nil {
					firstNsErr = nsErr
				}

				continue
			}
		} else if !resolved {
			// ignoresub is set and no glue is available: skip this NS
			continue
		}

		fmt.Print("\n")

		// Send query to the next server down the authority chain
		result, err := doDNSQuery(qname, qtype, nextNs, t.Ns, t.Header().Name, rttIn+nsRtt, sub, depth+1)
		if err == nil {
			return result, nil
		}

		if firstNsErr == nil {
			firstNsErr = err
		}
	}

	if firstNsErr != nil {
		return rttIn, firstNsErr
	}

	return rtt + rttIn, fmt.Errorf("%w: %q/%v @ %v(%v)", ErrResolve, qname, dns.TypeToString[qtype],
		nsIP, nsLabel)
}

// getRRset returns RR set matching the given qname and qtype
func getRRset(rr []dns.RR, qname string, qtype uint16) []dns.RR {
	var rr1 []dns.RR
	for _, rr := range rr {
		if strings.EqualFold(rr.Header().Name, qname) && rr.Header().Rrtype == qtype {
			rr1 = append(rr1, rr)
		}
	}

	return rr1
}
