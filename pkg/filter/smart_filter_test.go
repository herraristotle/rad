package filter

import (
	"testing"

	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/model"

	"github.com/stretchr/testify/assert"
)

var (
	// queryUrls = []string{
	// 	"http://test.nil.local.com/cctv/abcd?keyword=radrad&end=1",
	// 	"http://test.nil.local.com/cctv/abcd?keyword=radrad&end=1",
	// }

	fragmentUrls = []string{
		// Baseline group.
		"http://testhtml5.vuwm.com/latest#/page/1",
		"http://testhtml5.vuwm.com/latest#/page/search?keyword=Rad&source=2&demo=1423&c=afa",
		// Marked as {{long}}.
		"http://testhtml5.vuwm.com/latest#/page/search/fasdfsdafsdfsdfsdfasfsfasfafdsafssfasdfsd",

		// Control group.
		"http://testhtml5.vuwm.com/latest#/page/2",
		// Same shape: only numeric parameters differ, so they must not be marked {{long}}.
		"http://testhtml5.vuwm.com/latest#/page/search?keyword=Rad&source=1&demo=1255&c=afa",
	}

	// completeUrls = []string{
	// 	"https://test.local.com:1234/adfatd/123456/sx14xi?user=rad&pwd=fa1424&end=1#/user/info",
	// }
	smart = NewSmartFilter(NewSimpleFilter(""), true)
)

func TestDoFilter_countFragment(t *testing.T) {
	reqs := []model.Request{}
	for _, fu := range fragmentUrls {
		url, err := model.GetUrl(fu)
		assert.Nil(t, err)
		reqs = append(reqs, model.GetRequest(config.GET, url))
	}
	// #/page/1 and #/page/2 share one shape.
	assert.Equal(t, smart.calcFragmentID(reqs[0].URL.Fragment), smart.calcFragmentID(reqs[3].URL.Fragment))
	assert.Equal(t, smart.calcFragmentID(reqs[1].URL.Fragment), smart.calcFragmentID(reqs[4].URL.Fragment))
	for _, rq := range reqs[:2] {
		// The first occurrence must not be filtered.
		assert.Equal(t, smart.DoFilter(&rq), false)
	}
	for _, rq := range reqs[3:] {
		// The second occurrence of the same shape must be filtered.
		assert.Equal(t, smart.DoFilter(&rq), true)
	}
}

func TestFormFillSentinelIsExact(t *testing.T) {
	// Values that merely contain the brand name must not be mistaken for
	// rad's own form-fill marker (regression: a substring match on "Rad"
	// swallowed real values like "Radisson").
	for _, value := range []string{"Radisson", "Brad", "Gradient", "trademark", "Radware"} {
		f := NewSmartFilter(NewSimpleFilter(""), false)
		url, err := model.GetUrl("http://sentinel.local/list?name=" + value)
		assert.Nil(t, err)
		req := model.GetRequest(config.GET, url)
		f.DoFilter(&req)
		assert.NotEqual(t, CustomValueMark, req.Filter.MarkedQueryMap["name"],
			"value %q must not be marked as a form-fill sentinel", value)
	}

	// The values rad actually injects are still recognised.
	for _, value := range []string{config.DefaultInputText, "Rad6."} {
		f := NewSmartFilter(NewSimpleFilter(""), false)
		url, err := model.GetUrl("http://sentinel.local/list?name=" + value)
		assert.Nil(t, err)
		req := model.GetRequest(config.GET, url)
		f.DoFilter(&req)
		assert.Equal(t, CustomValueMark, req.Filter.MarkedQueryMap["name"],
			"value %q must be marked as a form-fill sentinel", value)
	}
}
