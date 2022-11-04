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
	"math/rand"

	"github.com/sean-/seed"
)

type Root struct {
	Name        string
	IPv4Address string
	IPv6Address string
}

type Hints struct {
	hints []Root
}

func init() {
	seed.Init()
}

// New returns a new and initialized root nameserver hints db
func New() *Hints {
	return &Hints{hints: []Root{
		{"a.root-servers.net.", "198.41.0.4", "2001:503:ba3e::2:30"},
		{"b.root-servers.net.", "192.228.79.201", "2001:478:65::53"},
		{"c.root-servers.net.", "192.33.4.12", "2001:500:2::c"},
		{"d.root-servers.net.", "199.7.91.13", "2001:500:2d::d"},
		{"e.root-servers.net.", "192.203.230.10", "NASA"},
		{"f.root-servers.net.", "192.5.5.241", "2001:500:2f::f"},
		{"g.root-servers.net.", "192.112.36.4", "U.S."},
		{"h.root-servers.net.", "128.63.2.53", "2001:500:1::803f:235"},
		{"i.root-servers.net.", "192.36.148.17", "2001:7FE::53"},
		{"j.root-servers.net.", "192.58.128.30", "2001:503:c27::2:30"},
		{"k.root-servers.net.", "193.0.14.129", "2001:7fd::1"},
		{"l.root-servers.net.", "199.7.83.42", "2001:500:3::42"},
		{"m.root-servers.net.", "202.12.27.33", "2001:dc3::35"},
	}}
}

// Get returns an array of root nameserver hints
func (h *Hints) Get() []Root {
	return h.hints
}

// GetRand returns a randomized item (root nameserver) from root hints
func (h *Hints) GetRand() (string, string, string) {
	n := rand.Int() % len(h.hints)
	return h.hints[n].Name, h.hints[n].IPv4Address, h.hints[n].IPv6Address
}
