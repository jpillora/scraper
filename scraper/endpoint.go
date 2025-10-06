package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/enetx/g"
	"github.com/enetx/surf"
	"github.com/itchyny/gojq"
)

//Endpoint represents a single remote endpoint. The performed
//query can be modified between each call by parameterising
//URL. See documentation.
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

//extract 1 result using this endpoints extractor map
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
	//render url using params
	url, err := template(true, e.URL, params)
	if err != nil {
		return nil, err
	}
	//default method
	method := e.Method
	if method == "" {
		method = "GET"
	}
	//render body (if set)
	body := io.Reader(nil)
	if e.Body != "" {
		s, err := template(true, e.Body, params)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(s)
		if e.Debug {
			logf("req: %s %s (body size %d)", method, url, len(s))
		}
	} else {
		if e.Debug {
			logf("req: %s %s", method, url)
		}
	}
	//show results
	//create surf client
	client := surf.NewClient()

	//create surf request based on method
	var surfReq *surf.Request
	switch method {
	case "GET":
		surfReq = client.Get(g.String(url))
	case "POST":
		bodyData := ""
		if body != nil {
			// Read body content for POST
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				return nil, err
			}
			bodyData = string(bodyBytes)
		}
		surfReq = client.Post(g.String(url), bodyData)
	case "PUT":
		bodyData := ""
		if body != nil {
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				return nil, err
			}
			bodyData = string(bodyBytes)
		}
		surfReq = client.Put(g.String(url), bodyData)
	case "DELETE":
		surfReq = client.Delete(g.String(url))
	default:
		surfReq = client.Get(g.String(url))
	}

	//add headers
	if e.Headers != nil {
		headers := make([]any, 0, len(e.Headers)*2)
		for k, v := range e.Headers {
			headers = append(headers, k, v)
			if e.Debug {
				logf("header: %s=%s", k, v)
			}
		}
		surfReq = surfReq.AddHeaders(headers...)
	}

	//make backend HTTP request
	result := surfReq.Do()
	if result.IsErr() {
		return nil, result.Err()
	}

	resp := result.Ok()
	if e.Debug {
		logf("resp: %d (type: %s)", resp.StatusCode,
			resp.Headers.Get("Content-Type"))
	}

	// Choose extraction method based on mode
	mode := e.Mode
	if mode == "" {
		mode = "html" // default to HTML mode
	}

	switch mode {
	case "json":
		return e.extractJSON(resp.Body.Reader)
	default: // "html" or empty
		return e.extractHTML(resp.Body.Reader)
	}
}

// extractHTML extracts results from HTML response using CSS selectors
func (e *Endpoint) extractHTML(body io.Reader) ([]Result, error) {
	//parse HTML
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}
	sel := doc.Selection
	//results will be either a list of results, or a single result
	var results []Result
	if e.List != "" {
		sels := sel.Find(e.List)
		if e.Debug {
			logf("list: %s => #%d elements", e.List, sels.Length())
		}
		if e.Debug && sels.Length() == 0 {
			logf("no results, printing HTML")
			h, _ := sel.Html()
			fmt.Println(h)
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

// extractJSON extracts results from JSON response using jq selectors
func (e *Endpoint) extractJSON(body io.Reader) ([]Result, error) {
	// Parse JSON
	var data interface{}
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Default list selector to "." if empty
	listSelector := e.List
	if listSelector == "" {
		listSelector = "."
	}

	// Execute list selector
	items, err := e.executeJSONSelector(data, listSelector)
	if err != nil {
		return nil, fmt.Errorf("list selector error: %w", err)
	}

	// Extract results from items
	results := e.extractJSONResults(items)

	if e.Debug {
		logf("list: %s => #%d elements", listSelector, len(results))
	}

	return results, nil
}

// executeJSONSelector executes a jq selector and returns all matching items
func (e *Endpoint) executeJSONSelector(data interface{}, selector string) ([]interface{}, error) {
	query, err := gojq.Parse(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selector '%s': %w", selector, err)
	}

	var items []interface{}
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

// extractJSONResults extracts result fields from each item
func (e *Endpoint) extractJSONResults(items []interface{}) []Result {
	var results []Result

	for _, item := range items {
		r := e.extractJSONResult(item)
		if len(r) > 0 {
			results = append(results, r)
		}
	}

	return results
}

// extractJSONResult extracts result fields from a single item
func (e *Endpoint) extractJSONResult(item interface{}) Result {
	r := Result{}

	for field, extractors := range e.Result {
		if len(extractors) == 0 {
			continue
		}

		selStr := extractors[0].val
		val, err := e.extractJSONField(item, selStr)
		if err != nil {
			if e.Debug {
				logf("failed to extract field %s: %v", field, err)
			}
			continue
		}
		r[field] = val
	}

	return r
}

// extractJSONField extracts a single field using a jq selector
func (e *Endpoint) extractJSONField(data interface{}, selector string) (string, error) {
	query, err := gojq.Parse(selector)
	if err != nil {
		return "", fmt.Errorf("failed to parse selector '%s': %w", selector, err)
	}

	iter := query.Run(data)
	v, ok := iter.Next()
	if !ok {
		return "", fmt.Errorf("no result from selector '%s'", selector)
	}
	if err, ok := v.(error); ok {
		return "", err
	}

	// Convert result to string
	return fmt.Sprintf("%v", v), nil
}
