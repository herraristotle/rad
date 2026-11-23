package filter

import (
	"testing"

	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/model"
	"github.com/stretchr/testify/assert"
)

func mkReq(t *testing.T, raw string) *model.Request {
	t.Helper()
	u, err := model.GetUrl(raw)
	assert.Nil(t, err)
	req := model.GetRequest(config.GET, u)
	return &req
}

func TestElementFilterDisabledAtZero(t *testing.T) {
	f := NewElementFilter(0)
	a := mkReq(t, "http://x.com/a/1")
	b := mkReq(t, "http://x.com/a/1")
	assert.False(t, f.DoFilter(a))
	assert.False(t, f.DoFilter(b), "strength 0 must not filter anything")
}

func TestElementFilterCollapsesNumericIDs(t *testing.T) {
	f := NewElementFilter(1)
	assert.False(t, f.DoFilter(mkReq(t, "http://x.com/user/123")))
	assert.True(t, f.DoFilter(mkReq(t, "http://x.com/user/456")),
		"numeric path segments should be treated as the same element")
	assert.False(t, f.DoFilter(mkReq(t, "http://x.com/user/profile")),
		"a non-numeric segment is a different element")
}

func TestElementFilterStrengthTiers(t *testing.T) {
	// At strength 1 a long id-like segment is distinct from a numeric one.
	low := NewElementFilter(1)
	assert.False(t, low.DoFilter(mkReq(t, "http://x.com/item/abcdef1234567890")))
	assert.False(t, low.DoFilter(mkReq(t, "http://x.com/item/zzzzzz9999999999")),
		"strength 1 should not collapse arbitrary long ids")

	// At strength 3 long id-like segments collapse together.
	mid := NewElementFilter(3)
	assert.False(t, mid.DoFilter(mkReq(t, "http://x.com/item/abcdef1234567890")))
	assert.True(t, mid.DoFilter(mkReq(t, "http://x.com/item/zzzzzz9999999999")),
		"strength 3 should collapse id-like segments")
}

func TestElementFilterQueryValues(t *testing.T) {
	// Strength 1 ignores query values entirely.
	f := NewElementFilter(1)
	assert.False(t, f.DoFilter(mkReq(t, "http://x.com/list?page=1")))
	assert.True(t, f.DoFilter(mkReq(t, "http://x.com/list?page=2")))

	// A different query key is still a different element.
	g := NewElementFilter(1)
	assert.False(t, g.DoFilter(mkReq(t, "http://x.com/list?page=1")))
	assert.False(t, g.DoFilter(mkReq(t, "http://x.com/list?sort=asc")))
}

func TestElementFilterValueClassAtHigherStrength(t *testing.T) {
	// At strength 4 numeric and string values are different classes.
	f := NewElementFilter(4)
	assert.False(t, f.DoFilter(mkReq(t, "http://x.com/s?q=123")))
	assert.True(t, f.DoFilter(mkReq(t, "http://x.com/s?q=999")), "both numeric -> same class")
	assert.False(t, f.DoFilter(mkReq(t, "http://x.com/s?q=abc")), "string is a different class")
}

func TestElementFilterIsPerHost(t *testing.T) {
	f := NewElementFilter(1)
	assert.False(t, f.DoFilter(mkReq(t, "http://a.com/user/1")))
	assert.False(t, f.DoFilter(mkReq(t, "http://b.com/user/1")),
		"the same shape on a different host is a different element")
}
