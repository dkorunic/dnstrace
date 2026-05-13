// Copyright (C) 2019  Dinko Korunic
// SPDX-License-Identifier: MIT

package main

import (
	"io"
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

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(ip)
}
