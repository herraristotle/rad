package rconfig

import (
	"testing"

	"github.com/herraristotle/rad/pkg/model"
	"github.com/stretchr/testify/assert"
)

func req(t *testing.T, raw string) *model.Request {
	t.Helper()
	u, err := model.GetUrl(raw)
	assert.Nil(t, err)
	r := model.GetRequest("GET", u)
	return &r
}

func TestEmptyFilterAllowsEverything(t *testing.T) {
	var cfg FilterConfig
	assert.True(t, cfg.Match(req(t, "http://example.com/a?x=1")))
}

func TestHostnameFilters(t *testing.T) {
	allow := FilterConfig{HostnameAllowed: []string{"*.example.com", "10.0.0.0/24"}}
	assert.True(t, allow.Match(req(t, "http://api.example.com/a")))
	assert.True(t, allow.Match(req(t, "http://10.0.0.7/a")))
	assert.False(t, allow.Match(req(t, "http://other.com/a")))
	assert.False(t, allow.Match(req(t, "http://10.0.1.7/a")))

	deny := FilterConfig{HostnameDisallowed: []string{"*.example.com"}}
	assert.False(t, deny.Match(req(t, "http://api.example.com/a")))
	assert.True(t, deny.Match(req(t, "http://other.com/a")))

	// IP ranges are per octet: a-b.c.d-e, so 1.1-4.1.1-8 covers
	// 1.1.1.1 .. 1.4.1.8.
	rng := FilterConfig{HostnameAllowed: []string{"1.1-4.1.1-8"}}
	assert.True(t, rng.Match(req(t, "http://1.1.1.1/")))
	assert.True(t, rng.Match(req(t, "http://1.4.1.8/")))
	assert.False(t, rng.Match(req(t, "http://4.1.1.8/")))
	assert.False(t, rng.Match(req(t, "http://1.5.1.1/")))
}

func TestPortFilters(t *testing.T) {
	allow := FilterConfig{PortAllowed: []string{"80", "8000-8100"}}
	assert.True(t, allow.Match(req(t, "http://example.com:80/")))
	assert.True(t, allow.Match(req(t, "http://example.com:8050/")))
	assert.False(t, allow.Match(req(t, "http://example.com:9000/")))

	deny := FilterConfig{PortDisallowed: []string{"8080"}}
	assert.False(t, deny.Match(req(t, "http://example.com:8080/")))
	assert.True(t, deny.Match(req(t, "http://example.com:80/")))
}

func TestPathFilters(t *testing.T) {
	allow := FilterConfig{PathAllowed: []string{"test", "*admin*"}}
	assert.True(t, allow.Match(req(t, "http://example.com/a/test/b")))
	assert.True(t, allow.Match(req(t, "http://example.com/x/admin/y")))
	assert.False(t, allow.Match(req(t, "http://example.com/a/b/c")))

	deny := FilterConfig{PathDisallowed: []string{"*logout*"}}
	assert.False(t, deny.Match(req(t, "http://example.com/user/logout")))
	assert.True(t, deny.Match(req(t, "http://example.com/user/info")))
}

func TestQueryAndFragmentFilters(t *testing.T) {
	deny := FilterConfig{QueryKeyDisallowed: []string{"token", "*session*"}}
	assert.False(t, deny.Match(req(t, "http://example.com/a?token=1")))
	assert.False(t, deny.Match(req(t, "http://example.com/a?my_session_id=1")))
	assert.True(t, deny.Match(req(t, "http://example.com/a?id=1")))

	allow := FilterConfig{QueryKeyAllowed: []string{"id"}}
	assert.True(t, allow.Match(req(t, "http://example.com/a?id=1")))
	assert.False(t, allow.Match(req(t, "http://example.com/a?other=1")))

	frag := FilterConfig{FragmentDisallowed: []string{"*debug*"}}
	assert.False(t, frag.Match(req(t, "http://example.com/a#debug")))
	assert.True(t, frag.Match(req(t, "http://example.com/a#main")))
}
