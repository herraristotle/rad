package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/gogf/gf/encoding/gcharset"
	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/js"
	"github.com/herraristotle/rad/pkg/logger"
	model2 "github.com/herraristotle/rad/pkg/model"
)

type Tab struct {
	Ctx              *context.Context
	Cancel           context.CancelFunc
	NavigateReq      model2.Request
	ExtraHeaders     map[string]interface{}
	ResultList       []*model2.Request
	TopFrameId       string
	LoaderID         string
	NavNetworkID     string
	PageCharset      string
	PageBindings     map[string]interface{}
	FoundRedirection bool
	DocBodyNodeId    cdp.NodeID
	config           TabConfig

	lock sync.Mutex

	WG            sync.WaitGroup //wait counter for this tab
	collectLinkWG sync.WaitGroup
	loadedWG      sync.WaitGroup //wait counter for post-Loaded work
	formSubmitWG  sync.WaitGroup //wait counter for form submission
	removeLis     sync.WaitGroup //remove event listeners
	domWG         sync.WaitGroup //wait counter for DOMContentLoaded
	fillFormWG    sync.WaitGroup //form-filling task
}

type TabConfig struct {
	TabRunTimeout           time.Duration
	DomContentLoadedTimeout time.Duration
	EventTriggerMode        string        // how events are triggered: async or sequential
	EventTriggerInterval    time.Duration // interval between event triggers, in milliseconds
	BeforeExitDelay         time.Duration // wait time before exit, to let DOM render and XHR requests fire
	EncodeURLWithCharset    bool
	IgnoreKeywords          []string //
	Proxy                   string
	CustomFormValues        map[string]string
	CustomFormKeywordValues map[string]string
}

type bindingCallPayload struct {
	Name string   `json:"name"`
	Seq  int      `json:"seq"`
	Args []string `json:"args"`
}

func NewTab(browser *Browser, navigateReq model2.Request, config TabConfig) *Tab {
	var tab Tab
	tab.ExtraHeaders = map[string]interface{}{}
	var DOMContentLoadedRun = false
	tab.Ctx, tab.Cancel = browser.NewTab(config.TabRunTimeout)
	for key, value := range browser.ExtraHeaders {
		navigateReq.Headers[key] = value
		if key != "Host" {
			tab.ExtraHeaders[key] = value
		}
	}
	tab.NavigateReq = navigateReq
	tab.config = config
	tab.DocBodyNodeId = 0

	// Register the request interception listener
	chromedp.ListenTarget(*tab.Ctx, func(v interface{}) {
		switch v := v.(type) {
		// Dispatch to the right action per event type
		case *network.EventRequestWillBeSent:
			if string(v.RequestID) == string(v.LoaderID) && v.Type == "Document" && tab.TopFrameId == "" {
				tab.LoaderID = string(v.LoaderID)
				tab.TopFrameId = string(v.FrameID)
			}

		// pause as the request is sent, i.e. request interception
		case *fetch.EventRequestPaused:
			tab.WG.Add(1)
			go tab.InterceptRequest(v)

		// Parse URLs out of every JS file and add them to the results
		// Parse URLs out of the HTML document
		// Detect the current page's encoding
		case *network.EventResponseReceived:
			if v.Response.MimeType == "application/javascript" || v.Response.MimeType == "text/html" || v.Response.MimeType == "application/json" {
				tab.WG.Add(1)
				go tab.ParseResponseURL(v)
			}
			if v.RequestID.String() == tab.NavNetworkID {
				tab.WG.Add(1)
				go tab.GetContentCharset(v)
			}
		// Handle server-side 3XX redirects
		case *network.EventResponseReceivedExtraInfo:
			if v.RequestID.String() == tab.NavNetworkID {
				tab.WG.Add(1)
				go tab.HandleRedirectionResp(v)
			}
		//case *network.EventLoadingFailed:
		//	logger.Logger.Error("EventLoadingFailed ", v.ErrorText)
		// 401 and 407 demand authentication and block the page, so they must be resolved
		case *fetch.EventAuthRequired:
			tab.WG.Add(1)
			go tab.HandleAuthRequired(v)

		// DOMContentLoaded
		// Start form filling and the DOM node observer
		// run only once
		case *page.EventDomContentEventFired:
			if DOMContentLoadedRun {
				return
			}
			DOMContentLoadedRun = true
			tab.WG.Add(1)
			go tab.AfterDOMRun()
		// Loaded
		case *page.EventLoadEventFired:
			if DOMContentLoadedRun {
				return
			}
			DOMContentLoadedRun = true
			tab.WG.Add(1)
			go tab.AfterDOMRun()

		// close Dialog
		case *page.EventJavascriptDialogOpening:
			tab.WG.Add(1)
			go tab.dismissDialog()

		// handle expose function
		case *runtime.EventBindingCalled:
			tab.WG.Add(1)
			go tab.HandleBindingCalled(v)
		}
	})

	return &tab
}

func (tab *Tab) Start() {
	logger.Logger.Info("Crawling " + tab.NavigateReq.Method + " " + tab.NavigateReq.URL.String())
	defer tab.Cancel()
	if err := chromedp.Run(*tab.Ctx,
		RunWithTimeOut(tab.Ctx, tab.config.DomContentLoadedTimeout, chromedp.Tasks{
			//
			runtime.Enable(),
			// Enable the network layer API
			network.Enable(),
			// Enable the request interception API
			fetch.Enable().WithHandleAuthRequests(true),
			// Bind the callback
			// callback used by XSS-Scan
			runtime.AddBinding("addLink"),
			runtime.AddBinding("Test"),
			// Run the initialization JS
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				_, err = page.AddScriptToEvaluateOnNewDocument(js.TabInitJS).Do(ctx)
				if err != nil {
					return err
				}
				return nil
			}),
			network.SetExtraHTTPHeaders(tab.ExtraHeaders),
			// Navigate
			chromedp.Navigate(tab.NavigateReq.URL.String()),
		}),
	); err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Logger.Debug("Crawling Canceled")
			return
		}
		logger.Logger.Warn("navigate timeout ", tab.NavigateReq.URL.String())
	}

	waitDone := func() <-chan struct{} {
		tab.WG.Wait()
		ch := make(chan struct{})
		defer close(ch)
		return ch
	}

	select {
	case <-waitDone():
		logger.Logger.Debug("all navigation tasks done.")
	case <-time.After(tab.config.DomContentLoadedTimeout + time.Second*10):
		logger.Logger.Warn("navigation tasks TIMEOUT.")
	}

	// Wait for all links to be collected
	logger.Logger.Debug("collectLinks start.")
	tab.collectLinkWG.Add(3)
	go tab.collectLinks()
	tab.collectLinkWG.Wait()
	logger.Logger.Debug("collectLinks end.")

	// Detect the page encoding and encode every URL
	if tab.config.EncodeURLWithCharset {
		tab.DetectCharset()
		tab.EncodeAllURLWithCharset()
	}

	//fmt.Println(tab.NavigateReq.URL.String(), len(tab.ResultList))
	//for _, v := range tab.ResultList {
	//	v.SimplePrint()
	//}
	// fmt.Println("Finished " + tab.NavigateReq.Method + " " + tab.NavigateReq.URL.String())
}

func RunWithTimeOut(ctx *context.Context, timeout time.Duration, tasks chromedp.Tasks) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return tasks.Do(timeoutContext)
	}
}

/*
*
Add the collected URLs to the result list; Host binding must be handled here
*/
func (tab *Tab) AddResultUrl(method string, _url string, source string) {
	navUrl := tab.NavigateReq.URL
	url, err := model2.GetUrl(_url, *navUrl)
	if err != nil {
		return
	}
	option := model2.Options{
		Headers:  map[string]interface{}{},
		PostData: "",
	}
	referer := navUrl.String()

	// Apply Host binding
	if host, ok := tab.NavigateReq.Headers["Host"]; ok {
		if host != navUrl.Hostname() && url.Hostname() == host {
			url, _ = model2.GetUrl(strings.Replace(url.String(), "://"+url.Hostname(), "://"+navUrl.Hostname(), -1), *navUrl)
			option.Headers["Host"] = host
			referer = strings.Replace(navUrl.String(), navUrl.Host, host.(string), -1)
		}
	}
	// Add cookies
	if cookie, ok := tab.NavigateReq.Headers["Cookie"]; ok {
		option.Headers["Cookie"] = cookie
	}

	// Fix up the Referer
	option.Headers["Referer"] = referer
	for key, value := range tab.ExtraHeaders {
		option.Headers[key] = value
	}
	req := model2.GetRequest(method, url, option)
	req.Source = source

	tab.lock.Lock()
	tab.ResultList = append(tab.ResultList, &req)
	tab.lock.Unlock()
}

/*
*
Add the request to the result list; Host binding was already handled during interception, so nothing to do here
*/
func (tab *Tab) AddResultRequest(req model2.Request) {
	for key, value := range tab.ExtraHeaders {
		req.Headers[key] = value
	}
	tab.lock.Lock()
	tab.ResultList = append(tab.ResultList, &req)
	tab.lock.Unlock()
}

/*
*
Get the CDP execution context for this tab
*/
func (tab *Tab) GetExecutor() context.Context {
	c := chromedp.FromContext(*tab.Ctx)
	ctx := cdp.WithExecutor(*tab.Ctx, c.Target)
	return ctx
}

/*
*
Close the dialog
*/
func (tab *Tab) dismissDialog() {
	defer tab.WG.Done()
	ctx := tab.GetExecutor()
	_ = page.HandleJavaScriptDialog(false).Do(ctx)
}

/*
*
Handle the callback
*/
func (tab *Tab) HandleBindingCalled(event *runtime.EventBindingCalled) {
	defer tab.WG.Done()
	payload := []byte(event.Payload)
	var bcPayload bindingCallPayload
	_ = json.Unmarshal(payload, &bcPayload)
	if bcPayload.Name == "addLink" && len(bcPayload.Args) > 1 {
		tab.AddResultUrl(config.GET, bcPayload.Args[0], bcPayload.Args[1])
	}
	if bcPayload.Name == "Test" {
		fmt.Println(bcPayload.Args)
	}
	tab.Evaluate(fmt.Sprintf(js.DeliverResultJS, bcPayload.Name, bcPayload.Seq, "s"))
}

/*
*
Execute JS
*/
func (tab *Tab) Evaluate(expression string) {
	ctx := tab.GetExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	_, exception, err := runtime.Evaluate(expression).Do(tCtx)
	if exception != nil {
		logger.Logger.Debug("tab Evaluate: ", exception.Text)
	}
	if err != nil {
		logger.Logger.Debug("tab Evaluate: ", err)
	}
}

/*
*
Fetch Node IDs matching the condition immediately, without waiting
*/
func (tab *Tab) GetNodeIDs(sel string) ([]cdp.NodeID, error) {
	ctx := tab.GetExecutor()
	return dom.QuerySelectorAll(tab.DocBodyNodeId, sel).Do(ctx)
}

/*
*
Execute JS against the given Node
*/
func (tab *Tab) EvaluateWithNode(expression string, node *cdp.Node) error {
	ctx := tab.GetExecutor()
	var res bool
	err := chromedp.EvaluateAsDevTools(js.Snippet(expression, js.CashX(true), "", node), &res).Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

/*
*
Detect the page encoding
*/
func (tab *Tab) DetectCharset() {
	ctx := tab.GetExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancel()
	var content string
	var ok bool
	var getCharsetRegex = regexp.MustCompile("charset=(.+)$")
	err := chromedp.AttributeValue(`meta[http-equiv=Content-Type]`, "content", &content, &ok, chromedp.ByQuery).Do(tCtx)
	if err != nil || !ok {
		return
	}
	if strings.Contains(content, "charset=") {
		charset := getCharsetRegex.FindString(content)
		if charset != "" {
			tab.PageCharset = strings.ToUpper(strings.Replace(charset, "charset=", "", -1))
			tab.PageCharset = strings.TrimSpace(tab.PageCharset)
		}
	}
}

func (tab *Tab) EncodeAllURLWithCharset() {
	if tab.PageCharset == "" || tab.PageCharset == "UTF-8" {
		return
	}
	for _, req := range tab.ResultList {
		newRawQuery, err := gcharset.UTF8To(tab.PageCharset, req.URL.RawQuery)
		if err == nil {
			req.URL.RawQuery = newRawQuery
		}
		newRawPath, err := gcharset.UTF8To(tab.PageCharset, req.URL.RawPath)
		if err == nil {
			req.URL.RawPath = newRawPath
		}
	}
}

func IsIgnoredByKeywordMatch(req model2.Request, IgnoreKeywords []string) bool {
	for _, _str := range IgnoreKeywords {
		if strings.Contains(req.URL.String(), _str) {
			logger.Logger.Info("ignore request: ", req.SimpleFormat())
			return true
		}
	}
	return false
}
