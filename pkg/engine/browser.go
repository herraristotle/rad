package engine

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/herraristotle/rad/pkg/logger"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type Browser struct {
	Ctx          *context.Context
	Cancel       *context.CancelFunc
	tabs         []*context.Context
	tabCancels   []context.CancelFunc
	ExtraHeaders map[string]interface{}
	// profileDir is the temporary Chrome profile owned by this browser, so it
	// can be removed once Chrome has fully exited.
	profileDir string
	lock       sync.Mutex
}

// BrowserOptions carries the browser-level settings resolved from the CLI and
// the YAML config.
type BrowserOptions struct {
	ChromiumPath string
	ExtraHeaders map[string]interface{}
	Proxy        string
	NoHeadless   bool
	EnableImage  bool
	ForceSandbox bool
}

func InitBrowser(chromiumPath string, extraHeaders map[string]interface{}, proxy string, noHeadless bool) *Browser {
	return InitBrowserWithOptions(BrowserOptions{
		ChromiumPath: chromiumPath,
		ExtraHeaders: extraHeaders,
		Proxy:        proxy,
		NoHeadless:   noHeadless,
	})
}

func InitBrowserWithOptions(o BrowserOptions) *Browser {
	var bro Browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],

		// 无头模式
		chromedp.Flag("headless", !o.NoHeadless),
		// https://github.com/chromedp/chromedp/issues/997#issuecomment-1030596050
		// incognito mode not used
		// 禁用GPU，不显示GUI
		chromedp.Flag("disable-gpu", true),
		// Sandbox: keep it off by default (required in most containers), but
		// honour force_sandbox from the config.
		chromedp.Flag("no-sandbox", !o.ForceSandbox),
		// 忽略证书错误
		chromedp.Flag("ignore-certificate-errors", true),

		chromedp.Flag("disable-images", !o.EnableImage),
		//
		chromedp.Flag("disable-web-security", true),
		//
		chromedp.Flag("disable-xss-auditor", true),
		//
		chromedp.Flag("disable-setuid-sandbox", true),

		chromedp.Flag("allow-running-insecure-content", true),

		chromedp.Flag("disable-webgl", true),

		chromedp.Flag("disable-popup-blocking", true),

		chromedp.WindowSize(1920, 1080),
	)
	// 设置浏览器代理
	if o.Proxy != "" {
		opts = append(opts, chromedp.ProxyServer(o.Proxy))
	}

	if len(o.ChromiumPath) > 0 {

		// 指定执行路径
		opts = append(opts, chromedp.ExecPath(o.ChromiumPath))
	}

	// Own the profile directory so cleanup is explicit and predictable.
	if dir, err := os.MkdirTemp("", "rad-chrome-"); err == nil {
		bro.profileDir = dir
		opts = append(opts, chromedp.UserDataDir(dir))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	bctx, _ := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	// https://github.com/chromedp/chromedp/issues/824#issuecomment-845664441
	// 如果需要在一个浏览器上创建多个tab，则需要先创建浏览器的上下文，即运行下面的语句
	err := chromedp.Run(bctx)
	if err != nil {
		// not found chrome process need exit
		logger.Logger.Fatal("chromedp run error: ", err.Error())
	}
	bro.Cancel = &cancel
	bro.Ctx = &bctx
	bro.ExtraHeaders = o.ExtraHeaders
	return &bro
}

func ConnectBrowser(wsUrl string, extraHeaders map[string]interface{}) *Browser {
	var bro Browser
	allocCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), wsUrl)
	bctx, _ := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(log.Printf),
	)

	err := chromedp.Run(bctx)
	if err != nil {
		// couldn't connect to the remote browser, need to exit
		logger.Logger.Fatal("chromedp run error: ", err.Error())
	}
	bro.Cancel = &cancel
	bro.Ctx = &bctx
	bro.ExtraHeaders = extraHeaders

	return &bro
}

func (bro *Browser) NewTab(timeout time.Duration) (*context.Context, context.CancelFunc) {
	bro.lock.Lock()
	defer bro.lock.Unlock()

	ctx, cancel := chromedp.NewContext(*bro.Ctx)
	tCtx, cancelTimeout := context.WithTimeout(ctx, timeout)
	bro.tabs = append(bro.tabs, &tCtx)
	// Release the timeout context and then the chromedp context, so neither
	// is leaked for the lifetime of the browser. Safe to call more than once.
	cancelTab := func() {
		cancelTimeout()
		cancel()
	}
	bro.tabCancels = append(bro.tabCancels, cancelTab)

	return &tCtx, cancelTab
}

func (bro *Browser) Close() {
	logger.Logger.Info("closing browser.")
	for _, cancel := range bro.tabCancels {
		cancel()
	}

	for _, ctx := range bro.tabs {
		err := browser.Close().Do(*ctx)
		if err != nil {
			logger.Logger.Debug(err)
		}
	}

	err := browser.Close().Do(*bro.Ctx)
	if err != nil {
		logger.Logger.Debug(err)
	}

	(*bro.Cancel)()

	bro.removeProfileDir()
}

// removeProfileDir deletes the temporary Chrome profile owned by this browser.
//
// Chrome's helper processes can recreate files fractionally after RemoveAll
// returns, so a single successful removal is not proof the directory is gone.
// The removal is retried until the directory stays absent across two checks, or
// until a short deadline passes.
func (bro *Browser) removeProfileDir() {
	if bro.profileDir == "" {
		return
	}
	dir := bro.profileDir
	bro.profileDir = ""

	deadline := time.Now().Add(3 * time.Second)
	for {
		err := os.RemoveAll(dir)
		if err == nil {
			if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
				time.Sleep(150 * time.Millisecond)
				if _, again := os.Stat(dir); os.IsNotExist(again) {
					logger.Logger.Debug("removed chrome profile dir: ", dir)
					return
				}
			}
		}
		if time.Now().After(deadline) {
			logger.Logger.Warn("could not remove chrome profile dir ", dir,
				"; it will be left for the system to clean up")
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
