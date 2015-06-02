package scraper

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

//a single result
type result map[string]string

//the configuration file
type config map[string]*endpoint

type Handler struct {
	Logs   bool
	config config
}

func (h *Handler) LoadConfigFile(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return h.LoadConfig(b)
}

func (h *Handler) LoadConfig(b []byte) error {
	c := config{}
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	//replace config
	h.config = c
	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for path, e := range h.config {
		if r.URL.Path == string(path) {
			h.execute(e, w, r)
			return
		}
	}
	w.WriteHeader(404)
	w.Write([]byte("Not found"))
}

func (h *Handler) execute(e *endpoint, w http.ResponseWriter, r *http.Request) {

	url, err := template(e.URL, r.URL.Query())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if h.Logs {
		log.Printf("scraper GET %s => %s", url, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	sel := doc.Selection

	var out interface{}
	//out will be either a list of results, or a single result
	if e.List != "" {
		var results []result
		sels := sel.Find(e.List)
		sels.Each(func(i int, sel *goquery.Selection) {
			r := e.extract(sel)
			if len(r) == len(e.Result) {
				results = append(results, r)
			}
		})
		out = results
	} else {
		out = e.extract(sel)
	}

	b, _ := json.MarshalIndent(out, "", "  ")
	w.Write(b)
}
