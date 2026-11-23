package main

import (
	"github.com/herraristotle/rad/pkg/config"
	"github.com/urfave/cli/v2"
)

var cliFlags = []cli.Flag{
	SetTarget(),
	SetUrlFile(),
	SetWaitLogin(),
	SetConfigFile(),
	SetChromePath(),
	SetChromeWSUrl(),
	SetCustomHeaders(),
	SetPostData(),
	SetMaxCrawledCount(),
	SetFilterMod(),
	SetOutputMode(),
	SetOutputJSON(),
	SetTextOutput(),
	SetFullTextOutput(),
	SetJSONOutputChaitin(),
	SetAutoIndex(),
	SetNoBanner(),
	//SetIgcognitoContext(),
	SetMaxTabCount(),
	SetFuzzPath(),
	SetFuzzPathDict(),
	SetRobotsPath(),
	SetRequestProxy(),
	SetEncodeURL(),
	SetTabRunTTL(),
	SetWaitDomContentLoadedTTL(),
	SetEventTriggerMode(),
	SetEventTriggerInterval(),
	SetBeforeExitDelay(),
	SetIgnoreUrlKeywords(),
	SetFormValues(),
	SetFormKeywordValue(),
	SetPushToProxy(),
	SetPushPoolMax(),
	SetLogLevel(),
	SetNoHeadless(),
	SetMaxTime(),
}

func SetChromePath() *cli.PathFlag {
	return &cli.PathFlag{
		Name:        "chromium-path",
		Aliases:     []string{"chrome"},
		Usage:       "`Path` of chromium executable. Such as \"/home/test/chrome-linux/chrome\"",
		Destination: &taskConfig.ChromiumPath,
		EnvVars:     []string{"RAD_CHROMIUM_PATH"},
	}
}

func SetChromeWSUrl() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "chrome-ws-url",
		Aliases:     []string{"w"},
		Usage:       "`URL` of chromium websockets debugger. Please note that any chromium flags specified will not apply when using this option. Such as \"ws://127.0.0.1:9222/devtools/browser/94b947d7-2a08-4cee-8e21-d77055e8c465\"",
		Destination: &taskConfig.ChromiumWSUrl,
		EnvVars:     []string{"RAD_CHROME_WS_URL"},
	}
}

func SetCustomHeaders() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "custom-headers",
		Usage:       "add additional `Headers` to each request. The input string will be called json.Unmarshal",
		Value:       `{"Spider-Name": "rad"}`,
		Destination: &taskConfig.ExtraHeadersString,
	}
}

func SetPostData() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "post-data",
		Aliases:     []string{"d"},
		Usage:       "set `PostData` to target and use POST method.",
		Destination: &postData,
	}
}

func SetMaxCrawledCount() *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "max-crawled-count",
		Aliases:     []string{"m"},
		Value:       config.MaxCrawlCount,
		Usage:       "the maximum `Number` of URLs visited by the crawler in this task.",
		Destination: &taskConfig.MaxCrawlCount,
	}
}

func SetFilterMod() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "filter-mode",
		Aliases:     []string{"f"},
		Value:       "smart",
		Usage:       "filtering `Mode` used for collected requests. Allowed mode:\"simple\", \"smart\" or \"strict\".",
		Destination: &taskConfig.FilterMode,
	}
}

func SetOutputMode() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "output-mode",
		Aliases:     []string{"o"},
		Value:       "console",
		Usage:       "console print or serialize output. Allowed mode:\"console\" ,\"json\" or \"none\".",
		Destination: &outputMode,
	}
}

func SetOutputJSON() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "output-json",
		Usage:       "write output to a json file.Such as result_www_test_com.json",
		Destination: &outputJsonPath,
	}
}

func SetMaxTabCount() *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "max-tab-count",
		Aliases:     []string{"tab"},
		Value:       8,
		Usage:       "maximum `Number` of tabs allowed.",
		Destination: &taskConfig.MaxTabsCount,
	}
}

func SetFuzzPath() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "fuzz-path",
		Value:       false,
		Usage:       "whether to fuzz the target with common paths.",
		Destination: &taskConfig.PathByFuzz,
	}
}

func SetFuzzPathDict() *cli.PathFlag {
	return &cli.PathFlag{
		Name:        "fuzz-path-dict",
		Usage:       "`Path` of fuzz dict. Such as \"/home/test/fuzz_path.txt\"",
		Destination: &taskConfig.FuzzDictPath,
	}
}

func SetRobotsPath() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "robots-path",
		Value:       false,
		Usage:       "whether to resolve paths from /robots.txt.",
		Destination: &taskConfig.PathFromRobots,
	}
}

func SetRequestProxy() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "request-proxy",
		Aliases:     []string{"http-proxy"},
		Usage:       "set upstream `proxy`; all requests connect through it.",
		Destination: &taskConfig.Proxy,
	}
}

//	return &cli.BoolFlag{
//		Name:        "bypass",
//		Value:       false,
//		Usage:       "whether to encode url with detected charset.",
//		Destination: &taskConfig.EncodeURLWithCharset,
//	},
func SetEncodeURL() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "encode-url",
		Value:       false,
		Usage:       "whether to encode url with detected charset.",
		Destination: &taskConfig.EncodeURLWithCharset,
	}
}

func SetTabRunTTL() *cli.DurationFlag {

	return &cli.DurationFlag{
		Name:        "tab-run-timeout",
		Value:       config.TabRunTimeout,
		Usage:       "the `Timeout` of a single tab task.",
		Destination: &taskConfig.TabRunTimeout,
	}
}

func SetWaitDomContentLoadedTTL() *cli.DurationFlag {
	return &cli.DurationFlag{
		Name:        "wait-dom-content-loaded-timeout",
		Value:       config.DomContentLoadedTimeout,
		Usage:       "the `Timeout` of waiting for a page dom ready.",
		Destination: &taskConfig.DomContentLoadedTimeout,
	}
}

func SetEventTriggerMode() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "event-trigger-mode",
		Value:       config.EventTriggerAsync,
		Usage:       "this `Value` determines how the crawler automatically triggers events.Allowed mode:\"async\" or \"sync\".",
		Destination: &taskConfig.EventTriggerMode,
	}
}

func SetEventTriggerInterval() *cli.DurationFlag {
	return &cli.DurationFlag{
		Name:        "event-trigger-interval",
		Value:       config.EventTriggerInterval,
		Usage:       "the `Interval` of triggering each event.",
		Destination: &taskConfig.EventTriggerInterval,
	}
}

func SetBeforeExitDelay() *cli.DurationFlag {
	return &cli.DurationFlag{
		Name:        "before-exit-delay",
		Value:       config.BeforeExitDelay,
		Usage:       "the `Time` of waiting before crawler exit.",
		Destination: &taskConfig.BeforeExitDelay,
	}
}

func SetIgnoreUrlKeywords() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:        "ignore-url-keywords",
		Aliases:     []string{"iuk"},
		Value:       cli.NewStringSlice(config.DefaultIgnoreKeywords...),
		Usage:       "rad will not crawl these URLs matched by `Keywords`. e.g.: -iuk logout -iuk quit -iuk exit",
		DefaultText: "Default [logout quit exit]",
		Destination: ignoreKeywords,
	}
}

func SetFormValues() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:        "form-values",
		Aliases:     []string{"fv"},
		Usage:       "custom filling text for each form type. e.g.: -fv username=rad_nice -fv password=admin123",
		Destination: customFormTypeValues,
	}
}

// 根据关键词自行选择填充文本
func SetFormKeywordValue() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:        "form-keyword-values",
		Aliases:     []string{"fkv"},
		Usage:       "custom filling text, fuzzy matched by keyword. e.g.: -fkv user=rad_nice -fkv pass=admin123",
		Destination: customFormKeywordValues,
	}
}

func SetPushToProxy() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "push-to-proxy",
		Usage:       "every request in 'req_list' will be pushed to the proxy `Address`. Such as \"http://127.0.0.1:8080/\"",
		Destination: &pushAddress,
	}
}

func SetPushPoolMax() *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "push-pool-max",
		Usage:       "maximum `Number` of concurrency when pushing results to proxy.",
		Value:       DefaultMaxPushProxyPoolMax,
		Destination: &pushProxyPoolMax,
	}
}

func SetLogLevel() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "log-level",
		Usage:       "log print `Level`, options include debug, info, warn, error and fatal.",
		Value:       DefaultLogLevel,
		Destination: &logLevel,
	}
}

func SetNoHeadless() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "no-headless",
		Aliases:     []string{"disable-headless", "dh"},
		Value:       false,
		Usage:       "disable headless mode",
		Destination: &taskConfig.NoHeadless,
	}
}

func SetMaxTime() *cli.Int64Flag {
	return &cli.Int64Flag{
		Name:        "max-run-time",
		Usage:       "the `Timeout` of the task.",
		Value:       config.MaxRunTime,
		Destination: &taskConfig.MaxRunTime,
	}
}

func SetConfigFile() *cli.PathFlag {
	return &cli.PathFlag{
		Name:        "config",
		Aliases:     []string{"c"},
		Usage:       "`Path` of the YAML config file. If it does not exist, a default rad_config.yml is written and rad exits so you can review it.",
		Destination: &configFile,
	}
}

// SetTarget accepts the target via a flag as chaitin/rad does (-t/--target/
// --url), in addition to positional arguments.
func SetTarget() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:        "target",
		Aliases:     []string{"t", "url", "u"},
		Usage:       "the `target` that will be crawled",
		Destination: &targets,
	}
}

// SetUrlFile reads targets from a file, one URL per line.
func SetUrlFile() *cli.PathFlag {
	return &cli.PathFlag{
		Name:        "url-file",
		Aliases:     []string{"uf"},
		Usage:       "the url-file that will be crawled, one URL per line",
		Destination: &urlFile,
	}
}

// SetWaitLogin disables headless mode and waits for the user to log in before
// crawling, matching chaitin/rad's -wait-login.
func SetWaitLogin() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "wait-login",
		Usage:       "open a browser for manual login, then press Enter to start crawling (implies --no-headless)",
		Destination: &waitLogin,
	}
}

func SetTextOutput() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "text-output",
		Aliases:     []string{"text"},
		Usage:       "output urls to the specified text `file`",
		Destination: &textOutputPath,
	}
}

func SetFullTextOutput() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "full-text-output",
		Aliases:     []string{"full"},
		Usage:       "output full request dump to the specified text `file`",
		Destination: &fullTextOutputPath,
	}
}

func SetJSONOutputChaitin() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "json-output",
		Aliases:     []string{"json"},
		Usage:       "output full request dump to the specified json `file`, body encoded with base64",
		Destination: &jsonOutputPath,
	}
}

func SetAutoIndex() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "auto-index",
		Aliases:     []string{"index"},
		Usage:       "if the output file exists, automatically add a sequence number to the file name",
		Destination: &autoIndex,
	}
}

func SetNoBanner() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "no-banner",
		Usage:       "disable banner print",
		Destination: &noBanner,
	}
}
