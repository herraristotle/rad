package requests

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/herraristotle/rad/pkg/logger"
	"github.com/pkg/errors"
)

// DefaultUa is used by the standalone HTTP client, which has no browser to
// inherit a user agent from.
const DefaultUa = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)" +
	" Chrome/120.0.0.0 Safari/537.36"

const defaultTimeout int = 15

// Cap the response at 100K, which covers the vast majority of cases
const defaultResponseLength = 10240
const defaultRetry = 0

var ContentTypes = map[string]string{
	"json":      "application/json",
	"xml":       "application/xml",
	"soap":      "application/soap+xml",
	"multipart": "multipart/form-data",
	"form":      "application/x-www-form-urlencoded; charset=utf-8",
}

// ReqInfo wraps an HTTP request element for quick, simple requests
type ReqInfo struct {
	Verb    string
	Url     string
	Headers map[string]string
	Body    []byte
}

type ReqOptions struct {
	Timeout       int    // in seconds
	Retry         int    // 0 is the default; -1 disables retries
	VerifySSL     bool   // default false
	AllowRedirect bool   // default false
	Proxy         string // proxy settings, support http/https proxy only, e.g. http://127.0.0.1:8080
}

type session struct {
	ReqOptions
	client *http.Client
}

// getSessionByOptions Build a session from the given options
func getSessionByOptions(options *ReqOptions) *session {
	if options == nil {
		options = &ReqOptions{}
	}
	// Set client timeout and SSL verification
	timeout := time.Duration(options.Timeout) * time.Second
	if options.Timeout == 0 {
		timeout = time.Duration(defaultTimeout) * time.Second
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !options.VerifySSL},
	}
	if options.Proxy != "" {
		proxyUrl, err := url.Parse(options.Proxy)
		if err == nil {
			tr.Proxy = http.ProxyURL(proxyUrl)
		}
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr}
	// Control redirect following
	if !options.AllowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	// Copy the options onto the session
	return &session{
		ReqOptions: ReqOptions{
			options.Timeout,
			options.Retry,
			options.VerifySSL,
			options.AllowRedirect,
			options.Proxy,
		},
		client: client,
	}
}

// Get issues a GET request
func Get(url string, headers map[string]string, options *ReqOptions) (*Response, error) {
	sess := getSessionByOptions(options)
	return sess.doRequest("GET", url, headers, nil)
}

// Request issues a custom request type
func Request(verb string, url string, headers map[string]string, body []byte, options *ReqOptions) (*Response, error) {
	sess := getSessionByOptions(options)
	return sess.doRequest(verb, url, headers, body)
}

// session functions

// Get performs a GET on the session
func (sess *session) Get(url string, headers map[string]string) (*Response, error) {
	return sess.doRequest("GET", url, headers, nil)
}

// Post performs a POST on the session
func (sess *session) Post(url string, headers map[string]string, body []byte) (*Response, error) {
	return sess.doRequest("POST", url, headers, body)
}

// Request performs a custom request type on the session
func (sess *session) Request(verb string, url string, headers map[string]string, body []byte) (*Response, error) {
	return sess.doRequest(verb, url, headers, body)
}

// Request is the fast path for a ReqInfo
func (r *ReqInfo) Request() (*Response, error) {
	return Request(r.Verb, r.Url, r.Headers, r.Body, nil)
}

func (r *ReqInfo) RequestWithOptions(options *ReqOptions) (*Response, error) {
	return Request(r.Verb, r.Url, r.Headers, r.Body, options)
}

func (r *ReqInfo) Clone() *ReqInfo {
	return &ReqInfo{
		Verb:    r.Verb,
		Url:     r.Url,
		Headers: r.Headers,
		Body:    r.Body,
	}
}

func (r *ReqInfo) SetHeader(name, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
}

// doRequest The function that actually performs the request
func (sess *session) doRequest(verb string, url string, headers map[string]string, body []byte) (*Response, error) {
	logger.Logger.Debug("do request to ", url)
	verb = strings.ToUpper(verb)
	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		// usually because the URL contains a bare %
		url = escapePercentSign(url)
		req, err = http.NewRequest(verb, url, bodyReader)
	}
	if err != nil {
		return nil, errors.Wrap(err, "build request error")
	}

	// Set the headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	// Set the default headers
	defaultHeaders := map[string]string{
		"User-Agent": DefaultUa,
		"Range":      fmt.Sprintf("bytes=0-%d", defaultResponseLength),
		"Connection": "close",
	}
	for key, value := range defaultHeaders {
		if _, ok := headers[key]; !ok {
			req.Header.Set(key, value)
		}
	}
	// Set the Host header
	if host, ok := headers["Host"]; ok {
		req.Host = host
	}
	// Set the default Content-Type header
	if verb == "POST" && headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		// Referer, Origin and X-Requested-With must be set manually
	}
	// Override the Connection header
	req.Header.Set("Connection", "close")

	// Set the retry count
	retry := sess.Retry
	if retry == 0 {
		retry = defaultRetry
	} else if retry == -1 {
		retry = 0
	}

	// Perform the request
	var resp *http.Response
	for i := 0; i <= retry; i++ {
		resp, err = sess.client.Do(req)
		if err != nil {
			// sleep 0.1s
			time.Sleep(100 * time.Microsecond)
			continue
		} else {
			break
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "error occurred during request")
	}
	// With a Range header most web servers reply 206 PARTIAL CONTENT; normalize it to 200 OK
	if resp.StatusCode == 206 {
		resp.StatusCode = 200
		resp.Status = "200 OK"
	}

	return NewResponse(resp), nil
}
