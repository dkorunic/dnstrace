// Copyright (C) 2019  Dinko Korunic
// SPDX-License-Identifier: MIT

package cache

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func newA(name, ip string) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
		},
		A: net.ParseIP(ip).To4(),
	}
}

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestAddAndGet(t *testing.T) {
	c := New()
	rr := newA("example.com.", "1.2.3.4")
	c.Add("example.com.", rr)

	v, ok := c.Get("example.com.")
	if !ok {
		t.Fatal("Get() returned false for existing key")
	}

	if len(v) != 1 {
		t.Fatalf("expected 1 record, got %d", len(v))
	}
}

func TestGetMissing(t *testing.T) {
	c := New()

	_, ok := c.Get("missing.example.com.")
	if ok {
		t.Fatal("Get() returned true for missing key")
	}
}

func TestAddDeduplication(t *testing.T) {
	c := New()
	rr := newA("example.com.", "1.2.3.4")
	c.Add("example.com.", rr)
	c.Add("example.com.", rr)

	v, ok := c.Get("example.com.")
	if !ok {
		t.Fatal("Get() returned false")
	}

	if len(v) != 1 {
		t.Fatalf("expected 1 record after deduplication, got %d", len(v))
	}
}

func TestAddMultipleDistinct(t *testing.T) {
	c := New()
	c.Add("example.com.", newA("example.com.", "1.2.3.4"))
	c.Add("example.com.", newA("example.com.", "5.6.7.8"))

	v, ok := c.Get("example.com.")
	if !ok {
		t.Fatal("Get() returned false")
	}

	if len(v) != 2 {
		t.Fatalf("expected 2 records, got %d", len(v))
	}
}

func TestSetReplacesExisting(t *testing.T) {
	c := New()
	c.Add("example.com.", newA("example.com.", "1.2.3.4"))
	c.Set("example.com.", newA("example.com.", "5.6.7.8"))

	v, ok := c.Get("example.com.")
	if !ok {
		t.Fatal("Get() returned false after Set()")
	}

	if len(v) != 1 {
		t.Fatalf("expected 1 record after Set(), got %d", len(v))
	}

	if v[0].A.String() != "5.6.7.8" {
		t.Fatalf("expected IP 5.6.7.8 after Set(), got %s", v[0].A.String())
	}
}

func TestGetCaseInsensitive(t *testing.T) {
	c := New()
	c.Add("Example.COM.", newA("Example.COM.", "1.2.3.4"))

	_, ok := c.Get("example.com.")
	if !ok {
		t.Fatal("Get() should be case-insensitive")
	}
}

func TestGetRandReturnsFalseForMissing(t *testing.T) {
	c := New()

	_, ok := c.GetRand("missing.example.com.")
	if ok {
		t.Fatal("GetRand() returned true for missing key")
	}
}

func TestGetRandReturnsItemFromKnownSet(t *testing.T) {
	c := New()
	c.Add("example.com.", newA("example.com.", "1.2.3.4"))
	c.Add("example.com.", newA("example.com.", "5.6.7.8"))

	validIPs := map[string]bool{"1.2.3.4": true, "5.6.7.8": true}

	for range 20 {
		v, ok := c.GetRand("example.com.")
		if !ok {
			t.Fatal("GetRand() returned false for existing key")
		}

		if !validIPs[v.A.String()] {
			t.Fatalf("GetRand() returned unexpected IP: %s", v.A.String())
		}
	}
}
