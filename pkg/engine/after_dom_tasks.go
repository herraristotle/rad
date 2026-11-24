package engine

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/js"
	"github.com/herraristotle/rad/pkg/logger"
)

/*
*
Runs after DOMContentLoaded.
*/
func (tab *Tab) AfterDOMRun() {
	defer tab.WG.Done()

	logger.Logger.Debug("afterDOMRun start")

	// Get the body node ID so child nodes can be looked up later.
	if !tab.getBodyNodeId() {
		logger.Logger.Debug("no body document NodeID, exit.")
		return
	}

	tab.domWG.Add(2)
	go tab.fillForm()
	go tab.setObserverJS()
	tab.domWG.Wait()
	logger.Logger.Debug("afterDOMRun end")
	tab.WG.Add(1)
	go tab.AfterLoadedRun()
}

/*
*
getBodyNodeID returns the body node ID for non-blocking child lookups.
Wait at most 3 seconds; give up if the DOM is still not rendered.
*/
func (tab *Tab) getBodyNodeId() bool {
	var docNodeIDs []cdp.NodeID
	ctx := tab.GetExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	// Get the frame document root.
	err := chromedp.NodeIDs(`body`, &docNodeIDs, chromedp.ByQuery).Do(tCtx)
	if len(docNodeIDs) == 0 || err != nil {
		// not root node yet?
		logger.Logger.Debug("getBodyNodeId failed, maybe DOM not ready?")
		if err != nil {
			logger.Logger.Debug(err)
		}
		return false
	}
	tab.DocBodyNodeId = docNodeIDs[0]
	return true
}

/*
*
fillForm fills forms automatically.
*/
func (tab *Tab) fillForm() {
	defer tab.domWG.Done()
	logger.Logger.Debug("fillForm start")
	tab.fillFormWG.Add(3)
	f := FillForm{
		tab: tab,
	}

	go f.fillInput()
	go f.fillMultiSelect()
	go f.fillTextarea()

	tab.fillFormWG.Wait()
	logger.Logger.Debug("fillForm end")
}

/*
*
setObserverJS installs a DOM mutation observer.
*/
func (tab *Tab) setObserverJS() {
	defer tab.domWG.Done()
	logger.Logger.Debug("setObserverJS start")
	// Install a DOM mutation observer.
	go tab.Evaluate(js.ObserverJS)
	logger.Logger.Debug("setObserverJS end")
}

type FillForm struct {
	tab *Tab
}

/*
*
fillInput fills every input element.
*/
func (f *FillForm) fillInput() {
	defer f.tab.fillFormWG.Done()
	var nodes []*cdp.Node
	ctx := f.tab.GetExecutor()

	tCtx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	// Check whether any input exists first, to exit early and avoid waiting.
	inputNodes, inputErr := f.tab.GetNodeIDs(`input`)
	if inputErr != nil || len(inputNodes) == 0 {
		logger.Logger.Debug("fillInput: get form input element err")
		if inputErr != nil {
			logger.Logger.Debug(inputErr)
		}
		return
	}
	// Get every input element.
	err := chromedp.Nodes(`input`, &nodes, chromedp.ByQueryAll).Do(tCtx)

	if err != nil {
		logger.Logger.Debug("get all input element err")
		logger.Logger.Debug(err)
		return
	}

	// Pick inputs with an empty type or type=text.
	for _, node := range nodes {
		// Fallback timeout.
		tCtxN, cancelN := context.WithTimeout(ctx, time.Second*5)
		attrType := node.AttributeValue("type")
		if attrType == "text" || attrType == "" {
			inputName := node.AttributeValue("id") + node.AttributeValue("class") + node.AttributeValue("name")
			value := f.GetMatchInputText(inputName)
			var nodeIds = []cdp.NodeID{node.NodeID}
			// Try simulated input first.
			_ = chromedp.SendKeys(nodeIds, value, chromedp.ByNodeID).Do(tCtxN)
			// Then assign the JS property directly.
			_ = chromedp.SetAttributeValue(nodeIds, "value", value, chromedp.ByNodeID).Do(tCtxN)
		} else if attrType == "email" || attrType == "password" || attrType == "tel" {
			value := f.GetMatchInputText(attrType)
			var nodeIds = []cdp.NodeID{node.NodeID}
			// Try simulated input first.
			_ = chromedp.SendKeys(nodeIds, value, chromedp.ByNodeID).Do(tCtxN)
			// Then assign the JS property directly.
			_ = chromedp.SetAttributeValue(nodeIds, "value", value, chromedp.ByNodeID).Do(tCtxN)
		} else if attrType == "radio" || attrType == "checkbox" {
			var nodeIds = []cdp.NodeID{node.NodeID}
			_ = chromedp.SetAttributeValue(nodeIds, "checked", "true", chromedp.ByNodeID).Do(tCtxN)
		} else if attrType == "file" || attrType == "image" {
			var nodeIds = []cdp.NodeID{node.NodeID}
			wd, _ := os.Getwd()
			filePath := wd + "/upload/image.png"
			_ = chromedp.RemoveAttribute(nodeIds, "accept", chromedp.ByNodeID).Do(tCtxN)
			_ = chromedp.RemoveAttribute(nodeIds, "required", chromedp.ByNodeID).Do(tCtxN)
			_ = chromedp.SendKeys(nodeIds, filePath, chromedp.ByNodeID).Do(tCtxN)
		}
		cancelN()
	}
}

func (f *FillForm) fillTextarea() {
	defer f.tab.fillFormWG.Done()
	ctx := f.tab.GetExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	value := f.GetMatchInputText("other")

	textareaNodes, textareaErr := f.tab.GetNodeIDs(`textarea`)
	if textareaErr != nil || len(textareaNodes) == 0 {
		logger.Logger.Debug("fillTextarea: get textarea element err")
		if textareaErr != nil {
			logger.Logger.Debug(textareaErr)
		}
		return
	}

	_ = chromedp.SendKeys(textareaNodes, value, chromedp.ByNodeID).Do(tCtx)
}

func (f *FillForm) fillMultiSelect() {
	defer f.tab.fillFormWG.Done()
	ctx := f.tab.GetExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	optionNodes, optionErr := f.tab.GetNodeIDs(`select option:first-child`)
	if optionErr != nil || len(optionNodes) == 0 {
		logger.Logger.Debug("fillMultiSelect: get select option element err")
		if optionErr != nil {
			logger.Logger.Debug(optionErr)
		}
		return
	}
	_ = chromedp.SetAttributeValue(optionNodes, "selected", "true", chromedp.ByNodeID).Do(tCtx)
	_ = chromedp.SetJavascriptAttribute(optionNodes, "selected", "true", chromedp.ByNodeID).Do(tCtx)
}

func (f *FillForm) GetMatchInputText(name string) string {
	// Fuzzy-match when a custom keyword is set.
	for key, value := range f.tab.config.CustomFormKeywordValues {
		if strings.Contains(name, key) {
			return value
		}
	}

	name = strings.ToLower(name)
	for key, item := range config.InputTextMap {
		for _, keyword := range item["keyword"].([]string) {
			if strings.Contains(name, keyword) {
				if customValue, ok := f.tab.config.CustomFormValues[key]; ok {
					return customValue
				} else {
					return item["value"].(string)
				}
			}
		}
	}
	return f.tab.config.CustomFormValues["default"]
}
