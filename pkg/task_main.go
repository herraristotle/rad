package pkg

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/herraristotle/rad/pkg/config"
	engine2 "github.com/herraristotle/rad/pkg/engine"
	"github.com/herraristotle/rad/pkg/filter"
	filter2 "github.com/herraristotle/rad/pkg/filter"
	"github.com/herraristotle/rad/pkg/logger"
	"github.com/herraristotle/rad/pkg/model"
	"github.com/herraristotle/rad/pkg/rconfig"
	"github.com/herraristotle/rad/pkg/tools"
	"github.com/herraristotle/rad/pkg/tools/requests"

	"github.com/panjf2000/ants/v2"
)

type CrawlerTask struct {
	Browser       *engine2.Browser      //
	RootDomain    string                // root domain of the crawl, used for subdomain collection
	Targets       []*model.Request      // input targets
	Result        *Result               // final result
	Config        *TaskConfig           // task configuration
	filter        filter.FilterHandler  // request filter
	elementFilter *filter.ElementFilter // similar-element filtering
	Pool          *ants.Pool            // worker pool
	taskWG        sync.WaitGroup        // waits for all pool tasks to finish
	crawledCount  int                   // number of pages crawled
	taskCountLock sync.Mutex            // guards crawledCount
	siteVisits    map[string]int        // pages crawled per site
	Start         time.Time             // start time

}

type Result struct {
	ReqList       []*model.Request // results on the same domain
	AllReqList    []*model.Request // requests across every domain
	AllDomainList []string         // every domain seen
	SubDomainList []string         // subdomains seen
	resultLock    sync.Mutex       // guards result merging
}

type tabTask struct {
	crawlerTask *CrawlerTask
	browser     *engine2.Browser
	req         *model.Request
}

/*
*
NewCrawlerTask creates a crawler task.
*/
// normalizeLimits applies the documented bounds so every entry point (CLI
// flags, the YAML config and library callers) behaves the same:
//
//   - max_page_concurrent is documented as "not greater than 10" and is clamped
//     to that ceiling rather than silently over-provisioning browsers.
//   - page_analyze_timeout_second defaults to 300s when omitted, so page
//     analysis is always bounded.
func normalizeLimits(taskConf *TaskConfig) {
	if taskConf.MaxTabsCount > config.MaxTabsCeiling {
		logger.Logger.Warnf("max tab count %d exceeds the supported ceiling, using %d",
			taskConf.MaxTabsCount, config.MaxTabsCeiling)
		taskConf.MaxTabsCount = config.MaxTabsCeiling
	}
	if taskConf.MaxTabsCount <= 0 {
		taskConf.MaxTabsCount = config.MaxTabsCount
	}
	if taskConf.PageAnalyzeTimeout <= 0 {
		taskConf.PageAnalyzeTimeout = config.DefaultPageAnalyzeTimeout
	}
}

func NewCrawlerTask(targets []*model.Request, taskConf TaskConfig) (*CrawlerTask, error) {
	normalizeLimits(&taskConf)

	crawlerTask := CrawlerTask{
		Result:     &Result{},
		Config:     &taskConf,
		siteVisits: map[string]int{},
	}

	baseFilter := filter.NewSimpleFilter(targets[0].URL.Host)

	if taskConf.FilterMode == config.SmartFilterMode {
		crawlerTask.filter = filter.NewSmartFilterWithOptions(baseFilter, false, taskConf.ParentPathDetect)

	} else if taskConf.FilterMode == config.StrictFilterMode {
		crawlerTask.filter = filter.NewSmartFilterWithOptions(baseFilter, true, taskConf.ParentPathDetect)

	} else {
		crawlerTask.filter = baseFilter
	}

	// Similar-element filtering, driven by element_filter_strength and
	// max_similar_request in rad_config.yml.
	if taskConf.ElementFilterStrength > 0 || taskConf.MaxSimilarRequest > 0 {
		strength := taskConf.ElementFilterStrength
		if strength <= 0 {
			strength = 1
		}
		crawlerTask.elementFilter = filter.NewElementFilter(strength)
		crawlerTask.elementFilter.MaxSimilar = taskConf.MaxSimilarRequest
	}

	if len(targets) == 1 {
		_newReq := *targets[0]
		newReq := &_newReq
		_newURL := *_newReq.URL
		newReq.URL = &_newURL
		if targets[0].URL.Scheme == "http" {
			newReq.URL.Scheme = "https"
		} else {
			newReq.URL.Scheme = "http"
		}
		targets = append(targets, newReq)
	}
	crawlerTask.Targets = targets[:]

	for _, req := range targets {
		req.Source = config.FromTarget
	}

	// Keep behaviour and data separate, then initialise the defaults.
	// A functional option and a proxy initialise taskConf.
	for _, fn := range []TaskConfigOptFunc{
		WithTabRunTimeout(config.TabRunTimeout),
		WithMaxTabsCount(config.MaxTabsCount),
		WithMaxCrawlCount(config.MaxCrawlCount),
		WithDomContentLoadedTimeout(config.DomContentLoadedTimeout),
		WithEventTriggerInterval(config.EventTriggerInterval),
		WithBeforeExitDelay(config.BeforeExitDelay),
		WithEventTriggerMode(config.DefaultEventTriggerMode),
		WithIgnoreKeywords(config.DefaultIgnoreKeywords),
	} {
		fn(&taskConf)
	}

	if taskConf.ExtraHeadersString != "" {
		err := json.Unmarshal([]byte(taskConf.ExtraHeadersString), &taskConf.ExtraHeaders)
		if err != nil {
			logger.Logger.Error("custom headers can't be Unmarshal.")
			return nil, err
		}
	}

	if len(taskConf.ChromiumWSUrl) > 0 {
		crawlerTask.Browser = engine2.ConnectBrowser(taskConf.ChromiumWSUrl, taskConf.ExtraHeaders)
	} else {
		crawlerTask.Browser = engine2.InitBrowserWithOptions(engine2.BrowserOptions{
			ChromiumPath: taskConf.ChromiumPath,
			ExtraHeaders: taskConf.ExtraHeaders,
			Proxy:        taskConf.Proxy,
			NoHeadless:   taskConf.NoHeadless,
			EnableImage:  taskConf.EnableImage,
			ForceSandbox: taskConf.ForceSandbox,
		})
	}
	crawlerTask.RootDomain = targets[0].URL.RootDomain()

	// Create the worker pool.
	p, _ := ants.NewPool(taskConf.MaxTabsCount)
	crawlerTask.Pool = p

	return &crawlerTask, nil
}

/*
*
generateTabTask builds a tab task for the given request.
*/
func (t *CrawlerTask) generateTabTask(req *model.Request) *tabTask {
	task := tabTask{
		crawlerTask: t,
		browser:     t.Browser,
		req:         req,
	}
	return &task
}

/*
*
Run starts the task.
*/
func (t *CrawlerTask) Run() {
	defer t.Pool.Release()  // release the worker pool
	defer t.Browser.Close() // close the browser

	t.Start = time.Now()
	if t.Config.PathFromRobots {
		reqsFromRobots := GetPathsFromRobots(*t.Targets[0])
		logger.Logger.Info("get paths from robots.txt: ", len(reqsFromRobots))
		t.Targets = append(t.Targets, reqsFromRobots...)
	}

	if t.Config.FuzzDictPath != "" {
		if t.Config.PathByFuzz {
			logger.Logger.Warn("`--fuzz-path` is ignored, using `--fuzz-path-dict` instead")
		}
		reqsByFuzz := GetPathsByFuzzDict(*t.Targets[0], t.Config.FuzzDictPath)
		t.Targets = append(t.Targets, reqsByFuzz...)
	} else if t.Config.PathByFuzz {
		reqsByFuzz := GetPathsByFuzz(*t.Targets[0])
		logger.Logger.Info("get paths by fuzzing: ", len(reqsByFuzz))
		t.Targets = append(t.Targets, reqsByFuzz...)
	}

	t.Result.AllReqList = t.Targets[:]

	var initTasks []*model.Request
	for _, req := range t.Targets {
		if !t.Config.NewTaskFilter.Match(req) {
			logger.Logger.Debugf("filter req by new_task_filter_config: " + req.URL.RequestURI())
			continue
		}
		if t.filter.DoFilter(req) {
			logger.Logger.Debugf("filter req: " + req.URL.RequestURI())
			continue
		}
		initTasks = append(initTasks, req)
		t.Result.ReqList = append(t.Result.ReqList, req)
	}
	logger.Logger.Info("filter repeat, target count: ", len(initTasks))

	for _, req := range initTasks {
		if t.Config.EntranceRetry > 0 && req.Depth == 0 {
			t.probeEntrance(req)
		}
		if !engine2.IsIgnoredByKeywordMatch(*req, t.Config.IgnoreKeywords) {
			t.addTask2Pool(req)
		}
	}

	t.taskWG.Wait()

	// Deduplicate every collected request.
	todoFilterAll := make([]*model.Request, len(t.Result.AllReqList))
	copy(todoFilterAll, t.Result.AllReqList)

	t.Result.AllReqList = []*model.Request{}
	var simpleFilter filter2.SimpleFilter
	for _, req := range todoFilterAll {
		if !simpleFilter.UniqueFilter(req) {
			t.Result.AllReqList = append(t.Result.AllReqList, req)
		}
	}

	// All domains.
	t.Result.AllDomainList = AllDomainCollect(t.Result.AllReqList)
	// Subdomains.
	t.Result.SubDomainList = SubDomainCollect(t.Result.AllReqList, t.RootDomain)
}

// headersFor resolves the global headers plus any rad_config.yml
// domain_headers entry that matches the request's hostname.
// allowHost enforces the subdomain setting: with subdomain=false only the
// root domain is crawled, so links to subdomains are dropped.
func (t *CrawlerTask) allowHost(req *model.Request) bool {
	if t.Config.Subdomain || req == nil || req.URL == nil {
		return true
	}
	host := req.URL.Hostname()
	root := t.RootDomain
	if root == "" {
		return true
	}
	if host == root || host == "www."+root {
		return true
	}
	logger.Logger.Debugf("filter req because subdomain crawling is disabled: " + req.URL.RequestURI())
	return false
}

func (t *CrawlerTask) headersFor(req *model.Request) map[string]interface{} {
	headers := map[string]interface{}{}
	for key, value := range t.Config.ExtraHeaders {
		headers[key] = value
	}
	if t.Config.UserAgent != "" {
		headers["User-Agent"] = t.Config.UserAgent
	}
	if req == nil || req.URL == nil {
		return headers
	}
	hostname := req.URL.Hostname()
	for _, entry := range t.Config.DomainHeaders {
		if !rconfig.DomainMatch(entry.Domain, hostname) {
			continue
		}
		for key, value := range entry.Headers {
			headers[key] = value
		}
	}
	return headers
}

// allowDepth reports whether req is still within max_depth. Depth 0 means
// unlimited.
func (t *CrawlerTask) allowDepth(req *model.Request) bool {
	if t.Config.MaxDepth <= 0 || req.Depth <= t.Config.MaxDepth {
		return true
	}
	logger.Logger.Debugf("filter req by max_depth: " + req.URL.RequestURI())
	return false
}

/*
*
addTask2Pool adds a request to the worker pool.
Filtering happens immediately before the request is queued.
*/
// probeEntrance performs the entrance_retry attempts for a target, so a
// transient failure on the very first request does not lose the whole crawl.
func (t *CrawlerTask) probeEntrance(req *model.Request) {
	for attempt := 0; attempt < t.Config.EntranceRetry; attempt++ {
		opts := &requests.ReqOptions{
			Timeout:       int(t.Config.TabRunTimeout.Seconds()),
			AllowRedirect: true,
			Proxy:         t.Config.Proxy,
		}
		_, err := requests.Request(req.Method, req.URL.String(),
			tools.ConvertHeaders(t.headersFor(req)), []byte(req.PostData), opts)
		if err == nil {
			return
		}
		logger.Logger.Warn("entrance retry ", attempt+1, "/", t.Config.EntranceRetry, ": ", err)
	}
}

func (t *CrawlerTask) addTask2Pool(req *model.Request) {
	if !t.allowHost(req) {
		return
	}
	// A request that may not be sent must never reach the pool.
	if !t.Config.RequestSendFilter.Match(req) {
		logger.Logger.Debugf("filter req by request_send_filter_config: " + req.URL.RequestURI())
		return
	}
	if !t.allowDepth(req) {
		return
	}

	t.taskCountLock.Lock()
	if t.Config.MaxPageVisitPerSite > 0 {
		site := req.URL.RootDomain()
		if t.siteVisits[site] >= t.Config.MaxPageVisitPerSite {
			t.taskCountLock.Unlock()
			logger.Logger.Debugf("filter req by max_page_visit_per_site: " + req.URL.RequestURI())
			return
		}
		t.siteVisits[site]++
	}
	if t.crawledCount >= t.Config.MaxCrawlCount {
		t.taskCountLock.Unlock()
		return
	} else {
		t.crawledCount += 1
	}

	if t.Start.Add(time.Second * time.Duration(t.Config.MaxRunTime)).Before(time.Now()) {
		t.taskCountLock.Unlock()
		return
	}
	t.taskCountLock.Unlock()

	t.taskWG.Add(1)
	task := t.generateTabTask(req)
	go func() {
		err := t.Pool.Submit(task.Task)
		if err != nil {
			t.taskWG.Done()
			logger.Logger.Error("addTask2Pool ", err)
		}
	}()
}

/*
*
tabTask runs a single tab and implements the work pool interface.
*/
func (t *tabTask) Task() {
	defer t.crawlerTask.taskWG.Done()

	// Set the tab timeout. When a maximum runtime is configured, use whichever of the tab timeout and the remaining runtime is smaller.
	timeremaining := t.crawlerTask.Start.Add(time.Duration(t.crawlerTask.Config.MaxRunTime) * time.Second).Sub(time.Now())
	tabTime := t.crawlerTask.Config.TabRunTimeout
	if t.crawlerTask.Config.TabRunTimeout > timeremaining {
		tabTime = timeremaining
	}

	if tabTime <= 0 {
		return
	}

	tab := engine2.NewTab(t.browser, *t.req, engine2.TabConfig{
		TabRunTimeout:           tabTime,
		DomContentLoadedTimeout: t.crawlerTask.Config.DomContentLoadedTimeout,
		EventTriggerMode:        t.crawlerTask.Config.EventTriggerMode,
		EventTriggerInterval:    t.crawlerTask.Config.EventTriggerInterval,
		BeforeExitDelay:         t.crawlerTask.Config.BeforeExitDelay,
		EncodeURLWithCharset:    t.crawlerTask.Config.EncodeURLWithCharset,
		IgnoreKeywords:          t.crawlerTask.Config.IgnoreKeywords,
		CustomFormValues:        t.crawlerTask.Config.CustomFormValues,
		CustomFormKeywordValues: t.crawlerTask.Config.CustomFormKeywordValues,
		Headers:                 t.crawlerTask.headersFor(t.req),
		MaxInteractive:          t.crawlerTask.Config.MaxInteractive,
		Retry:                   t.crawlerTask.Config.Retry,
		PageAnalyzeTimeout:      t.crawlerTask.Config.PageAnalyzeTimeout,
	})
	tab.Start()

	// The output filter applies to everything rad reports, so drop blocked
	// requests before they reach either result list.
	reportable := make([]*model.Request, 0, len(tab.ResultList))
	for _, req := range tab.ResultList {
		if !t.crawlerTask.Config.RequestOutputFilter.Match(req) {
			logger.Logger.Debugf("filter req by request_output_filter_config: " + req.URL.RequestURI())
			continue
		}
		reportable = append(reportable, req)
	}

	// Collect the result.
	t.crawlerTask.Result.resultLock.Lock()
	t.crawlerTask.Result.AllReqList = append(t.crawlerTask.Result.AllReqList, reportable...)
	t.crawlerTask.Result.resultLock.Unlock()

	for _, req := range reportable {
		if t.crawlerTask.elementFilter != nil && t.crawlerTask.elementFilter.DoFilter(req) {
			logger.Logger.Debugf("filter req by element_filter_strength: " + req.URL.RequestURI())
			continue
		}
		if !t.crawlerTask.filter.DoFilter(req) {
			t.crawlerTask.Result.resultLock.Lock()
			t.crawlerTask.Result.ReqList = append(t.crawlerTask.Result.ReqList, req)
			t.crawlerTask.Result.resultLock.Unlock()
			if !engine2.IsIgnoredByKeywordMatch(*req, t.crawlerTask.Config.IgnoreKeywords) {
				// Discovered requests are one level deeper than the page
				// that produced them.
				req.Depth = t.req.Depth + 1
				t.crawlerTask.addTask2Pool(req)
			}
		}
	}
}
