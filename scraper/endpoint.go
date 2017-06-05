package scraper

import (
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
	url, err := template(true, e.URL, params)
	if err != nil {
		return nil, err
	}

	method := e.Method
	if method == "" {
		method = "GET"
	}

	body := io.Reader(nil)
	if e.Body != "" {
		if s, err := template(true, e.Body, params); err != nil {
			return nil, err
		} else {
			body = strings.NewReader(s)
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if e.Headers != nil {
		for k, v := range e.Headers {
			logf("setting header %s to %s", k, v)
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if e.Debug {
		logf("%s %s => %s", method, url, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	sel := doc.Selection

	var results []Result
	//out will be either a list of results, or a single result
	if e.List != "" {
		sels := sel.Find(e.List)
		if e.Debug {
			logf("list: %s => #%d elements", e.List, sels.Length())
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
		results[0] = e.extract(sel)
	}

	return results, err
}
