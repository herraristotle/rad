package config

import (
	"time"

	mapset "github.com/deckarep/golang-set"
)

const (
	// DefaultUA is intentionally empty: an empty user agent means rad uses the
	// real UA of the browser it drives instead of claiming to be some fixed
	// Chrome version. Set user_agent in rad_config.yml to override it.
	DefaultUA    = ""
	MaxTabsCount = 10
	// MaxTabsCeiling is the hard upper bound on concurrent tabs. chaitin/rad's
	// config documents max_page_concurrent as "not greater than 10"; values
	// above it are clamped rather than silently over-provisioning browsers.
	MaxTabsCeiling = 10
	// DefaultPageAnalyzeTimeout bounds page analysis when the config omits
	// page_analyze_timeout_second. It mirrors chaitin/rad's default so an
	// absent key behaves the same as the documented value.
	DefaultPageAnalyzeTimeout = 300 * time.Second
	TabRunTimeout             = 10 * time.Second
	DefaultInputText          = "Rad"
	FormInputKeyword          = "Rad"
	SuspectURLRegex           = `(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;|*()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`
	URLRegex                  = `((https?|ftp|file):)?//[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`
	AttrURLRegex              = ``
	DomContentLoadedTimeout   = 10 * time.Second
	EventTriggerInterval      = 100 * time.Millisecond
	BeforeExitDelay           = 1 * time.Second
	DefaultEventTriggerMode   = EventTriggerAsync
	MaxCrawlCount             = 1000
	MaxRunTime                = 60 * 60
)

// Request methods.
const (
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	DELETE  = "DELETE"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
)

// Filter modes.
const (
	SimpleFilterMode = "simple"
	SmartFilterMode  = "smart"
	StrictFilterMode = "strict"
)

// Event trigger modes.
const (
	EventTriggerAsync = "async"
	EventTriggerSync  = "sync"
)

// Where a request came from.
const (
	FromTarget      = "Target"     // initial input target
	FromNavigation  = "Navigation" // page navigation
	FromXHR         = "XHR"        // asynchronous XHR request
	FromDOM         = "DOM"        // extracted from the DOM
	FromJSFile      = "JavaScript" // extracted from JavaScript
	FromFuzz        = "PathFuzz"   // initial path fuzzing
	FromRobots      = "robots.txt" //robots.txt
	FromComment     = "Comment"    // found in page comments
	FromWebSocket   = "WebSocket"
	FromEventSource = "EventSource"
	FromFetch       = "Fetch"
	FromHistoryAPI  = "HistoryAPI"
	FromOpenWindow  = "OpenWindow"
	FromHashChange  = "HashChange"
	FromStaticRes   = "StaticResource"
	FromStaticRegex = "StaticRegex"
)

// content-type
const (
	JSON       = "application/json"
	URLENCODED = "application/x-www-form-urlencoded"
	MULTIPART  = "multipart/form-data"
)

var (
	StaticSuffix = []string{
		"png", "gif", "jpg", "mp4", "mp3", "mng", "pct", "bmp", "jpeg", "pst", "psp", "ttf",
		"tif", "tiff", "ai", "drw", "wma", "ogg", "wav", "ra", "aac", "mid", "au", "aiff",
		"dxf", "eps", "ps", "svg", "3gp", "asf", "asx", "avi", "mov", "mpg", "qt", "rm",
		"wmv", "m4a", "bin", "xls", "xlsx", "ppt", "pptx", "doc", "docx", "odt", "ods", "odg",
		"odp", "exe", "zip", "rar", "tar", "gz", "iso", "rss", "pdf", "txt", "dll", "ico",
		"gz2", "apk", "crt", "woff", "map", "woff2", "webp", "less", "dmg", "bz2", "otf", "swf",
		"flv", "mpeg", "dat", "xsl", "csv", "cab", "exif", "wps", "m4v", "rmvb",
	}
	StaticSuffixSet mapset.Set
)

var (
	ScriptSuffix = []string{
		"php", "asp", "jsp", "asa",
	}
	ScriptSuffixSet mapset.Set
)

var DefaultIgnoreKeywords = []string{"logout", "quit", "exit"}
var AllowedFormName = []string{"default", "mail", "code", "phone", "username", "password", "qq", "id_card", "url", "date", "number"}

type ContinueResourceList []string

var InputTextMap = map[string]map[string]interface{}{
	"mail": {
		"keyword": []string{"mail"},
		"value":   "rad@example.com",
	},
	"code": {
		"keyword": []string{"yanzhengma", "code", "ver", "captcha"},
		"value":   "123a",
	},
	"phone": {
		"keyword": []string{"phone", "number", "tel", "shouji"},
		"value":   "18812345678",
	},
	"username": {
		"keyword": []string{"name", "user", "id", "login", "account"},
		"value":   "rad@example.com",
	},
	"password": {
		"keyword": []string{"pass", "pwd"},
		"value":   "Rad6.",
	},
	"qq": {
		"keyword": []string{"qq", "wechat", "tencent", "weixin"},
		"value":   "123456789",
	},
	"IDCard": {
		"keyword": []string{"card", "shenfen"},
		"value":   "511702197409284963",
	},
	"url": {
		"keyword": []string{"url", "site", "web", "blog", "link"},
		"value":   "https://example.com/",
	},
	"date": {
		"keyword": []string{"date", "time", "year", "now"},
		"value":   "2018-01-01",
	},
	"number": {
		"keyword": []string{"day", "age", "num", "count"},
		"value":   "10",
	},
}

// FormFillSentinelValues are the brand-derived values rad injects when
// auto-filling forms. The smart filter treats these exact values as its
// "this param was filled by us" marker, so they must be compared by exact
// equality: a substring match would also catch real values such as
// "Radisson" or "Gradient".
var FormFillSentinelValues = []string{
	DefaultInputText,
	InputTextMap["password"]["value"].(string),
}

// IsFormFillSentinel reports whether value is one of the values rad itself
// writes into forms.
func IsFormFillSentinel(value string) bool {
	for _, sentinel := range FormFillSentinelValues {
		if value == sentinel {
			return true
		}
	}
	return false
}

func init() {
	StaticSuffixSet = initSet(StaticSuffix)
	ScriptSuffixSet = initSet(ScriptSuffix)
}

func initSet(suffixs []string) mapset.Set {
	set := mapset.NewSet()
	for _, s := range suffixs {
		set.Add(s)
	}
	return set
}
