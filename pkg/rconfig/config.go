// Package rconfig loads rad's YAML configuration file and applies it to a
// crawler TaskConfig.
//
// A config file is optional: rad works exactly as before when none is given.
// When -config points at a missing file, rad writes a fully commented default
// file there and exits, so the file doubles as documentation.
package rconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultFileName is the config file rad looks for when -config is omitted.
const DefaultFileName = "rad_config.yml"

// DomainHeader sets request headers for the domains matched by Domain.
// Domain uses glob syntax, e.g. "*", "*.example.com", "example.com".
type DomainHeader struct {
	Domain  string            `yaml:"domain"`
	Headers map[string]string `yaml:"headers"`
}

// FilterConfig decides whether a URL/request passes a given stage. Each list
// is a set of glob patterns; empty lists mean "no restriction". A request is
// allowed only if it matches every populated allow list and no disallow list.
type FilterConfig struct {
	HostnameAllowed    []string `yaml:"hostname_allowed"`
	HostnameDisallowed []string `yaml:"hostname_disallowed"`
	PortAllowed        []string `yaml:"port_allowed"`
	PortDisallowed     []string `yaml:"port_disallowed"`
	PathAllowed        []string `yaml:"path_allowed"`
	PathDisallowed     []string `yaml:"path_disallowed"`
	QueryKeyAllowed    []string `yaml:"query_key_allowed"`
	QueryKeyDisallowed []string `yaml:"query_key_disallowed"`
	FragmentAllowed    []string `yaml:"fragment_allowed"`
	FragmentDisallowed []string `yaml:"fragment_disallowed"`
	PostKeyAllowed     []string `yaml:"post_key_allowed"`
	PostKeyDisallowed  []string `yaml:"post_key_disallowed"`
}

// Config mirrors chaitin/rad's rad_config.yml so pipelines can be ported over.
type Config struct {
	ExecPath         string `yaml:"exec_path"`
	DisableHeadless  bool   `yaml:"disable_headless"`
	Subdomain        bool   `yaml:"subdomain"`
	ForceSandbox     bool   `yaml:"force_sandbox"`
	EnableImage      bool   `yaml:"enable_image"`
	ParentPathDetect bool   `yaml:"parent_path_detect"`
	Proxy            string `yaml:"proxy"`
	UserAgent        string `yaml:"user_agent"`

	DomainHeaders []DomainHeader `yaml:"domain_headers"`

	MaxDepth                 int `yaml:"max_depth"`
	NavigateTimeoutSecond    int `yaml:"navigate_timeout_second"`
	LoadTimeoutSecond        int `yaml:"load_timeout_second"`
	Retry                    int `yaml:"retry"`
	PageAnalyzeTimeoutSecond int `yaml:"page_analyze_timeout_second"`
	MaxInteractive           int `yaml:"max_interactive"`
	MaxInteractiveDepth      int `yaml:"max_interactive_depth"`
	MaxPageConcurrent        int `yaml:"max_page_concurrent"`
	MaxPageVisit             int `yaml:"max_page_visit"`
	MaxPageVisitPerSite      int `yaml:"max_page_visit_per_site"`
	ElementFilterStrength    int `yaml:"element_filter_strength"`

	EntranceRetry     int `yaml:"entrance_retry"`
	MaxSimilarRequest int `yaml:"max_similar_request"`

	NewTaskFilterConfig       FilterConfig `yaml:"new_task_filter_config"`
	RequestSendFilterConfig   FilterConfig `yaml:"request_send_filter_config"`
	RequestOutputFilterConfig FilterConfig `yaml:"request_output_filter_config"`
}

// Load reads path. If the file does not exist, it writes the annotated default
// config there and returns ErrCreated so the caller can tell the user to review
// it before running.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if writeErr := WriteDefault(path); writeErr != nil {
			return nil, writeErr
		}
		return nil, ErrCreated
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// ErrCreated is returned by Load when it generated a default config file.
var ErrCreated = fmt.Errorf("default config file created")

// WriteDefault writes the annotated default config to path.
func WriteDefault(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(DefaultYAML), 0o644)
}

// FindDefault returns DefaultFileName in the working directory if it exists.
func FindDefault() string {
	if _, err := os.Stat(DefaultFileName); err == nil {
		return DefaultFileName
	}
	return ""
}
