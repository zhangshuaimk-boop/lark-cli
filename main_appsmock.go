// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build appsmock

// Build with: go build -tags appsmock
//
// At runtime, set LARK_CLI_APPS_MOCK=http://127.0.0.1:PORT to route every
// apps-domain request (/open-apis/spark/...) to a local mock server.
// Non-apps requests are unaffected.

package main

import (
	_ "github.com/larksuite/cli/extension/credential/appsmock" // dummy credential provider
	_ "github.com/larksuite/cli/extension/transport/appsmock"  // URL rewriter
)
