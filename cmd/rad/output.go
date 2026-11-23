package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/herraristotle/rad/pkg"
	model2 "github.com/herraristotle/rad/pkg/model"
)

// fullTextSeparator sits between request dumps in --full-text-output. It
// matches chaitin/rad: the dump ends with "\r\n\r\n", then a bare "\n", the
// rule, and another "\n".
const fullTextSeparator = "\n--------------------------------------------------\n"

// chaitinEntry is one record of --json-output, matching chaitin/rad's schema:
// the body is base64 encoded in b64_body.
type chaitinEntry struct {
	Method  string            `json:"Method"`
	URL     string            `json:"URL"`
	Header  map[string]string `json:"Header,omitempty"`
	B64Body string            `json:"b64_body,omitempty"`
}

// resolveOutputPath applies --auto-index: when the target exists, append -1,
// -2, ... until an unused name is found.
func resolveOutputPath(path string, autoIndex bool) string {
	if !autoIndex {
		return path
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// writeTextOutput writes "METHOD URL" lines, one per reportable request.
func writeTextOutput(path string, reqs []*model2.Request, autoIndex bool) error {
	var b strings.Builder
	for _, req := range reqs {
		b.WriteString(req.Method)
		b.WriteString(" ")
		b.WriteString(req.URL.String())
		b.WriteString("\n")
	}
	return os.WriteFile(resolveOutputPath(path, autoIndex), []byte(b.String()), 0o644)
}

// writeFullTextOutput writes a raw HTTP request dump per request, separated by
// the same rule chaitin/rad uses.
func writeFullTextOutput(path string, reqs []*model2.Request, autoIndex bool) error {
	dumps := make([]string, 0, len(reqs))
	for _, req := range reqs {
		dumps = append(dumps, buildRawRequest(req))
	}
	return os.WriteFile(resolveOutputPath(path, autoIndex), []byte(strings.Join(dumps, fullTextSeparator)), 0o644)
}

// buildRawRequest renders a request as a raw HTTP/1.1 dump.
func buildRawRequest(req *model2.Request) string {
	var b strings.Builder
	uri := req.URL.RequestURI()
	if uri == "" {
		uri = "/"
	}
	fmt.Fprintf(&b, "%s %s HTTP/1.1\r\n", req.Method, uri)
	hasHost := false
	for key, value := range req.Headers {
		if strings.EqualFold(key, "Host") {
			hasHost = true
		}
		fmt.Fprintf(&b, "%s: %s\r\n", key, value)
	}
	if !hasHost {
		fmt.Fprintf(&b, "Host: %s\r\n", req.URL.Host)
	}
	b.WriteString("\r\n")
	b.WriteString(req.PostData)
	return b.String()
}

// writeJSONOutput writes the chaitin/rad JSON array schema, body base64 encoded.
func writeJSONOutput(path string, reqs []*model2.Request, autoIndex bool) error {
	entries := make([]chaitinEntry, 0, len(reqs))
	for _, req := range reqs {
		entry := chaitinEntry{Method: req.Method, URL: req.URL.String()}
		if len(req.Headers) > 0 {
			entry.Header = make(map[string]string, len(req.Headers))
			for key, value := range req.Headers {
				entry.Header[key] = fmt.Sprintf("%v", value)
			}
		}
		if req.PostData != "" {
			entry.B64Body = base64.StdEncoding.EncodeToString([]byte(req.PostData))
		}
		entries = append(entries, entry)
	}
	// Match chaitin/rad's layout: "[\n" then one compact object per line.
	var b strings.Builder
	b.WriteString("[\n")
	for i, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		b.Write(data)
		if i < len(entries)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("]\n")
	return os.WriteFile(resolveOutputPath(path, autoIndex), []byte(b.String()), 0o644)
}

// applyOutputs writes every requested output file. All three receive the same
// reportable request set (req_list).
func applyOutputs(result *pkg.Result) error {
	if textOutputPath != "" {
		if err := writeTextOutput(textOutputPath, result.ReqList, autoIndex); err != nil {
			return err
		}
	}
	if fullTextOutputPath != "" {
		if err := writeFullTextOutput(fullTextOutputPath, result.ReqList, autoIndex); err != nil {
			return err
		}
	}
	if jsonOutputPath != "" {
		if err := writeJSONOutput(jsonOutputPath, result.ReqList, autoIndex); err != nil {
			return err
		}
	}
	return nil
}
