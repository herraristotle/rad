package pkg

import (
	"time"

	"github.com/herraristotle/rad/pkg/rconfig"
)

type TaskConfig struct {
	MaxCrawlCount           int    // maximum number of pages to crawl
	FilterMode              string // simple、smart、strict
	ExtraHeaders            map[string]interface{}
	ExtraHeadersString      string
	AllDomainReturn         bool // collect all domains
	SubDomainReturn         bool // collect subdomains
	NoHeadless              bool // disable headless mode
	DomContentLoadedTimeout time.Duration
	TabRunTimeout           time.Duration     // per-tab timeout
	PathByFuzz              bool              // fuzz paths using the built-in wordlist
	FuzzDictPath            string            // path to a directory wordlist for fuzzing
	PathFromRobots          bool              // derive paths from robots.txt
	MaxTabsCount            int               // maximum tabs, i.e. concurrent crawls
	ChromiumPath            string            // path to the Chromium executable
	ChromiumWSUrl           string            // Websocket debugging URL for a running chrome session
	EventTriggerMode        string            // how events are triggered: async or sequential
	EventTriggerInterval    time.Duration     // interval between event triggers
	BeforeExitDelay         time.Duration     // grace period before exit, letting the DOM render and XHR fire
	EncodeURLWithCharset    bool              // encode URLs using the detected charset
	IgnoreKeywords          []string          // keywords that stop a URL from being scanned or requested
	Proxy                   string            // request proxy
	CustomFormValues        map[string]string // custom form fill values
	CustomFormKeywordValues map[string]string // custom form fill values keyed by keyword
	MaxRunTime              int64             // maximum crawl time in seconds; the task ends gracefully once in-flight requests finish

	// Loaded from rad_config.yml when -config is supplied.
	ConfigFile            string                 // path of the config file, empty when unused
	UserAgent             string                 // request UA; empty keeps the browser's real UA
	DomainHeaders         []rconfig.DomainHeader // per-domain extra headers
	MaxDepth              int                    // maximum page depth, 0 means unlimited
	MaxInteractive        int                    // maximum interactions per page, 0 means unlimited
	MaxInteractiveDepth   int                    // interaction depth limit, 0 means unlimited
	MaxPageVisitPerSite   int                    // pages allowed per site, 0 means unlimited
	EnableImage           bool                   // load images in the browser
	ForceSandbox          bool                   // force the chrome sandbox on
	ParentPathDetect      bool                   // enable parent-directory pseudo-static detection
	ElementFilterStrength int                    // 0 disables similar-element filtering, 1-7 increases it
	PageAnalyzeTimeout    time.Duration          // per-page analysis budget
	MaxSimilarRequest     int                    // max similar fetch/XHR requests per site, <=0 unlimited
	EntranceRetry         int                    // retries for the entrance URL
	Subdomain             bool                   // crawl subdomains automatically
	Retry                 int                    // retries after a failed visit
	NewTaskFilter         rconfig.FilterConfig   // may this link enter the crawl queue
	RequestSendFilter     rconfig.FilterConfig   // may this request be sent
	RequestOutputFilter   rconfig.FilterConfig   // may this request be reported
}

type TaskConfigOptFunc func(*TaskConfig)

func NewTaskConfig(optFuncs ...TaskConfigOptFunc) *TaskConfig {
	conf := &TaskConfig{}
	for _, fn := range optFuncs {
		fn(conf)
	}
	return conf
}

func WithMaxCrawlCount(maxCrawlCount int) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.MaxCrawlCount == 0 {
			tc.MaxCrawlCount = maxCrawlCount
		}
	}
}

func WithFilterMode(gen string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.FilterMode == "" {
			tc.FilterMode = gen
		}
	}
}

func WithExtraHeaders(gen map[string]interface{}) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.ExtraHeaders == nil {
			tc.ExtraHeaders = gen
		}
	}
}

func WithExtraHeadersString(gen string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.ExtraHeadersString == "" {
			tc.ExtraHeadersString = gen
		}
	}
}

func WithAllDomainReturn(gen bool) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if !tc.AllDomainReturn {
			tc.AllDomainReturn = gen
		}
	}
}
func WithSubDomainReturn(gen bool) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if !tc.SubDomainReturn {
			tc.SubDomainReturn = gen
		}
	}
}

func WithNoHeadless(gen bool) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if !tc.NoHeadless {
			tc.NoHeadless = gen
		}
	}
}

func WithDomContentLoadedTimeout(gen time.Duration) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.DomContentLoadedTimeout == 0 {
			tc.DomContentLoadedTimeout = gen
		}
	}
}

func WithTabRunTimeout(gen time.Duration) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.TabRunTimeout == 0 {
			tc.TabRunTimeout = gen
		}
	}
}
func WithPathByFuzz(gen bool) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if !tc.PathByFuzz {
			tc.PathByFuzz = gen
		}
	}
}
func WithFuzzDictPath(gen string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.FuzzDictPath == "" {
			tc.FuzzDictPath = gen
		}
	}
}
func WithPathFromRobots(gen bool) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if !tc.PathFromRobots {
			tc.PathFromRobots = gen
		}
	}
}
func WithMaxTabsCount(gen int) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.MaxTabsCount == 0 {
			tc.MaxTabsCount = gen
		}
	}
}
func WithChromiumPath(gen string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.ChromiumPath == "" {
			tc.ChromiumPath = gen
		}
	}
}
func WithEventTriggerMode(gen string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.EventTriggerMode == "" {
			tc.EventTriggerMode = gen
		}
	}
}
func WithEventTriggerInterval(gen time.Duration) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.EventTriggerInterval == 0 {
			tc.EventTriggerInterval = gen
		}
	}
}
func WithBeforeExitDelay(gen time.Duration) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.BeforeExitDelay == 0 {
			tc.BeforeExitDelay = gen
		}
	}
}
func WithEncodeURLWithCharset(gen bool) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if !tc.EncodeURLWithCharset {
			tc.EncodeURLWithCharset = gen
		}
	}
}
func WithIgnoreKeywords(gen []string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.IgnoreKeywords == nil || len(tc.IgnoreKeywords) == 0 {
			tc.IgnoreKeywords = gen
		}
	}
}
func WithProxy(gen string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.Proxy == "" {
			tc.Proxy = gen
		}
	}
}
func WithCustomFormValues(gen map[string]string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.CustomFormValues == nil || len(tc.CustomFormValues) == 0 {
			tc.CustomFormValues = gen
		}
	}
}
func WithCustomFormKeywordValues(gen map[string]string) TaskConfigOptFunc {
	return func(tc *TaskConfig) {
		if tc.CustomFormKeywordValues == nil || len(tc.CustomFormKeywordValues) == 0 {
			tc.CustomFormKeywordValues = gen
		}
	}
}
