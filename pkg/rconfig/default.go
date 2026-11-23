package rconfig

// DefaultYAML is written to disk on first run. It mirrors chaitin/rad's
// rad_config.yml so existing pipelines can be ported over unchanged.
const DefaultYAML = `# rad configuration file.
# Every key is optional; values shown here are the defaults.

exec_path: ""                     # path to the chrome executable
disable_headless: false           # disable headless mode
subdomain: true                   # crawl subdomains automatically
force_sandbox: false              # force sandbox on; false keeps the default (sandbox off in containers)
enable_image: false               # load images
parent_path_detect: true          # probe parent directories
proxy: ""                         # proxy for all requests
user_agent: ""                    # request user-agent; empty uses the browser's real UA
domain_headers:                   # per-domain headers: []{domain, map[headerKey]HeaderValue}
  - domain: '*'                   # which domains, glob syntax
    headers: {                    # headers, map[key]value
      Cookie: somecookie
    }
max_depth: 10                     # maximum page depth
navigate_timeout_second: 10       # navigation timeout, seconds
load_timeout_second: 10           # page load timeout, seconds
retry: 0                          # retries after a failed visit
page_analyze_timeout_second: 300  # page analysis timeout, seconds
max_interactive: 1000             # maximum interactions per page
max_interactive_depth: 10         # interaction depth limit
max_page_concurrent: 10           # maximum concurrent pages (<=10)
max_page_visit: 1000              # total pages allowed
max_page_visit_per_site: 1000     # pages allowed per site
element_filter_strength: 0        # similar-element filtering strength 1-7; 0 disables it
new_task_filter_config:           # decide whether a link may enter the crawl queue
  hostname_allowed: []
  hostname_disallowed: []
  port_allowed: []
  port_disallowed: []
  path_allowed: []
  path_disallowed: []
  query_key_allowed: []
  query_key_disallowed: []
  fragment_allowed: []
  fragment_disallowed: []
  post_key_allowed: []
  post_key_disallowed: []
request_send_filter_config:       # decide whether a request may be sent
  hostname_allowed: []
  hostname_disallowed: []
  port_allowed: []
  port_disallowed: []
  path_allowed: []
  path_disallowed: []
  query_key_allowed: []
  query_key_disallowed: []
  fragment_allowed: []
  fragment_disallowed: []
  post_key_allowed: []
  post_key_disallowed: []
entrance_retry: 0                 # retries for the entrance URL
max_similar_request: 0            # max similar fetch/XHR requests (<=0 means unlimited)
request_output_filter_config:     # decide whether a request may be reported
  hostname_allowed: []
  hostname_disallowed: []
  port_allowed: []
  port_disallowed: []
  path_allowed: []
  path_disallowed: []
  query_key_allowed: []
  query_key_disallowed: []
  fragment_allowed: []
  fragment_disallowed: []
  post_key_allowed: []
  post_key_disallowed: []
`
