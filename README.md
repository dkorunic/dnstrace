dnstrace
===

[![GitHub license](https://img.shields.io/github/license/dkorunic/dnstrace.svg)](https://github.com/dkorunic/dnstrace/blob/master/LICENSE.txt)
[![GitHub release](https://img.shields.io/github/release/dkorunic/dnstrace.svg)](https://github.com/dkorunic/dnstrace/releases/latest)
[![Build Status](https://travis-ci.org/dkorunic/dnstrace.svg)](https://travis-ci.org/dkorunic/dnstrace)
[![Go Report Card](https://goreportcard.com/badge/github.com/dkorunic/dnstrace)](https://goreportcard.com/report/github.com/dkorunic/dnstrace)


## About

`dnstrace` is yet another DNS query/response tracing tool. Its purpose is to emulate iterative (with RD flag being unset) DNS queries usually being sent from any DNS recursor and traverse DNS authoritative hierarchy in search for the given query name and query type, assuming IN class. It will display every query and response, indicating reasons (delegations, CNAME following, missing glue etc.) for any new sub-query and additionally displaying individual and total response time.

EDNS is supported (with 4096 message payload size), as well as TCP failover in case of UDP communication issues (truncated messages and/or timeouts). When using EDNS it is also possible to manually (or automatically through [Ipify](https://www.ipify.org/)) specify Client-Subnet option to test for Geo-aware DNS responses.

It is also possible to set Recursion Desired flag which essentially disables DNS tracing and relies on local resolver (from `/etc/resolver.conf`) or remote DNS cache/resolver to perform all iterative queries and return the final result.

[![asciicast](https://asciinema.org/a/247701.svg)](https://asciinema.org/a/247701)

## Installation

There are two ways of installing `dnstrace`:

### Manual

Download your preferred flavor from [the releases](https://github.com/dkorunic/dnstrace/releases/latest) page and install manually.

### Using go get

```shell
go get github.com/dkorunic/dnstrace
```

## Usage

Usage:

```shell

Usage: ./dnstrace [option] [qtype] qname [@server]
Options:
  -client string
    	Sends EDNS Client Subnet option with specified IP address
  -edns
    	Enable EDNS support in queries (default true)
  -fallback
    	Fallback to 4K UDP message buffer size and then to TCP (default true)
  -ignoresub
    	Ignore tracing sub-requests when missing glue
  -port int
    	Use to send DNS queries to non-standard ports (default 53)
  -recurse
    	Toggle RD (recursion desired) flag in queries
  -tcp
    	Use TCP when querying DNS servers

NB: Nameserver (@server) will be ignored if not using recurse flag and random root nameserver will be used instead.
Client option (-client) accepts "external" keyword and will use ipify.org to get your public IP.  When using recurse (-recurse)
flag, if nameserver is not specified (@server), system resolver (from /etc/resolv.conf) will be used.  All boolean flags
accept true or false arguments, for instance "-edns=false"

This tool is typically used to establish worst-case scenario RTT for iterative queries being sent from resolvers and
doesn't necessarily reflect real life.
```

Typical use case is to specify one or more **qtypes** (MX, A, NS etc.) and one or more **qnames** (for example `apple.com`, `www.google.com`, etc.). When there is no qtype specified, A is assumed. If temporary result is CNAME and qtype is A, `dnstrace` will attempt to follow CNAME to the target. Internet **qclass** (IN) is assumed at all times.

It is possible to override default flag values by specifiying values, for example `-edns=false` or `-client=8.8.8.8`.

Typical use case would be:

```shell
dnstrace a porn.xxx
```

Then we are not interested in seeing sub-queries being sent to resolve missing glue for NS delegations, `-ignoresub` flag can be used.

## Bugs, feature requests, etc.

Please open a PR or report an issue. Thanks!
