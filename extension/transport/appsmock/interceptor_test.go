// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build appsmock

package appsmock

import (
	"net/http"
	"testing"
)

func TestIsAppsDomain(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Positives — every URL hard-coded under shortcuts/apps/*.go must match.
		{"/open-apis/spark/v1/apps", true},
		{"/open-apis/spark/v1/apps/app_x/access-scope", true},
		{"/open-apis/spark/v1/apps/app_x/release", true},
		{"/open-apis/spark/v1/apps/app_x/sessions/conv_x/chat", true},
		{"/open-apis/spark/v1/apps/app_x/db_dev_init", true},
		// Negatives — other domains must pass through untouched.
		{"/open-apis/contact/v3/users/get", false},
		{"/open-apis/drive/v1/files", false},
		{"/open-apis/im/v1/messages", false},
		{"/open-apis/application/v6/applications/foo", false}, // app meta, NOT apps domain
		{"/", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsAppsDomain(tc.path); got != tc.want {
			t.Errorf("IsAppsDomain(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestInterceptor_PreRoundTrip_AppsRewrite(t *testing.T) {
	i := &Interceptor{mockScheme: "http", mockHost: "127.0.0.1:7878"}
	req, _ := http.NewRequest("GET", "https://open.feishu.cn/open-apis/spark/v1/apps?page_size=20", nil)
	post := i.PreRoundTrip(req)
	if post != nil {
		t.Fatalf("PreRoundTrip should return nil post hook, got non-nil")
	}
	if req.URL.Scheme != "http" || req.URL.Host != "127.0.0.1:7878" {
		t.Errorf("URL not rewritten: scheme=%q host=%q", req.URL.Scheme, req.URL.Host)
	}
	if req.URL.Path != "/open-apis/spark/v1/apps" {
		t.Errorf("path mangled: %q", req.URL.Path)
	}
	if req.URL.RawQuery != "page_size=20" {
		t.Errorf("query mangled: %q", req.URL.RawQuery)
	}
	if req.Host != "" {
		t.Errorf("req.Host should be cleared, got %q", req.Host)
	}
	if got := req.Header.Get("X-Lark-Cli-Appsmock-Origin"); got != "open.feishu.cn" {
		t.Errorf("audit header = %q, want %q", got, "open.feishu.cn")
	}
}

func TestInterceptor_PreRoundTrip_NonAppsPassthrough(t *testing.T) {
	i := &Interceptor{mockScheme: "http", mockHost: "127.0.0.1:7878"}
	req, _ := http.NewRequest("GET", "https://open.feishu.cn/open-apis/contact/v3/users/u_x", nil)
	if post := i.PreRoundTrip(req); post != nil {
		t.Fatalf("PreRoundTrip should return nil post hook for non-apps, got non-nil")
	}
	// Non-apps requests must NOT have URL rewritten.
	if req.URL.Host != "open.feishu.cn" {
		t.Errorf("non-apps URL was rewritten: host=%q", req.URL.Host)
	}
	if req.URL.Scheme != "https" {
		t.Errorf("non-apps scheme was rewritten: %q", req.URL.Scheme)
	}
	if req.Header.Get("X-Lark-Cli-Appsmock-Origin") != "" {
		t.Errorf("audit header should be absent for non-apps requests")
	}
}
