// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build appsmock

// Package appsmock provides a dummy credential provider for the apps-mock
// demo. When LARK_CLI_APPS_MOCK is set (an HTTP URL pointing to a local
// mock server), this provider supplies placeholder account info and a
// dummy access token so the CLI's auth pipeline can proceed normally.
// The companion transport interceptor rewrites apps-domain requests to
// the mock server, which is expected to ignore Authorization headers.
//
// This package is build-tag gated; it has zero effect on default builds.
package appsmock

import (
	"context"
	"os"

	"github.com/larksuite/cli/extension/credential"
)

// EnvMockURL is the environment variable that activates apps-mock mode.
// Its value should be an http://host:port URL pointing to the local
// mock server (see demo/cmd/mockserver).
const EnvMockURL = "LARK_CLI_APPS_MOCK"

// DummyAppID is the placeholder app id reported to the credential layer.
// The mock server does not validate it.
const DummyAppID = "appsmock_demo_app"

// DummyToken is the placeholder access token. The mock server ignores
// Authorization headers entirely, so the value is arbitrary; it exists
// only to keep the credential pipeline happy.
const DummyToken = "appsmock-dummy-token"

// Provider is the credential provider for apps-mock mode.
type Provider struct{}

func (p *Provider) Name() string { return "appsmock" }

// Priority below 10 so it is consulted before the default env/file providers.
func (p *Provider) Priority() int { return 0 }

func (p *Provider) ResolveAccount(ctx context.Context) (*credential.Account, error) {
	if os.Getenv(EnvMockURL) == "" {
		return nil, nil // not active, defer to next provider
	}
	return &credential.Account{
		AppID:               DummyAppID,
		AppSecret:           credential.NoAppSecret,
		Brand:               credential.BrandFeishu,
		DefaultAs:           credential.IdentityUser,
		SupportedIdentities: credential.SupportsAll,
	}, nil
}

func (p *Provider) ResolveToken(ctx context.Context, req credential.TokenSpec) (*credential.Token, error) {
	if os.Getenv(EnvMockURL) == "" {
		return nil, nil
	}
	return &credential.Token{
		Value:  DummyToken,
		Scopes: "", // empty → scope pre-check is skipped
		Source: "appsmock",
	}, nil
}

func init() {
	if os.Getenv(EnvMockURL) == "" {
		return
	}
	credential.Register(&Provider{})
}
