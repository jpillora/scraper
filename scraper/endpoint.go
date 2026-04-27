package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/enetx/g"
	"github.com/enetx/surf"
	"github.com/itchyny/gojq"
)

// shared HTTP client — reuses connection pool across requests
var client = surf.NewClient()

// Endpoint represents a single remote endpoint. The performed
// query can be modified between each call by parameterising
// URL. See documentation.
type Endpoint struct {
	Name    string                `json:"name,omitempty"`
	Mode    string                `json:"mode,omitempty"`
	Method  string                `json:"method,omitempty"`
	URL     string                `json:"url"`
	Body    string                `json:"body,omitempty"`
	Headers map[string]string     `json:"headers,omitempty"`
	List    string                `json:"list,omitempty"`
	Result  map[string]Extractors `json:"result"`
	Debug   bool
}

// extract 1 result using this endpoints extractor map
func (e *Endpoint) extract(sel *goquery.Selection) Result {
	r := Result{}
	for field, ext := range e.Result {
		if v := ext.execute(sel); v != "" {
			r[field] = v
		} else if e.Debug {
			logf("missing %s", field)
		}
	}
	return r
}

// Execute will execute an Endpoint with the given params
func (e *Endpoint) Execute(params map[string]string) ([]Result, error) {
	url, err := template(true, e.URL, params)
	if err != nil {
		return nil, err
	}
	method := e.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := newRequest(method, url)
	if err != nil {
		return nil, err
	}
	if e.Body != "" {
		body, err := template(true, e.Body, params)
		if err != nil {
			return nil, err
		}
		req = req.Body(body)
		if e.Debug {
			logf("req: %s %s (body size %d)", method, url, len(body))
		}
	} else if e.Debug {
		logf("req: %s %s", method, url)
	}
	if len(e.Headers) > 0 {
		headers := make([]any, 0, len(e.Headers)*2)
		for k, v := range e.Headers {
			headers = append(headers, k, v)
			if e.Debug {
				logf("header: %s=%s", k, v)
			}
		}
		req = req.AddHeaders(headers...)
	}

	result := req.Do()
	if result.IsErr() {
		return nil, result.Err()
	}
	resp := result.Ok()
	defer resp.Body.Close()

	if e.Debug {
		logf("resp: %d (type: %s)", resp.StatusCode, resp.Headers.Get("Content-Type"))
	}

	mode := e.Mode
	if mode == "" {
		mode = "html"
	}
	switch mode {
	case "html":
		return e.extractHTML(resp.Body.Reader)
	case "json":
		return e.extractJSON(resp.Body.Reader)
	default:
		return nil, fmt.Errorf("unknown mode %q (expected \"html\" or \"json\")", mode)
	}
}

// newRequest builds a surf request for the given method. surf no longer
// exposes a generic dispatch — each verb has its own builder method.
func newRequest(method, url string) (*surf.Request, error) {
	u := g.String(url)
	switch method {
	case http.MethodGet:
		return client.Get(u), nil
	case http.MethodPost:
		return client.Post(u), nil
	case http.MethodPut:
		return client.Put(u), nil
	case http.MethodPatch:
		return client.Patch(u), nil
	case http.MethodDelete:
		return client.Delete(u), nil
	case http.MethodHead:
		return client.Head(u), nil
	}
	return nil, fmt.Errorf("unsupported HTTP method %q", method)
}

// extractHTML extracts results from an HTML response using CSS selectors
func (e *Endpoint) extractHTML(body io.Reader) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}
	sel := doc.Selection
	var results []Result
	if e.List != "" {
		sels := sel.Find(e.List)
		if e.Debug {
			logf("list: %s => #%d elements", e.List, sels.Length())
			if sels.Length() == 0 {
				logf("no results, printing HTML")
				h, _ := sel.Html()
				fmt.Println(h)
			}
		}
		sels.Each(func(i int, sel *goquery.Selection) {
			r := e.extract(sel)
			if len(r) == len(e.Result) {
				results = append(results, r)
			} else if e.Debug {
				logf("excluded #%d: has %d fields, expected %d", i, len(r), len(e.Result))
			}
		})
	} else {
		results = append(results, e.extract(sel))
	}
	return results, nil
}

// extractJSON extracts results from a JSON response using jq selectors
func (e *Endpoint) extractJSON(body io.Reader) ([]Result, error) {
	var data any
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	listSelector := e.List
	if listSelector == "" {
		listSelector = "."
	}
	items, err := runJQ(data, listSelector)
	if err != nil {
		return nil, fmt.Errorf("list selector %q: %w", listSelector, err)
	}
	if e.Debug {
		logf("list: %s => #%d elements", listSelector, len(items))
	}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		r := e.extractJSONResult(item)
		if len(r) > 0 {
			results = append(results, r)
		}
	}
	return results, nil
}

// extractJSONResult extracts result fields from a single item.
// An extractor list is treated as a jq pipeline: ["a", "b", "c"] becomes
// "a | b | c" — matching the chaining semantics of HTML-mode extractors.
func (e *Endpoint) extractJSONResult(item any) Result {
	r := Result{}
	for field, extractors := range e.Result {
		if len(extractors) == 0 {
			continue
		}
		parts := make([]string, len(extractors))
		for i, ex := range extractors {
			parts[i] = ex.val
		}
		sel := strings.Join(parts, " | ")
		matches, err := runJQ(item, sel)
		if err != nil {
			if e.Debug {
				logf("field %q (%s): %v", field, sel, err)
			}
			continue
		}
		if len(matches) == 0 {
			continue
		}
		r[field] = jsonValueString(matches[0])
	}
	return r
}

// runJQ compiles and runs a jq selector against data, returning all matches.
func runJQ(data any, selector string) ([]any, error) {
	query, err := gojq.Parse(selector)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	var items []any
	iter := query.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		items = append(items, v)
	}
	return items, nil
}

// jsonValueString converts a jq result value into a string suitable for
// inclusion in a Result map. Scalars use their natural string form;
// objects/arrays are re-encoded as JSON.
func jsonValueString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		// jq numbers come back as float64; prefer integer formatting when exact.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case int, int64:
		return fmt.Sprintf("%d", x)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(b)
	}
}
