package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/js"
	"github.com/herraristotle/rad/pkg/logger"
	"github.com/herraristotle/rad/pkg/tools"
)

/**
Executes JS against a DOM node.
err := EvaluateAsDevTools(snippet(submitJS, cashX(true), sel, nodes[0]), &res).Do(ctx)

See chromedp's submit implementation for the same pattern.
*/

/*
*
Runs after the page has loaded.
Also waits for afterDOMRun to complete.
*/
func (tab *Tab) AfterLoadedRun() {
	defer tab.WG.Done()
	logger.Logger.Debug("afterLoadedRun start")
	tab.formSubmitWG.Add(2)
	tab.loadedWG.Add(3)
	tab.removeLis.Add(1)

	go tab.formSubmit()
	tab.formSubmitWG.Wait()
	logger.Logger.Debug("formSubmit end")

	if tab.config.EventTriggerMode == config.EventTriggerAsync {
		go tab.triggerJavascriptProtocol()
		go tab.triggerInlineEvents()
		go tab.triggerDom2Events()
		tab.waitForAnalysis()
	} else if tab.config.EventTriggerMode == config.EventTriggerSync {
		tab.triggerInlineEvents()
		time.Sleep(tab.config.EventTriggerInterval)
		tab.triggerDom2Events()
		time.Sleep(tab.config.EventTriggerInterval)
		tab.triggerJavascriptProtocol()
	}

	// After triggering an event, wait briefly so the browser can send the XHR and update the DOM.
	time.Sleep(tab.config.BeforeExitDelay)

	go tab.RemoveDOMListener()
	tab.removeLis.Wait()
	logger.Logger.Debug("afterLoadedRun end")
}

// waitForAnalysis waits for the async event triggers, but gives up after
// page_analyze_timeout_second so one pathological page cannot stall the crawl.
func (tab *Tab) waitForAnalysis() {
	if tab.config.PageAnalyzeTimeout <= 0 {
		tab.loadedWG.Wait()
		return
	}
	done := make(chan struct{})
	go func() {
		tab.loadedWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(tab.config.PageAnalyzeTimeout):
		logger.Logger.Warn("page analysis timeout after ", tab.config.PageAnalyzeTimeout, ", continuing: ", tab.NavigateReq.URL.String())
	}
}

/*
*
formSubmit submits forms automatically.
*/
func (tab *Tab) formSubmit() {

	logger.Logger.Debug("formSubmit start")

	// First point the form at a target.
	tab.setFormToFrame()

	// Then try three ways of submitting the form.
	go tab.clickSubmit()
	go tab.clickAllButton()
}

/*
*
setFormTarget points the form at a frame.
*/
func (tab *Tab) setFormToFrame() {
	// Create the frame first.
	nameStr := tools.RandSeq(8)
	tab.Evaluate(fmt.Sprintf(js.NewFrameTemplate, nameStr, nameStr))

	// Then point every form node at it.
	ctx := tab.GetExecutor()
	formNodes, formErr := tab.GetNodeIDs(`form`)
	if formErr != nil || len(formNodes) == 0 {
		logger.Logger.Debug("setFormToFrame: get form element err")
		if formErr != nil {
			logger.Logger.Debug(formErr)
		}
		return
	}
	tCtx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	_ = chromedp.SetAttributeValue(formNodes, "target", nameStr, chromedp.ByNodeID).Do(tCtx)
}

/*
*
clickSubmitButton clicks a button with type=submit.
*/
func (tab *Tab) clickSubmit() {
	defer tab.formSubmitWG.Done()

	// First click the button with type=submit.
	ctx := tab.GetExecutor()

	// Otherwise get every form node and submit it directly.
	formNodes, formErr := tab.GetNodeIDs(`form`)
	if formErr != nil || len(formNodes) == 0 {
		logger.Logger.Debug("clickSubmit: get form element err")
		if formErr != nil {
			logger.Logger.Debug(formErr)
		}
		return
	}
	tCtx1, cancel1 := context.WithTimeout(ctx, time.Second*2)
	defer cancel1()
	_ = chromedp.Submit(formNodes, chromedp.ByNodeID).Do(tCtx1)

	// Get every input element.
	inputNodes, inputErr := tab.GetNodeIDs(`form input[type=submit]`)
	if inputErr != nil || len(inputNodes) == 0 {
		logger.Logger.Debug("clickSubmit: get form input element err")
		if inputErr != nil {
			logger.Logger.Debug(inputErr)
		}
		return
	}
	tCtx2, cancel2 := context.WithTimeout(ctx, time.Second*2)
	defer cancel2()
	_ = chromedp.Click(inputNodes, chromedp.ByNodeID).Do(tCtx2)
}

/*
*
click all button
*/
func (tab *Tab) clickAllButton() {
	defer tab.formSubmitWG.Done()

	// Get button nodes inside forms.
	ctx := tab.GetExecutor()
	// Get every button element.
	btnNodeIDs, bErr := tab.GetNodeIDs(`form button`)
	if bErr != nil || len(btnNodeIDs) == 0 {
		logger.Logger.Debug("clickAllButton: get form button element err")
		if bErr != nil {
			logger.Logger.Debug(bErr)
		}
		return
	}
	tCtx, cancel1 := context.WithTimeout(ctx, time.Second*2)
	defer cancel1()
	_ = chromedp.Click(btnNodeIDs, chromedp.ByNodeID).Do(tCtx)

	// Click using JS.
	var btnNodes []*cdp.Node
	tCtx2, cancel2 := context.WithTimeout(ctx, time.Second*2)
	defer cancel2()
	err := chromedp.Nodes(btnNodeIDs, &btnNodes, chromedp.ByNodeID).Do(tCtx2)
	if err != nil {
		return
	}
	for _, node := range btnNodes {
		_ = tab.EvaluateWithNode(js.FormNodeClickJS, node)
	}
}

/*
*
triggerInlineEvent fires an inline event.
*/
func (tab *Tab) triggerInlineEvents() {
	defer tab.loadedWG.Done()
	logger.Logger.Debug("triggerInlineEvents start")
	tab.Evaluate(fmt.Sprintf(js.TriggerInlineEventJS, tab.config.EventTriggerInterval.Seconds()*1000))
	logger.Logger.Debug("triggerInlineEvents end")
}

/*
*
triggerDOM2Event fires a DOM level 2 event.
*/
func (tab *Tab) triggerDom2Events() {
	defer tab.loadedWG.Done()
	logger.Logger.Debug("triggerDom2Events start")
	tab.Evaluate(fmt.Sprintf(js.TriggerDom2EventJS, tab.config.EventTriggerInterval.Seconds()*1000))
	logger.Logger.Debug("triggerDom2Events end")
}

/*
*
Returns true when an anchor's href uses a pseudo-protocol,
*/
func (tab *Tab) triggerJavascriptProtocol() {
	defer tab.loadedWG.Done()
	logger.Logger.Debug("clickATagJavascriptProtocol start")
	tab.Evaluate(fmt.Sprintf(js.TriggerJavascriptProtocol, tab.config.EventTriggerInterval.Seconds()*1000,
		tab.config.EventTriggerInterval.Seconds()*1000))
	logger.Logger.Debug("clickATagJavascriptProtocol end")
}

/*
*
RemoveDOMListener stops watching DOM mutations.
*/
func (tab *Tab) RemoveDOMListener() {
	defer tab.removeLis.Done()
	logger.Logger.Debug("RemoveDOMListener start")
	// Remove the DOM mutation listener.
	tab.Evaluate(js.RemoveDOMListenerJS)
	logger.Logger.Debug("RemoveDOMListener end")
}
