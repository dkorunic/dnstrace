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

package hints

import (
	"net"
	"testing"
)

func TestNew(t *testing.T) {
	h := New()
	if h == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetReturns13Roots(t *testing.T) {
	h := New()

	if got := len(h.Get()); got != 13 {
		t.Fatalf("expected 13 root servers, got %d", got)
	}
}

func TestAllHintsHaveNonEmptyFields(t *testing.T) {
	h := New()

	for _, r := range h.Get() {
		if r.Name == "" {
			t.Error("root server has empty Name")
		}

		if r.IPv4Address == "" {
			t.Errorf("root server %q has empty IPv4Address", r.Name)
		}

		if r.IPv6Address == "" {
			t.Errorf("root server %q has empty IPv6Address", r.Name)
		}
	}
}

func TestAllHintsHaveValidIPs(t *testing.T) {
	h := New()

	for _, r := range h.Get() {
		if net.ParseIP(r.IPv4Address) == nil {
			t.Errorf("root server %q has invalid IPv4: %q", r.Name, r.IPv4Address)
		}

		if net.ParseIP(r.IPv6Address) == nil {
			t.Errorf("root server %q has invalid IPv6: %q", r.Name, r.IPv6Address)
		}
	}
}

func TestGetRandReturnsKnownServer(t *testing.T) {
	h := New()

	known := make(map[string]bool)
	for _, r := range h.Get() {
		known[r.Name] = true
	}

	for range 50 {
		name, _, _ := h.GetRand()
		if !known[name] {
			t.Fatalf("GetRand() returned unknown server name: %q", name)
		}
	}
}

func TestGetRandReturnsConsistentNameAndIPv4(t *testing.T) {
	h := New()

	nameToIPv4 := make(map[string]string)
	for _, r := range h.Get() {
		nameToIPv4[r.Name] = r.IPv4Address
	}

	name, ipv4, _ := h.GetRand()
	if nameToIPv4[name] != ipv4 {
		t.Fatalf("GetRand() name/IPv4 mismatch: name=%q ipv4=%q", name, ipv4)
	}
}

func TestGetRandReturnsConsistentNameAndIPv6(t *testing.T) {
	h := New()

	nameToIPv6 := make(map[string]string)
	for _, r := range h.Get() {
		nameToIPv6[r.Name] = r.IPv6Address
	}

	name, _, ipv6 := h.GetRand()
	if nameToIPv6[name] != ipv6 {
		t.Fatalf("GetRand() name/IPv6 mismatch: name=%q ipv6=%q", name, ipv6)
	}
}
