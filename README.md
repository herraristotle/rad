# rad

![chromedp](https://img.shields.io/badge/chromedp-v0.5.2-brightgreen.svg) [![BlackHat EU Arsenal](https://img.shields.io/badge/BlackHat%20Europe-2021%20Arsenal-blue.svg)](https://www.blackhat.com/eu-21/arsenal/schedule/index.html#crawlergo-a-powerful-browser-crawler-for-web-vulnerability-scanners-25113)

> A powerful browser crawler for web vulnerability scanners

> **rad** is a renamed fork of [crawlergo](https://github.com/Qianlitp/crawlergo), originally created by Qianlitp ([@9ian1i](https://github.com/Qianlitp)). Upstream attribution and the GPLv3 license are retained.

rad is a browser crawler that uses `chrome headless` mode for URL collection. It hooks key positions of the whole web page with DOM rendering stage, automatically fills and submits forms, with intelligent JS event triggering, and collects as many entries exposed by the website as possible. The built-in URL de-duplication module filters out a large number of pseudo-static URLs, still maintains a fast parsing and crawling speed for large websites, and finally gets a high-quality collection of request results.

rad currently supports the following features:
* chrome browser environment rendering
* Intelligent form filling, automated submission
* Full DOM event collection with automated triggering
* Smart URL de-duplication to remove most duplicate requests
* Intelligent analysis of web pages and collection of URLs, including javascript file content, page comments, robots.txt files and automatic Fuzz of common paths
* Support Host binding, automatically fix and add Referer
* Support browser request proxy
* Support pushing the results to passive web vulnerability scanners

## Screenshot

![](./imgs/demo.gif)

## Installation

**Please read and accept the [disclaimer](./Disclaimer.md) before installing or using rad.**

**Build**

- compilation for current platform

```shell
make build
```

- compile for all platforms
```shell
make build_all
```

1. rad relies only on the chrome environment to run, go to [download](https://www.chromium.org/getting-involved/download-chromium) for the new version of chromium.
2. Build rad with `make build`; the binary lands in `./bin/rad`. Copy it to any directory. If you are on linux or macOS, please give rad **executable permissions (+x)**.
3. Prebuilt binaries are published on this repository's [releases page](https://github.com/herraristotle/rad/releases). A release is created automatically when a `v*` tag is pushed. Assets are named `rad_<tag>_<os>_<arch>.tar.gz`, or `.zip` on Windows; extract one and the binary inside is `rad` (`rad.exe` on Windows). A `checksums.txt` with SHA-256 hashes is attached to each release — verify your download with `sha256sum -c checksums.txt`. You can also build from source as above.

> If you are using a linux system and chrome prompts you with missing dependencies, please see TroubleShooting below

## Quick Start
### Go！

Assuming your chromium installation directory is `/tmp/chromium/`, set up 10 tabs open at the same time and crawl the `testphp.vulnweb.com`:

```shell
bin/rad --chrome /tmp/chromium/chrome --tab 10 -t http://testphp.vulnweb.com/
```

### Docker usage

You can also use this with docker without headache: 

```shell
# clone this repository, then:
docker build . -t rad
docker run rad http://testphp.vulnweb.com/
```

### Using Proxy

```shell
bin/rad --chrome /tmp/chromium/chrome --tab 10 --request-proxy socks5://127.0.0.1:7891 -t http://testphp.vulnweb.com/
```

### Calling rad with python

By default, rad prints the results directly on the screen. We next set the output mode to `json`, and the sample code for calling it using python is as follows:

```python
#!/usr/bin/python3
# coding: utf-8

import simplejson
import subprocess

def main():
    target = "http://testphp.vulnweb.com/"
    cmd = ["bin/rad", "--chrome", "/tmp/chromium/chrome", "-o", "json", "-t", target]
    rsp = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output, error = rsp.communicate()
    # "--[Mission Complete]--" separates the report from anything printed before it
    result = simplejson.loads(output.decode().split("--[Mission Complete]--")[1])
    req_list = result["req_list"]
    print(req_list[0])

if __name__ == '__main__':
    main()
```

### Crawl Results

When the output mode is set to `json`, the returned result, after JSON deserialization, contains four parts:

* `all_req_list`： All requests found during this crawl task, containing any resource type from other domains.
* `req_list`：Returns the **current domain results** of this crawl task, pseudo-statically de-duplicated, without static resource links. It is a subset of `all_req_list `.
* `all_domain_list`：List of all domains found.
* `sub_domain_list`：List of subdomains found.

## Examples

rad returns the full request and URL, which can be used in a variety of ways:

* Used in conjunction with other passive web vulnerability scanners

  First, start a passive scanner and set the listening address to: `http://127.0.0.1:1234/`

  Next, assuming rad is on the same machine as the scanner, start rad and set the parameters:

  `--push-to-proxy http://127.0.0.1:1234/`

* Host binding (not available for high version chrome)  [(example)](./examples/host_binding.py)

* Custom Cookies  [(example)](./examples/request_with_cookie.py)

* Regularly clean up zombie processes generated by rad [(example)](./examples/zombie_clean.py) , contributed by @ring04h

## Bypass headless detect
rad can bypass headless mode detection by default.

https://intoli.com/blog/not-possible-to-block-chrome-headless/chrome-headless-test.html

![](./imgs/bypass.png)

## TroubleShooting

* 'Fetch.enable' wasn't found

  Fetch is a feature supported by the new version of chrome, if this error occurs, it means your version is too low, please upgrade the chrome version.
  
* chrome runs with missing dependencies such as xxx.so

  ```shell
  // Ubuntu
  apt-get install -yq --no-install-recommends \
       libasound2 libatk1.0-0 libc6 libcairo2 libcups2 libdbus-1-3 \
       libexpat1 libfontconfig1 libgcc1 libgconf-2-4 libgdk-pixbuf2.0-0 libglib2.0-0 libgtk-3-0 libnspr4 \
       libpango-1.0-0 libpangocairo-1.0-0 libstdc++6 libx11-6 libx11-xcb1 libxcb1 libgbm1 \
       libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxrandr2 libxrender1 libxss1 libxtst6 libnss3
       
  // CentOS 7
  sudo yum install pango.x86_64 libXcomposite.x86_64 libXcursor.x86_64 libXdamage.x86_64 libXext.x86_64 libXi.x86_64 \
       libXtst.x86_64 cups-libs.x86_64 libXScrnSaver.x86_64 libXrandr.x86_64 GConf2.x86_64 alsa-lib.x86_64 atk.x86_64 gtk3.x86_64 \
       ipa-gothic-fonts xorg-x11-fonts-100dpi xorg-x11-fonts-75dpi xorg-x11-utils xorg-x11-fonts-cyrillic xorg-x11-fonts-Type1 xorg-x11-fonts-misc -y
  
  sudo yum update nss -y
  ```

* Run prompt **Navigation timeout** / browser not found / don't know correct **browser executable path**

   Make sure the browser executable path is configured correctly, type: `chrome://version` in the address bar, and find the executable file path:

  ![](./imgs/chrome_path.png)


## Configuration file

rad can be driven entirely by a YAML config file. Pass `--config`/`-c` a path;
if the file does not exist rad writes a fully commented default there and exits
so you can review it first. When `-c` is omitted, `rad_config.yml` in the working
directory is used if present.

```shell
rad -c rad_config.yml http://example.com
```

Keys (all optional):

| Key | Default | Meaning |
|---|---|---|
| `exec_path` | `""` | Path to the chrome executable |
| `disable_headless` | `false` | Disable headless mode |
| `force_sandbox` | `false` | Force the chrome sandbox on |
| `enable_image` | `false` | Load images |
| `parent_path_detect` | `true` | Parent-directory pseudo-static detection |
| `proxy` | `""` | Proxy for all requests |
| `user_agent` | `""` | Request UA; **empty uses the browser's real UA** |
| `domain_headers` | | Per-domain headers, glob syntax, e.g. `[{domain: "*", headers: {Cookie: x}}]` |
| `max_depth` | `10` | Maximum page depth (0 = unlimited) |
| `navigate_timeout_second` | `10` | Navigation timeout |
| `load_timeout_second` | `10` | Page-load timeout |
| `retry` | `0` | Navigation retries after a failure |
| `page_analyze_timeout_second` | `300` | Per-page analysis budget; also applies when the key is omitted |
| `max_interactive` | `1000` | Maximum element interactions per page |
| `max_interactive_depth` | `10` | Interaction depth limit |
| `max_page_concurrent` | `10` | Maximum concurrent pages; values above 10 are clamped to 10 |
| `max_page_visit` | `1000` | Total pages allowed |
| `max_page_visit_per_site` | `1000` | Pages allowed per site |
| `subdomain` | `true` | Crawl subdomains automatically |
| `entrance_retry` | `0` | Extra attempts for the entrance URL |
| `max_similar_request` | `0` | Max similar fetch/XHR requests per site (<=0 unlimited) |
| `element_filter_strength` | `0` | Similar-element filtering, 0 off, 1-7 increasing |
| `new_task_filter_config` | | Whether a link may enter the crawl queue |
| `request_send_filter_config` | | Whether a request may be sent |
| `request_output_filter_config` | | Whether a request may be reported |

Each filter config accepts `hostname_allowed`/`hostname_disallowed`
(`t.com`, `*.t.com`, `1.1.1.1`, `1.1.1.1/24`, `1.1-4.1.1-8`),
`port_allowed`/`port_disallowed` (`80`, `80-85`), and
`path_`, `query_key_`, `fragment_`, `post_key_` allow/disallow glob lists.
Empty lists impose no restriction.

Command-line flags always win over the config file: only flags you actually
pass override a YAML value.

### User agent

`user_agent: ""` means rad sends **no** `User-Agent` override, so Chrome's real
UA is used. That is the default and the recommended setting. Set it only when
you need a fixed UA.

## Output formats

rad matches chaitin/rad's three output files:

* `--text-output/-text FILE` — `METHOD URL`, one per line.
* `--full-text-output/-full FILE` — raw HTTP/1.1 request dumps, separated by a
  50-dash rule.
* `--json-output/-json FILE` — a JSON array, one compact object per line:
  `{"Method","URL","Header","b64_body"}` with the body base64 encoded.
* `--auto-index/-index` — append `-1`, `-2`, ... when the file already exists.

By default results are printed to stdout as `METHOD URL` lines.

## Parameter precedence

Command-line flags win over the config file. Only flags actually passed on the
command line override a YAML value, so a config file is never silently ignored.

## chaitin/rad compatibility

The CLI is a drop-in replacement for chaitin/rad's documented usage:

| chaitin/rad | rad |
|---|---|
| `rad -t http://x` | same (`-t`/`--target`/`--url`/`-u`) |
| `rad -t http://x -wait-login` | same |
| `-text-output`, `-full-text-output`, `-json-output` | same |
| `-auto-index` | same |
| `-http-proxy host:port` | same (upstream proxy) |
| `-disable-headless`, `-dh` | same |
| `-url-file`, `-uf` | same |
| `-config`, `-c` | same (also rad's own long form) |
| `-no-banner` | same |

Because `-t` now means *target* (as in chaitin/rad), the max-tab-count flag uses
`-tab` instead.

## Parameters
### Required parameters
* `--chromium-path Path, --chrome Path`    The path to the chrome executable. (**Required**)
### Basic parameters
* `--config Path, -c Path`  Path of the YAML config file. If it does not exist, a default `rad_config.yml` is written and rad exits. (Default: `./rad_config.yml` if present)
* `--custom-headers Headers`   Customize the HTTP header. Please pass in the data after JSON serialization, this is globally defined and will be used for all requests. (Default: null)
* `--post-data PostData, -d PostData`   POST data. (Default: null)
* `--max-crawled-count Number, -m Number`    The maximum number of tasks for crawlers to avoid long crawling time due to pseudo-static. (Default: 200)
* `--filter-mode Mode, -f Mode`   Filtering mode, `simple`: only static resources and duplicate requests are filtered.  `smart`: with the ability to filter pseudo-static. `strict`: stricter pseudo-static filtering rules. (Default: smart)
* `--output-mode value, -o value`   Result output mode, `console`: print the glorified results directly to the screen. `json`: print the json serialized string of all results.  `none`: don't print the output. (Default: console)
* `--output-json filepath` Write the result to the specified file after JSON serializing it. (Default: null)
* `--request-proxy proxyAddress` socks5 proxy address, all network requests from rad and chrome browser are sent through the proxy. (Default: null)

### Expand input URL
* `--fuzz-path`  Use the built-in dictionary for path fuzzing. (Default: false)
* `--fuzz-path-dict`  Customize the Fuzz path by passing in a dictionary file path, e.g. /home/user/fuzz_dir.txt, each line of the file represents a path to be fuzzed. (Default: null)
* `--robots-path` Resolve the path from the /robots.txt file. (Default: false)

### Form auto-fill
* `--ignore-url-keywords, -iuk` URL keyword that you don't want to visit, generally used to exclude logout links when customizing cookies. Usage: `-iuk logout -iuk exit`. (default: "logout", "quit", "exit")
* `--form-values, -fv` Customize the value of the form fill, set by text type. Support definition types: default, mail, code, phone, username, password, qq, id_card, url, date and number. Text types are identified by the four attribute value keywords `id`, `name`, `class`, `type` of the input box label. For example, define the mailbox input box to be automatically filled with A and the password input box to be automatically filled with B, `-fv mail=A -fv password=B`.Where default represents the fill value when the text type is not recognized, as "Rad". (Default: Rad)
* `--form-keyword-values, -fkv` Customize the value of the form fill, set by keyword fuzzy match. The keyword matches the four attribute values of `id`, `name`, `class`, `type` of the input box label. For example, fuzzy match the pass keyword to fill 123456 and the user keyword to fill admin, `-fkv user=admin -fkv pass=123456`. (Default: Rad)

### Advanced settings for the crawling process
* `--max-tab-count Number, -t Number`   The maximum number of tabs the crawler can open at the same time. (Default: 8)
* `--tab-run-timeout Timeout`   Maximum runtime for a single tab page. (Default: 20s)
* `--wait-dom-content-loaded-timeout Timeout`  The maximum timeout to wait for the page to finish loading. (Default: 5s)
* `--event-trigger-interval Interval` The interval when the event is triggered automatically, generally used in the case of slow target network and DOM update conflicts that lead to URL miss capture. (Default: 100ms)
* `--event-trigger-mode Value` DOM event auto-triggered mode, with `async` and `sync`, for URL miss-catching caused by DOM update conflicts. (Default: async)
* `--before-exit-delay` Delay exit to close chrome at the end of a single tab task. Used to wait for partial DOM updates and XHR requests to be captured. (Default: 1s)

### Other
* `--push-to-proxy` The listener address of the crawler result to be received, usually the listener address of the passive scanner. (Default: null)
* `--push-pool-max` The maximum number of concurrency when sending crawler results to the listening address. (Default: 10)
* `--log-level` Logging levels, debug, info, warn, error and fatal. (Default: info)
* `--no-headless`  Turn off chrome headless mode to visualize the crawling process. (Default: false)

## Follow me

Weibo：[@9ian1i](https://weibo.com/u/5242748339) 
Twitter: [@9ian1i](https://twitter.com/9ian1i)

Related articles：[A browser crawler practice for web vulnerability scanning](https://www.anquanke.com/post/id/178339)
