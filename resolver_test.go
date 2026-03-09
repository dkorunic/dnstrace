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
	"net"
	"testing"

	"github.com/miekg/dns"
)

// makeTestA creates a dns.A record for use in tests.
func makeTestA(name, ip string) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
		},
		A: net.ParseIP(ip).To4(),
	}
}

// isRetryable tests

func TestIsRetryableTruncatedResponse(t *testing.T) {
	r := &dns.Msg{MsgHdr: dns.MsgHdr{Truncated: true}}

	if !isRetryable(r, nil) {
		t.Fatal("expected true for truncated response")
	}
}

func TestIsRetryableNilResponseNoError(t *testing.T) {
	if isRetryable(nil, nil) {
		t.Fatal("expected false for nil response and nil error")
	}
}

func TestIsRetryableNormalResponseNoError(t *testing.T) {
	r := &dns.Msg{}

	if isRetryable(r, nil) {
		t.Fatal("expected false for non-truncated response and nil error")
	}
}

func TestIsRetryableUDPTimeout(t *testing.T) {
	err := errors.New("read udp 127.0.0.1:53452->8.8.8.8:53: i/o timeout")

	if !isRetryable(nil, err) {
		t.Fatal("expected true for UDP i/o timeout error")
	}
}

func TestIsRetryableNonUDPError(t *testing.T) {
	err := errors.New("connection refused")

	if isRetryable(nil, err) {
		t.Fatal("expected false for non-UDP error")
	}
}

func TestIsRetryableReadTCPError(t *testing.T) {
	// TCP timeouts should NOT trigger retry (only "read udp ... i/o timeout" does)
	err := errors.New("read tcp 127.0.0.1:53452->8.8.8.8:53: i/o timeout")

	if isRetryable(nil, err) {
		t.Fatal("expected false for TCP timeout (only UDP triggers retry)")
	}
}

func TestIsRetryableUDPErrorWithoutTimeout(t *testing.T) {
	// Must have both "read udp" prefix AND "i/o timeout" suffix.
	err := errors.New("read udp 127.0.0.1:0->8.8.8.8:53: connection refused")

	if isRetryable(nil, err) {
		t.Fatal("expected false for UDP error without i/o timeout suffix")
	}
}

// getRRset tests

func TestGetRRsetMatchesNameAndType(t *testing.T) {
	rr := makeTestA("example.com.", "1.2.3.4")
	result := getRRset([]dns.RR{rr}, "example.com.", dns.TypeA)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
}

func TestGetRRsetFiltersByType(t *testing.T) {
	rr := makeTestA("example.com.", "1.2.3.4")
	result := getRRset([]dns.RR{rr}, "example.com.", dns.TypeMX)

	if len(result) != 0 {
		t.Fatalf("expected 0 results for wrong type, got %d", len(result))
	}
}

func TestGetRRsetFiltersByName(t *testing.T) {
	rr := makeTestA("example.com.", "1.2.3.4")
	result := getRRset([]dns.RR{rr}, "other.com.", dns.TypeA)

	if len(result) != 0 {
		t.Fatalf("expected 0 results for wrong name, got %d", len(result))
	}
}

func TestGetRRsetCaseInsensitive(t *testing.T) {
	rr := makeTestA("Example.COM.", "1.2.3.4")
	result := getRRset([]dns.RR{rr}, "example.com.", dns.TypeA)

	if len(result) != 1 {
		t.Fatalf("expected 1 result for case-insensitive name match, got %d", len(result))
	}
}

func TestGetRRsetEmptyInput(t *testing.T) {
	result := getRRset([]dns.RR{}, "example.com.", dns.TypeA)

	if len(result) != 0 {
		t.Fatalf("expected 0 results for empty input, got %d", len(result))
	}
}

func TestGetRRsetReturnsOnlyMatchingRecords(t *testing.T) {
	rr1 := makeTestA("example.com.", "1.2.3.4")
	rr2 := makeTestA("example.com.", "5.6.7.8")
	rr3 := makeTestA("other.com.", "9.10.11.12")

	result := getRRset([]dns.RR{rr1, rr2, rr3}, "example.com.", dns.TypeA)

	if len(result) != 2 {
		t.Fatalf("expected 2 results for example.com., got %d", len(result))
	}
}
