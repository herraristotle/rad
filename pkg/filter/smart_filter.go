package filter

import (
	"go/types"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/herraristotle/rad/pkg/config"
	"github.com/herraristotle/rad/pkg/logger"
	"github.com/herraristotle/rad/pkg/model"
	"github.com/herraristotle/rad/pkg/tools"

	mapset "github.com/deckarep/golang-set"
)

type SmartFilter struct {
	StrictMode bool
	// ParentPathDetect enables the parent-directory pseudo-static heuristic.
	ParentPathDetect bool
	*SimpleFilter
	filterLocationSet          mapset.Set // locations of non-logic params; globally marked for filtering
	filterParamKeyRepeatCount  sync.Map
	filterParamKeySingleValues sync.Map // count of duplicate occurrences per param name
	filterPathParamKeySymbol   sync.Map // mark count per param value under a given path
	filterParamKeyAllValues    sync.Map
	filterPathParamEmptyValues sync.Map
	filterParentPathValues     sync.Map
	uniqueMarkedIds            mapset.Set // unique IDs after marking, used for dedup
}

const (
	MaxParentPathCount         = 32 // max correction count for this path level relative to its parent
	MaxParamKeySingleCount     = 8  // max correction for duplicate param names in one URL
	MaxParamKeyAllCount        = 10 // max correction for one param name across all URLs in this round
	MaxPathParamEmptyCount     = 10 // max correction for empty-valued param names under one path
	MaxPathParamKeySymbolCount = 5  // if markings for one param under one path exceed this, mark it globally
)

const (
	CustomValueMark    = "{{Rad}}"
	FixParamRepeatMark = "{{fix_param}}"
	FixPathMark        = "{{fix_path}}"
	TooLongMark        = "{{long}}"
	NumberMark         = "{{number}}"
	ChineseMark        = "{{chinese}}"
	UpperMark          = "{{upper}}"
	LowerMark          = "{{lower}}"
	UrlEncodeMark      = "{{urlencode}}"
	UnicodeMark        = "{{unicode}}"
	BoolMark           = "{{bool}}"
	ListMark           = "{{list}}"
	TimeMark           = "{{time}}"
	MixAlphaNumMark    = "{{mix_alpha_num}}"
	MixSymbolMark      = "{{mix_symbol}}"
	MixNumMark         = "{{mix_num}}"
	NoLowerAlphaMark   = "{{no_lower}}"
	MixStringMark      = "{{mix_str}}"
)

var chineseRegex = regexp.MustCompile("[\u4e00-\u9fa5]+")
var urlencodeRegex = regexp.MustCompile("(?:%[A-Fa-f0-9]{2,6})+")
var unicodeRegex = regexp.MustCompile(`(?:\\u\w{4})+`)
var onlyAlphaRegex = regexp.MustCompile("^[a-zA-Z]+$")
var onlyAlphaUpperRegex = regexp.MustCompile("^[A-Z]+$")
var alphaUpperRegex = regexp.MustCompile("[A-Z]+")
var alphaLowerRegex = regexp.MustCompile("[a-z]+")
var replaceNumRegex = regexp.MustCompile(`[0-9]+\.[0-9]+|\d+`)
var onlyNumberRegex = regexp.MustCompile(`^[0-9]+$`)
var numberRegex = regexp.MustCompile(`[0-9]+`)
var OneNumberRegex = regexp.MustCompile(`[0-9]`)
var numSymbolRegex = regexp.MustCompile(`\.|_|-`)
var timeSymbolRegex = regexp.MustCompile(`-|:|\s`)
var onlyAlphaNumRegex = regexp.MustCompile(`^[0-9a-zA-Z]+$`)
var markedStringRegex = regexp.MustCompile(`^{{.+}}$`)
var htmlReplaceRegex = regexp.MustCompile(`\.shtml|\.html|\.htm`)

func NewSmartFilter(base *SimpleFilter, strictMode bool) *SmartFilter {
	return NewSmartFilterWithOptions(base, strictMode, true)
}

// NewSmartFilterWithOptions builds a smart filter. parentPathDetect maps to
// parent_path_detect in rad_config.yml; when false the parent-directory
// pseudo-static heuristic is skipped.
func NewSmartFilterWithOptions(base *SimpleFilter, strictMode, parentPathDetect bool) *SmartFilter {
	s := &SmartFilter{ParentPathDetect: parentPathDetect}
	s.filterLocationSet = mapset.NewSet()
	s.filterParamKeyRepeatCount = sync.Map{}
	s.filterParamKeySingleValues = sync.Map{}
	s.filterPathParamKeySymbol = sync.Map{}
	s.filterParamKeyAllValues = sync.Map{}
	s.filterPathParamEmptyValues = sync.Map{}
	s.filterParentPathValues = sync.Map{}
	s.uniqueMarkedIds = mapset.NewSet()
	s.SimpleFilter = base
	s.StrictMode = strictMode
	return s
}

/*
*
Smart deduplication.
Optional strict mode.

Returns true when the request should be filtered out.
*/
func (s *SmartFilter) DoFilter(req *model.Request) bool {
	// First filter static resources, apply basic dedup, and drop other domains
	if s.SimpleFilter.DoFilter(req) {
		logger.Logger.Debugf("filter req by simplefilter: " + req.URL.RequestURI())
		return true
	}

	req.Filter.FragmentID = s.calcFragmentID(req.URL.Fragment)

	// Apply markings
	if req.Method == config.GET || req.Method == config.DELETE || req.Method == config.HEAD || req.Method == config.OPTIONS {
		s.getMark(req)
		s.repeatCountStatistic(req)
	} else if req.Method == config.POST || req.Method == config.PUT {
		s.postMark(req)
	} else {
		logger.Logger.Debug("dont support such method: " + req.Method)
	}

	// Deduplicate the marked requests
	uniqueId := req.Filter.UniqueId
	if s.uniqueMarkedIds.Contains(uniqueId) {
		logger.Logger.Debugf("filter req by uniqueMarkedIds 1: " + req.URL.RequestURI())
		return true
	}

	// Apply global marking for numeric params
	s.globalFilterLocationMark(req)

	// Next, deduplicate the marked GET requests
	if req.Method == config.GET || req.Method == config.DELETE || req.Method == config.HEAD || req.Method == config.OPTIONS {
		// Mark GET requests that exceed the thresholds
		s.overCountMark(req)

		// Recompute QueryMapId
		req.Filter.QueryMapId = getParamMapID(req.Filter.MarkedQueryMap)
		// Recompute PathId
		req.Filter.PathId = getPathID(req.Filter.MarkedPath)
	} else {
		// Recompute PostDataId
		req.Filter.PostDataId = getParamMapID(req.Filter.MarkedPostDataMap)
	}

	// Recompute the request unique ID
	req.Filter.UniqueId = getMarkedUniqueID(req)

	// Deduplicate again on the new IDs
	newUniqueId := req.Filter.UniqueId
	if s.uniqueMarkedIds.Contains(newUniqueId) {
		logger.Logger.Debugf("filter req by uniqueMarkedIds 2: " + req.URL.RequestURI())
		return true
	}

	// Add to the result set
	s.uniqueMarkedIds.Add(newUniqueId)
	return false
}

/*
*
The Query map decodes values automatically, so mark RawQuery beforehand
*/
func (s *SmartFilter) preQueryMark(rawQuery string) string {
	if chineseRegex.MatchString(rawQuery) {
		return chineseRegex.ReplaceAllString(rawQuery, ChineseMark)
	} else if urlencodeRegex.MatchString(rawQuery) {
		return urlencodeRegex.ReplaceAllString(rawQuery, UrlEncodeMark)
	} else if unicodeRegex.MatchString(rawQuery) {
		return unicodeRegex.ReplaceAllString(rawQuery, UnicodeMark)
	}
	return rawQuery
}

/*
*
Mark params and path of a GET request
*/
func (s *SmartFilter) getMark(req *model.Request) {
	// First, the pre-decode substitution
	todoURL := *(req.URL)
	todoURL.RawQuery = s.preQueryMark(todoURL.RawQuery)

	// Apply the markings in order
	queryMap := todoURL.QueryMap()
	queryMap = markParamName(queryMap)
	queryMap = s.markParamValue(queryMap, *req)
	markedPath := MarkPath(todoURL.Path)

	// Compute the unique ID
	var queryKeyID string
	var queryMapID string
	if len(queryMap) != 0 {
		queryKeyID = getKeysID(queryMap)
		queryMapID = getParamMapID(queryMap)
	} else {
		queryKeyID = ""
		queryMapID = ""
	}
	pathID := getPathID(markedPath)

	req.Filter.MarkedQueryMap = queryMap
	req.Filter.QueryKeysId = queryKeyID
	req.Filter.QueryMapId = queryMapID
	req.Filter.MarkedPath = markedPath
	req.Filter.PathId = pathID

	// Finally compute the unique request ID after marking
	req.Filter.UniqueId = getMarkedUniqueID(req)
}

/*
*
Mark params and path of a POST request
*/
func (s *SmartFilter) postMark(req *model.Request) {
	postDataMap := req.PostDataMap()

	postDataMap = markParamName(postDataMap)
	postDataMap = s.markParamValue(postDataMap, *req)
	markedPath := MarkPath(req.URL.Path)

	// Compute the unique ID
	var postDataMapID string
	if len(postDataMap) != 0 {
		postDataMapID = getParamMapID(postDataMap)
	} else {
		postDataMapID = ""
	}
	pathID := getPathID(markedPath)

	req.Filter.MarkedPostDataMap = postDataMap
	req.Filter.PostDataId = postDataMapID
	req.Filter.MarkedPath = markedPath
	req.Filter.PathId = pathID

	// Finally compute the unique request ID after marking
	req.Filter.UniqueId = getMarkedUniqueID(req)
}

/*
*
Mark parameter names
*/
func markParamName(paramMap map[string]interface{}) map[string]interface{} {
	markedParamMap := map[string]interface{}{}
	for key, value := range paramMap {
		// Leave pure alphabetic names alone
		if onlyAlphaRegex.MatchString(key) {
			markedParamMap[key] = value
			// Param name is too long
		} else if len(key) >= 32 {
			markedParamMap[TooLongMark] = value
			// Strip the digits
		} else {
			key = replaceNumRegex.ReplaceAllString(key, NumberMark)
			markedParamMap[key] = value
		}
	}
	return markedParamMap
}

/*
*
Mark parameter values
*/
func (s *SmartFilter) markParamValue(paramMap map[string]interface{}, req model.Request) map[string]interface{} {
	markedParamMap := map[string]interface{}{}
	for key, value := range paramMap {
		switch value.(type) {
		case bool:
			markedParamMap[key] = BoolMark
			continue
		case types.Slice:
			markedParamMap[key] = ListMark
			continue
		case float64:
			markedParamMap[key] = NumberMark
			continue
		}
		// Only handle strings
		valueStr, ok := value.(string)
		if !ok {
			continue
		}
		// Values rad injected itself mark this param as numeric rather than logic; record it for global filtering
		if config.IsFormFillSentinel(valueStr) {
			name := req.URL.Hostname() + req.URL.Path + req.Method + key
			s.filterLocationSet.Add(name)
			markedParamMap[key] = CustomValueMark
			// all uppercase letters
		} else if onlyAlphaUpperRegex.MatchString(valueStr) {
			markedParamMap[key] = UpperMark
			// value is 16 characters or longer
		} else if len(valueStr) >= 16 {
			markedParamMap[key] = TooLongMark
			// consists only of digits and a few symbols
		} else if onlyNumberRegex.MatchString(valueStr) || onlyNumberRegex.MatchString(numSymbolRegex.ReplaceAllString(valueStr, "")) {
			markedParamMap[key] = NumberMark
			// contains Chinese characters
		} else if chineseRegex.MatchString(valueStr) {
			markedParamMap[key] = ChineseMark
			// urlencode
		} else if urlencodeRegex.MatchString(valueStr) {
			markedParamMap[key] = UrlEncodeMark
			// unicode
		} else if unicodeRegex.MatchString(valueStr) {
			markedParamMap[key] = UnicodeMark
			// looks like a timestamp
		} else if onlyNumberRegex.MatchString(timeSymbolRegex.ReplaceAllString(valueStr, "")) {
			markedParamMap[key] = TimeMark
			// letters plus digits
		} else if onlyAlphaNumRegex.MatchString(valueStr) && numberRegex.MatchString(valueStr) {
			markedParamMap[key] = MixAlphaNumMark
			// contains some special symbols
		} else if hasSpecialSymbol(valueStr) {
			markedParamMap[key] = MixSymbolMark
			// more than 3 digits => treat as a numeric param
		} else if b := OneNumberRegex.ReplaceAllString(valueStr, "0"); strings.Count(b, "0") >= 3 {
			markedParamMap[key] = MixNumMark
			// strict mode
		} else if s.StrictMode {
			// no lowercase letters
			if !alphaLowerRegex.MatchString(valueStr) {
				markedParamMap[key] = NoLowerAlphaMark
				// typical values mix uppercase, lowercase, digits and underscores; more than three of these classes means pseudo-static
			} else {
				count := 0
				if alphaLowerRegex.MatchString(valueStr) {
					count += 1
				}
				if alphaUpperRegex.MatchString(valueStr) {
					count += 1
				}
				if numberRegex.MatchString(valueStr) {
					count += 1
				}
				if strings.Contains(valueStr, "_") || strings.Contains(valueStr, "-") {
					count += 1
				}
				if count >= 3 {
					markedParamMap[key] = MixStringMark
				}
			}
		} else {
			markedParamMap[key] = value
		}
	}
	return markedParamMap
}

/*
*
Mark the path
*/
func MarkPath(path string) string {
	pathParts := strings.Split(path, "/")
	for index, part := range pathParts {
		if len(part) >= 32 {
			pathParts[index] = TooLongMark
		} else if onlyNumberRegex.MatchString(numSymbolRegex.ReplaceAllString(part, "")) {
			pathParts[index] = NumberMark
		} else if strings.HasSuffix(part, ".html") || strings.HasSuffix(part, ".htm") || strings.HasSuffix(part, ".shtml") {
			part = htmlReplaceRegex.ReplaceAllString(part, "")
			// mixed uppercase, lowercase and digits
			if numberRegex.MatchString(part) && alphaUpperRegex.MatchString(part) && alphaLowerRegex.MatchString(part) {
				pathParts[index] = MixAlphaNumMark
				// pure digits
			} else if b := numSymbolRegex.ReplaceAllString(part, ""); onlyNumberRegex.MatchString(b) {
				pathParts[index] = NumberMark
			}
			// contains special symbols
		} else if hasSpecialSymbol(part) {
			pathParts[index] = MixSymbolMark
		} else if chineseRegex.MatchString(part) {
			pathParts[index] = ChineseMark
		} else if unicodeRegex.MatchString(part) {
			pathParts[index] = UnicodeMark
		} else if onlyAlphaUpperRegex.MatchString(part) {
			pathParts[index] = UpperMark
			// consists only of digits and a few symbols
		} else if b := numSymbolRegex.ReplaceAllString(part, ""); onlyNumberRegex.MatchString(b) {
			pathParts[index] = NumberMark
			// more than 3 digits => treat the path as pseudo-static
		} else if b := OneNumberRegex.ReplaceAllString(part, "0"); strings.Count(b, "0") > 3 {
			pathParts[index] = MixNumMark
		}
	}
	newPath := strings.Join(pathParts, "/")
	return newPath
}

/*
*
Global numeric-param filtering
*/
func (s *SmartFilter) globalFilterLocationMark(req *model.Request) {
	name := req.URL.Hostname() + req.URL.Path + req.Method
	if req.Method == config.GET || req.Method == config.DELETE || req.Method == config.HEAD || req.Method == config.OPTIONS {
		for key := range req.Filter.MarkedQueryMap {
			name += key
			if s.filterLocationSet.Contains(name) {
				req.Filter.MarkedQueryMap[key] = CustomValueMark
			}
		}
	} else if req.Method == config.POST || req.Method == config.PUT {
		for key := range req.Filter.MarkedPostDataMap {
			name += key
			if s.filterLocationSet.Contains(name) {
				req.Filter.MarkedPostDataMap[key] = CustomValueMark
			}
		}
	}
}

/*
*
Count duplicates globally across param names, param values and paths,
then mark the parts that exceed the thresholds
*/
func (s *SmartFilter) repeatCountStatistic(req *model.Request) {
	queryKeyId := req.Filter.QueryKeysId
	pathId := req.Filter.PathId
	if queryKeyId != "" {
		// count of duplicate occurrences per param name
		if v, ok := s.filterParamKeyRepeatCount.Load(queryKeyId); ok {
			s.filterParamKeyRepeatCount.Store(queryKeyId, v.(int)+1)
		} else {
			s.filterParamKeyRepeatCount.Store(queryKeyId, 1)
		}

		for key, value := range req.Filter.MarkedQueryMap {
			// duplicate param names within one URL
			paramQueryKey := queryKeyId + key

			if set, ok := s.filterParamKeySingleValues.Load(paramQueryKey); ok {
				set := set.(mapset.Set)
				set.Add(value)
			} else {
				s.filterParamKeySingleValues.Store(paramQueryKey, mapset.NewSet(value))
			}

			//count duplicates of one param across all URLs in this round
			if _, ok := s.filterParamKeyAllValues.Load(key); !ok {
				s.filterParamKeyAllValues.Store(key, mapset.NewSet(value))
			} else {
				if v, ok := s.filterParamKeyAllValues.Load(key); ok {
					set := v.(mapset.Set)
					if !set.Contains(value) {
						set.Add(value)
					}
				}
			}

			// if the value is empty, count empty-valued param names under this path
			if value == "" {
				if _, ok := s.filterPathParamEmptyValues.Load(pathId); !ok {
					s.filterPathParamEmptyValues.Store(pathId, mapset.NewSet(key))
				} else {
					if v, ok := s.filterPathParamEmptyValues.Load(pathId); ok {
						set := v.(mapset.Set)
						if !set.Contains(key) {
							set.Add(key)
						}
					}
				}
			}

			pathIdKey := pathId + key
			// count dedup markings per param value under one path
			if v, ok := s.filterPathParamKeySymbol.Load(pathIdKey); ok {
				if markedStringRegex.MatchString(value.(string)) {
					s.filterPathParamKeySymbol.Store(pathIdKey, v.(int)+1)
				}
			} else {
				s.filterPathParamKeySymbol.Store(pathIdKey, 1)
			}

		}
	}

	// Count this path level relative to its parent; when a file extension is present, let common script suffixes through
	if req.URL.ParentPath() == "" || inCommonScriptSuffix(req.URL.FileExt()) {
		return
	}

	//
	parentPathId := tools.StrMd5(req.URL.ParentPath())
	currentPath := strings.Replace(req.Filter.MarkedPath, req.URL.ParentPath(), "", -1)
	if _, ok := s.filterParentPathValues.Load(parentPathId); !ok {
		s.filterParentPathValues.Store(parentPathId, mapset.NewSet(currentPath))
	} else {
		if v, ok := s.filterParentPathValues.Load(parentPathId); ok {
			set := v.(mapset.Set)
			if !set.Contains(currentPath) {
				set.Add(currentPath)
			}
		}
	}
}

/*
*
After the duplicate counting, mark again whatever exceeds the thresholds
*/
func (s *SmartFilter) overCountMark(req *model.Request) {
	queryKeyId := req.Filter.QueryKeysId
	pathId := req.Filter.PathId
	// when the param value is not empty,
	if req.Filter.QueryKeysId != "" {
		// if duplicate param names in one URL exceed the threshold and the param has more than three distinct values, mark it
		if v, ok := s.filterParamKeyRepeatCount.Load(queryKeyId); ok && v.(int) > MaxParamKeySingleCount {
			for key := range req.Filter.MarkedQueryMap {
				paramQueryKey := queryKeyId + key
				if set, ok := s.filterParamKeySingleValues.Load(paramQueryKey); ok {
					set := set.(mapset.Set)
					if set.Cardinality() > 3 {
						req.Filter.MarkedQueryMap[key] = FixParamRepeatMark
					}
				}
			}
		}

		for key := range req.Filter.MarkedQueryMap {
			// across all URLs, if one param has more than the threshold of distinct values, mark it for dedup
			if paramKeySet, ok := s.filterParamKeyAllValues.Load(key); ok {
				paramKeySet := paramKeySet.(mapset.Set)
				if paramKeySet.Cardinality() > MaxParamKeyAllCount {
					req.Filter.MarkedQueryMap[key] = FixParamRepeatMark
				}
			}

			pathIdKey := pathId + key
			// if dedup markings for a GET param value under one PATH exceed the threshold, mark that param globally for the PATH
			if v, ok := s.filterPathParamKeySymbol.Load(pathIdKey); ok && v.(int) > MaxPathParamKeySymbolCount {
				req.Filter.MarkedQueryMap[key] = FixParamRepeatMark
			}
		}

		// Handle paths whose empty-valued param count exceeds the threshold, e.g. pseudo-static: http://bang.360.cn/?chu_xiu
		if v, ok := s.filterPathParamEmptyValues.Load(pathId); ok {
			set := v.(mapset.Set)
			if set.Cardinality() > MaxPathParamEmptyCount {
				newMarkerQueryMap := map[string]interface{}{}
				for key, value := range req.Filter.MarkedQueryMap {
					if value == "" {
						newMarkerQueryMap[FixParamRepeatMark] = ""
					} else {
						newMarkerQueryMap[key] = value
					}
				}
				req.Filter.MarkedQueryMap = newMarkerQueryMap
			}
		}
	}

	// Handle pseudo-static at this path level
	if !s.ParentPathDetect || req.URL.ParentPath() == "" || inCommonScriptSuffix(req.URL.FileExt()) {
		return
	}
	parentPathId := tools.StrMd5(req.URL.ParentPath())
	if set, ok := s.filterParentPathValues.Load(parentPathId); ok {
		set := set.(mapset.Set)
		if set.Cardinality() > MaxParentPathCount {
			if strings.HasSuffix(req.URL.ParentPath(), "/") {
				req.Filter.MarkedPath = req.URL.ParentPath() + FixPathMark
			} else {
				req.Filter.MarkedPath = req.URL.ParentPath() + "/" + FixPathMark
			}
		}
	}
}

// calcFragmentID Compute a unique fragment value when the fragment is formatted as a URL path
func (s *SmartFilter) calcFragmentID(fragment string) string {
	if fragment == "" || !strings.HasPrefix(fragment, "/") {
		return ""
	}
	fakeUrl, err := model.GetUrl(fragment)
	if err != nil {
		logger.Logger.Error("cannot calculate url fragment: ", err)
		return ""
	}
	// XXX: the fragment is marked with a throwaway request and never counted
	fakeReq := model.GetRequest(config.GET, fakeUrl)
	s.getMark(&fakeReq)
	// s.repeatCountStatistic(&fakeReq)
	return fakeReq.Filter.UniqueId
}

/*
*
Compute the unique request ID after marking
*/
func getMarkedUniqueID(req *model.Request) string {
	var paramId string
	if req.Method == config.GET || req.Method == config.DELETE || req.Method == config.HEAD || req.Method == config.OPTIONS {
		paramId = req.Filter.QueryMapId
	} else {
		paramId = req.Filter.PostDataId
	}

	uniqueStr := req.Method + paramId + req.Filter.PathId + req.URL.Host + req.Filter.FragmentID
	if req.RedirectionFlag {
		uniqueStr += "Redirection"
	}
	if req.URL.Path == "/" && req.URL.RawQuery == "" && req.URL.Scheme == "https" {
		uniqueStr += "https"
	}

	return tools.StrMd5(uniqueStr)
}

/*
*
Compute the unique ID after marking request param keys
*/
func getKeysID(dataMap map[string]interface{}) string {
	var keys []string
	var idStr string
	for key := range dataMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		idStr += key
	}
	return tools.StrMd5(idStr)
}

/*
*
Compute the unique ID after marking request params
*/
func getParamMapID(dataMap map[string]interface{}) string {
	var keys []string
	var idStr string
	var markReplaceRegex = regexp.MustCompile(`{{.+}}`)
	for key := range dataMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := dataMap[key]
		idStr += key
		if value, ok := value.(string); ok {
			idStr += markReplaceRegex.ReplaceAllString(value, "{{mark}}")
		}
	}
	return tools.StrMd5(idStr)
}

/*
*
Compute the unique ID after marking the PATH
*/
func getPathID(path string) string {
	return tools.StrMd5(path)
}

/*
*
Report whether the string contains any of these special symbols
*/
func hasSpecialSymbol(str string) bool {
	symbolList := []string{"{", "}", " ", "|", "#", "@", "$", "*", ",", "<", ">", "/", "?", "\\", "+", "="}
	for _, sym := range symbolList {
		if strings.Contains(str, sym) {
			return true
		}
	}
	return false
}

func inCommonScriptSuffix(suffix string) bool {
	return config.ScriptSuffixSet.Contains(suffix)
}
