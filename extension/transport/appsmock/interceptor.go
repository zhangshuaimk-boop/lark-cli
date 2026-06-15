// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build appsmock

// Package appsmock provides a transport interceptor that routes
// apps-domain requests (URL path prefix `/open-apis/spark/`) to a local
// mock server pointed to by LARK_CLI_APPS_MOCK. Non-apps requests are
// passed through unmodified so other domains keep hitting the real
// open-platform gateway.
//
// This is a demo / development aid; it has no production use.
package appsmock

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	exttransport "github.com/larksuite/cli/extension/transport"
)

// EnvMockURL mirrors extension/credential/appsmock.EnvMockURL.
// (Duplicated here so this package has no dependency on the credential
// sibling — they are independently activatable.)
const EnvMockURL = "LARK_CLI_APPS_MOCK"

// AppsDomainPrefixes is the canonical list of URL path prefixes that
// belong to the apps domain. Generated from `shortcuts/apps/common.go`
// (`apiBasePath = "/open-apis/spark/v1"`) and confirmed against every
// hard-coded URL in shortcuts/apps/*.go on 2026-06-15.
//
// Keep in lock-step with apps shortcuts when upstream changes.
var AppsDomainPrefixes = []string{
	"/open-apis/spark/",
}

// IsAppsDomain reports whether path belongs to the apps domain.
func IsAppsDomain(path string) bool {
	for _, p := range AppsDomainPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// Provider implements exttransport.Provider for apps-mock mode.
type Provider struct{}

func (p *Provider) Name() string { return "appsmock" }

func (p *Provider) ResolveInterceptor(ctx context.Context) exttransport.Interceptor {
	raw := os.Getenv(EnvMockURL)
	if raw == "" {
		return nil
	}
	mockURL, err := url.Parse(raw)
	if err != nil || mockURL.Host == "" {
		fmt.Fprintf(os.Stderr, "WARNING: invalid %s=%q, appsmock interceptor disabled: %v\n",
			EnvMockURL, raw, err)
		return nil
	}
	scheme := mockURL.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return &Interceptor{mockScheme: scheme, mockHost: mockURL.Host}
}

// Interceptor rewrites apps-domain request URLs to point at the mock server.
type Interceptor struct {
	mockScheme string // typically "http"
	mockHost   string // e.g. "127.0.0.1:7878"
}

// PreRoundTrip rewrites req.URL.Scheme/Host when the request targets the
// apps domain. Non-apps requests are passed through unmodified.
//
// The built-in transport chain continues normally after we return; the
// underlying http.Transport dials the new host based on the rewritten URL.
// We also clear req.Host so net/http rebuilds the Host header from the
// new URL, otherwise SDK-set Host=open.feishu.cn would still go on the wire.
func (i *Interceptor) PreRoundTrip(req *http.Request) func(resp *http.Response, err error) {
	if !IsAppsDomain(req.URL.Path) {
		return nil // not apps domain, pass through unmodified
	}
	origHost := req.URL.Host
	req.URL.Scheme = i.mockScheme
	req.URL.Host = i.mockHost
	req.Host = "" // force net/http to rebuild from URL
	// Stamp an audit header so the mock server can verify routing worked
	// and a developer eyeballing tcpdump knows what hit the mock.
	req.Header.Set("X-Lark-Cli-Appsmock-Origin", origHost)
	return nil
}

func init() {
	if os.Getenv(EnvMockURL) == "" {
		return
	}
	exttransport.Register(&Provider{})
}
