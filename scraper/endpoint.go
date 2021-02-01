package scraper

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

//Endpoint represents a single remote endpoint. The performed
//query can be modified between each call by parameterising
//URL. See documentation.
type Endpoint struct {
	Name    string                `json:"name,omitempty"`
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
	//create HTTP request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	h := http.Header{}
	if e.Headers != nil {
		for k, v := range e.Headers {
			h.Set(k, v)
		}
	}
	if e.Debug {
		for k := range h {
			logf("header: %s=%s", k, h.Get(k))
		}
	}
	req.Header = h
	//make backend HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if e.Debug {
		logf("resp: %d (type: %s, len: %s)", resp.StatusCode,
			resp.Header.Get("Content-Type"), resp.Header.Get("Content-Length"))
	}
	//parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
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
