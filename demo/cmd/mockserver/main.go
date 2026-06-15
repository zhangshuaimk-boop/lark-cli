// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// mockserver is a tiny HTTP server that mimics the apps-domain endpoints
// of the Lark open platform. It is intentionally unauthenticated — it
// accepts any Authorization header (or none) and returns canned JSON
// responses. Used by the appsmock build-tag demo to exercise the
// interceptor end-to-end without touching the real gateway.
//
// Usage:
//
//	go run ./demo/cmd/mockserver -addr 127.0.0.1:7878
//
// Every received request is logged to stderr (one JSON line per request)
// so the e2e harness can assert "this URL really hit the mock".
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7878", "address to listen on")
	logFile := flag.String("log", "", "if non-empty, append per-request JSON log lines to this file (in addition to stderr)")
	flag.Parse()

	var logSink io.Writer = os.Stderr
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("open log file: %v", err)
		}
		defer f.Close()
		logSink = io.MultiWriter(os.Stderr, f)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", makeHandler(logSink))

	fmt.Fprintf(os.Stderr, "appsmock listening on http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func makeHandler(logSink io.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		entry := map[string]interface{}{
			"method":         r.Method,
			"path":           r.URL.Path,
			"raw_query":      r.URL.RawQuery,
			"host":           r.Host,
			"authz_present":  r.Header.Get("Authorization") != "",
			"appsmock_orig":  r.Header.Get("X-Lark-Cli-Appsmock-Origin"),
			"cli_shortcut":   r.Header.Get("X-Cli-Shortcut"),
			"user_agent":     r.Header.Get("User-Agent"),
			"body_len_bytes": len(body),
		}
		line, _ := json.Marshal(entry)
		fmt.Fprintln(logSink, string(line))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Lark-Request-Id", "appsmock-demo-reqid")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseFor(r.URL.Path, body))
	}
}

// responseFor returns a canned Lark-flavored OpenAPI response for a given
// path. Shape mirrors what the real apps OpenAPI emits — `{code, msg, data}`
// — so lark-cli's response parser is happy.
func responseFor(path string, _ []byte) []byte {
	switch {
	case strings.HasPrefix(path, "/open-apis/spark/v1/apps") && strings.HasSuffix(path, "/apps"):
		// AppsList GET /open-apis/spark/v1/apps
		return mustJSON(map[string]interface{}{
			"code": 0,
			"msg":  "success",
			"data": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"app_id":       "mock_app_aaa",
						"name":         "Mocked App Alpha",
						"description":  "Returned by the appsmock demo server.",
						"is_published": true,
						"online_url":   "https://example.invalid/alpha",
						"updated_at":   1718438400,
						"created_at":   1718000000,
						"icon_url":     "https://example.invalid/icon.png",
					},
					{
						"app_id":       "mock_app_bbb",
						"name":         "Mocked App Beta",
						"description":  "Second canned item.",
						"is_published": false,
						"updated_at":   1718524800,
						"created_at":   1718100000,
					},
				},
				"has_more":   false,
				"page_token": "",
			},
		})
	default:
		// Generic OK envelope for unmatched paths — lets ad-hoc smoke tests work.
		return mustJSON(map[string]interface{}{
			"code": 0,
			"msg":  "appsmock generic ok",
			"data": map[string]interface{}{"mocked_path": path},
		})
	}
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
