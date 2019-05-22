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
	"fmt"
	"github.com/dkorunic/dnstrace/cache"
	"github.com/dkorunic/dnstrace/hints"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"net"
	"strconv"
	"strings"
	"time"
)

const defaultDNSTimeout = 2000 * time.Millisecond

var (
	roots  *hints.Hints
	aCache *cache.Cache
)

func doDNSQuery(qname string, qtype uint16, nsIP, nsLabel, zone string, rttIn time.Duration, sub bool) (time.Duration, error) {
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
				return rttIn, fmt.Errorf("failure to parse client-subnet IP: %s", *client)
			}

			if e.Address.To4() == nil {
				e.Family = 2 // IP6
				e.SourceNetmask = net.IPv6len * 8
			}
			o.Option = append(o.Option, e)
		}

		m.Extra = append(m.Extra, o)
	}

	r, rtt, err := c.Exchange(m, net.JoinHostPort(nsIP, strconv.Itoa(*port)))

Retry:
	// Truncated responses and UDP timeouts are candidates for retry
	if m.Truncated ||
		(err != nil && strings.HasPrefix(err.Error(), "read udp") &&
			strings.HasSuffix(err.Error(), "i/o timeout")) {

		if *fallback {
			// Enable EDNS and 4096 buffer size if previously not enabled
			if !*edns {
				color.Red("! Answer truncated, retrying with EDNS enabled and 4096 bytes as advertised payload size")

				o := new(dns.OPT)
				o.Hdr.Name = "."
				o.Hdr.Rrtype = dns.TypeOPT
				o.SetUDPSize(dns.DefaultMsgSize)
				m.Extra = append(m.Extra, o)

				r, rtt, err = c.Exchange(m, net.JoinHostPort(nsIP, strconv.Itoa(*port)))
				*edns = true

				goto Retry
			} else {
				// Retry with TCP
				color.Red("! Answer truncated, retrying with TCP")

				c.Net = "tcp"
				r, rtt, err = c.Exchange(m, net.JoinHostPort(nsIP, strconv.Itoa(*port)))
				*fallback = false

				goto Retry
			}
		}
	} else if err != nil {
		return rttIn, err
	}

	// Response ID mismatch
	if r.Id != m.Id {
		return rttIn, fmt.Errorf("id mismatch")
	}

	fmt.Printf("Answer RTT: %v from %v\n", rtt, nsIP)

	// Update A cache from A type RRs in ANSWER section
	for _, rr := range r.Answer {
		if rr.Header().Rrtype == dns.TypeA {
			rrn := strings.ToLower(rr.Header().Name)
			aCache.Add(rrn, rr.(*dns.A))
		}
	}

	// Update A cache from A type RRs in ADDITIONAL section
	for _, rr := range r.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			rrn := strings.ToLower(rr.Header().Name)
			aCache.Add(rrn, rr.(*dns.A))
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
				color.Green("%v", rr)
				color.Cyan("~ Following CNAME: %q/CNAME -> %q", rr.Header().Name,
					rr.(*dns.CNAME).Target)
				fmt.Printf("\n")

				return doDNSQuery(rr.(*dns.CNAME).Target, qtype, nsIP, nsLabel, zone, rtt+rttIn, sub)
			}
		}

		// End of response processing by default
		return rttIn + rtt, nil
	}

	// Process AUTHORITY section
	for _, ns := range r.Ns {
		if t, ok := ns.(*dns.NS); ok {
			nextNs := t.Ns
			resolvedNs := false

			// Attempt to match NS entries from AUTHORITY with A records in ADDITIONAL section
			if v, ok := aCache.GetRand(nextNs); ok {
				color.Cyan("+ Matched delegated NS and glue in additional section: %v(%v)",
					v.A, v.Header().Name)

				nextNs = v.A.String()
				resolvedNs = true
			}

			// We haven't managed to match glue (AUTHORITY/ADDITIONAL section records), so we need to resolve A
			// records for a NS in AUTHORITY section
			if !resolvedNs && !*ignoresub {
				color.Yellow("- No NS/glue match, we need extra lookups for %v", nextNs)
				fmt.Printf("\n")

				rLabel, rIP, _ := roots.GetRand()
				rtt2, err := doDNSQuery(nextNs, dns.TypeA, rIP, rLabel, ".", 0, true)
				if err != nil {
					return rttIn, err
				}

				// Check cached response
				if v, ok := aCache.GetRand(nextNs); ok {
					nextNs = v.A.String()
					rtt += rtt2
				} else {
					return rttIn, fmt.Errorf("unable to resolve %q/A (following authority NS)", nextNs)
				}
			}

			fmt.Print("\n")

			// Send query to the next server down the authority chain
			return doDNSQuery(qname, qtype, nextNs, t.Ns, t.Header().Name, rttIn+rtt, sub)
		}
	}

	return rtt + rttIn, fmt.Errorf("unable to resolve %q/%v @ %v(%v)", qname, dns.TypeToString[qtype],
		nsIP, nsLabel)
}

// getRRset returns RR set matching the given qname and qtype
func getRRset(rr []dns.RR, qname string, qtype uint16) []dns.RR {
	var rr1 []dns.RR
	for _, rr := range rr {
		if strings.ToLower(rr.Header().Name) == strings.ToLower(qname) && rr.Header().Rrtype == qtype {
			rr1 = append(rr1, rr)
		}
	}
	return rr1
}
