package scraper

import "github.com/PuerkitoBio/goquery"

type endpoint struct {
	URL    string
	List   string
	Result map[string]extractors
}

//extract 1 result using this endpoints extractor map
func (e *endpoint) extract(sel *goquery.Selection) result {
	r := result{}
	for field, ext := range e.Result {
		if v := ext.execute(sel); v != "" {
			r[field] = v
			// log.Printf("%s: %s", field, v)
		}
	}
	return r
}
