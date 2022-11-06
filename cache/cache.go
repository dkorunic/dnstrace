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

package cache

import (
	"math/rand"
	"strings"
	"sync"

	"filippo.io/mostly-harmless/cryptosource"

	"github.com/miekg/dns"
)

type Cache struct {
	items map[string][]*dns.A
	m     sync.RWMutex
	r     *rand.Rand
}

const (
	defaultSize = 100
)

// New returns a new and initialized A dns cache.
func New() *Cache {
	return &Cache{
		items: make(map[string][]*dns.A, defaultSize),
		r:     rand.New(cryptosource.New()), //nolint:gosec
	}
}

// Set adds a single item to the cache, replacing all existing items.
func (c *Cache) Set(qname string, rr *dns.A) {
	c.m.Lock()
	defer c.m.Unlock()

	c.items[strings.ToLower(qname)] = append([]*dns.A(nil), rr)
}

// Add an item to the cache only if item doesn't exist for a given key.
func (c *Cache) Add(qname string, rr *dns.A) {
	c.m.Lock()
	defer c.m.Unlock()

	// Check if rr is duplicate and skip adding if true
	if v, ok := c.items[strings.ToLower(qname)]; ok {
		for _, rr1 := range v {
			if dns.IsDuplicate(rr, rr1) {
				return
			}
		}
	}

	c.items[strings.ToLower(qname)] = append(c.items[strings.ToLower(qname)], rr)
}

// Get an item (list) from the cache, returning an item and a boolean indicating if the key has been found.
func (c *Cache) Get(qname string) ([]*dns.A, bool) {
	c.m.RLock()
	defer c.m.RUnlock()

	v, ok := c.items[strings.ToLower(qname)]
	if !ok {
		return nil, false
	}

	return v, true
}

// GetRand gets a randomized item (single item) from the cache, returning also a boolean indicating if the key has been
// found.
func (c *Cache) GetRand(qname string) (*dns.A, bool) {
	c.m.RLock()
	defer c.m.RUnlock()

	v, ok := c.items[strings.ToLower(qname)]
	if !ok {
		return nil, false
	}

	// Randomized item for a given key
	n := c.r.Int() % len(v)

	return v[n], true
}
