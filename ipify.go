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
	"io/ioutil"
	"net/http"
	"time"
)

const (
	defaultHTTPTImeout = 4000 * time.Millisecond
	IpifyURL           = "https://api.ipify.org?format=text"
	clientExternal     = "external"
)

// getIpify returns external IP address as from ipify.org
func getIpify() string {
	c := http.Client{Timeout: defaultHTTPTImeout}

	resp, err := c.Get(IpifyURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	return string(ip)
}
