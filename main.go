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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dkorunic/dnstrace/cache"
	"github.com/dkorunic/dnstrace/hints"
	"github.com/fatih/color"
	"github.com/miekg/dns"
)

var (
	recurse   = flag.Bool("recurse", false, "Toggle RD (recursion desired) flag in queries")
	edns      = flag.Bool("edns", true, "Enable EDNS support in queries")
	fallback  = flag.Bool("fallback", true, "Fallback to 4K UDP message buffer size and then to TCP")
	ignoresub = flag.Bool("ignoresub", false, "Ignore tracing sub-requests when missing glue")
	tcp       = flag.Bool("tcp", false, "Use TCP when querying DNS servers")
	client    = flag.String("client", "", "Sends EDNS Client Subnet option with specified IP address")
	port      = flag.Uint("port", 53, "Use to send DNS queries to non-standard ports")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %v [option] [qtype] qname [@server]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Printf(`
NB: Nameserver (@server) will be ignored if not using recurse flag and random root nameserver will be used instead. 
Client option (-client) accepts "external" keyword and will use ipify.org to get your public IP.  When using recurse (-recurse)
flag, if nameserver is not specified (@server), system resolver (from /etc/resolv.conf) will be used.  All boolean flags
accept true or false arguments, for instance "-edns=false"

This tool is typically used to establish worst-case scenario RTT for iterative queries being sent from resolvers and
doesn't necessarily reflect real life.
`)
		os.Exit(0)
	}

	var nsIP string
	var nsLabel string
	var qname []string
	var qtype []uint16

	flag.Parse()
	for _, arg := range flag.Args() {
		// Nameserver starts with '@'
		if arg[0] == '@' {
			nsIP = arg

			continue
		}

		// Presume next argument is qtype and attempt to match
		if v, ok := dns.StringToType[strings.ToUpper(arg)]; ok {
			qtype = append(qtype, v)

			continue
		}

		// Everything else is a qname
		qname = append(qname, arg)
	}

	// Qname is a mandatory argument
	if len(qname) == 0 {
		color.Red("Error: missing qname argument (target DNS label).")
		fmt.Printf("\n")
		flag.Usage()
	}

	// If qtype is missing, presume A
	if len(qtype) == 0 {
		qtype = append(qtype, dns.TypeA)
	}

	// Root hints initialisation
	roots = hints.New()

	// Use any of 13 root nameservers if we don't use recursion
	if !*recurse {
		nsLabel, nsIP, _ = roots.GetRand()
	} else {
		// Attempt to parse resolv.conf and use any nameservers if we don't have any provided
		if len(nsIP) == 0 {
			conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
			if err != nil {
				color.Red("Error getting nameservers from resolv.conf: %v", err)
				os.Exit(1)
			}

			// Use just the first nameserver
			nsIP = conf.Servers[0]
		} else {
			// We have a provided NS, so just strip the '@'
			nsIP = nsIP[1:]
		}
		nsLabel = nsIP
	}

	// Use Ipify for EDNS client-subnet
	if *client == clientExternal {
		*client = getIpify()
	}

	// A cache initialisation
	aCache = cache.New()

	// Resolve all qtypes for all qnames provided
	for _, qn := range qname {
		for _, qt := range qtype {
			fmt.Printf("Query name: %q, type: %v, nameserver IP: %v, nameserver label: %v\n"+
				"Query options: recurse: %v, EDNS: %v, client-subnet: %q, fallback: %v, TCP: %v, ignore sub-queries: %v\n\n",
				qn, dns.TypeToString[qt], nsIP, nsLabel, *recurse, *edns, *client, *fallback, *tcp, *ignoresub)

			rtt, err := doDNSQuery(dns.Fqdn(qn), qt, nsIP, nsLabel, ".", 0, false)
			if err != nil {
				color.Red("Error: %v", err)
			}

			fmt.Printf("\nTotal RTT: %v\n\n", rtt)
		}
	}
}
