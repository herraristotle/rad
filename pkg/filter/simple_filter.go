package filter

import (
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/model"
)

type SimpleFilter struct {
	UniqueSet       mapset.Set
	HostLimit       string
	staticSuffixSet mapset.Set
}

func NewSimpleFilter(host string) *SimpleFilter {
	staticSuffixSet := config.StaticSuffixSet.Clone()

	for _, suffix := range []string{"js", "css", "json"} {
		staticSuffixSet.Add(suffix)
	}
	s := &SimpleFilter{UniqueSet: mapset.NewSet(), staticSuffixSet: staticSuffixSet, HostLimit: host}
	return s
}

/*
*
DoFilter reports whether the request should be filtered out.
*/
func (s *SimpleFilter) DoFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	// First check whether the domain should be filtered.
	if s.HostLimit != "" && s.DomainFilter(req) {
		return true
	}
	// Deduplicate.
	if s.UniqueFilter(req) {
		return true
	}
	// Filter out static assets.
	if s.StaticFilter(req) {
		return true
	}
	return false
}

/*
*
UniqueFilter removes duplicate requests.
*/
func (s *SimpleFilter) UniqueFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if s.UniqueSet.Contains(req.UniqueId()) {
		return true
	} else {
		s.UniqueSet.Add(req.UniqueId())
		return false
	}
}

/*
*
StaticResourceFilter drops static assets.
*/
func (s *SimpleFilter) StaticFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	// Convert the slice to a map first.

	if req.URL.FileExt() == "" {
		return false
	}
	if s.staticSuffixSet.Contains(req.URL.FileExt()) {
		return true
	}
	return false
}

/*
*
OnlyDomainFilter keeps only links on the given domain.
*/
func (s *SimpleFilter) DomainFilter(req *model.Request) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if req.URL.Host == s.HostLimit || req.URL.Hostname() == s.HostLimit {
		return false
	}
	if strings.HasSuffix(s.HostLimit, ":80") && req.URL.Port() == "" && req.URL.Scheme == "http" {
		if req.URL.Hostname()+":80" == s.HostLimit {
			return false
		}
	}
	if strings.HasSuffix(s.HostLimit, ":443") && req.URL.Port() == "" && req.URL.Scheme == "https" {
		if req.URL.Hostname()+":443" == s.HostLimit {
			return false
		}
	}
	return true
}
