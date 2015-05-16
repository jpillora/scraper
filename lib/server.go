package scraper

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"strconv"

	"github.com/PuerkitoBio/goquery"
)

//a single result
type result map[string]string

//the configuration file
type config map[string]*endpoint

type Server struct {
	//config
	Host       string `help:"Listening interface"`
	Port       int    `help:"Listening port"`
	ConfigFile string `type:"arg" help:"Path to JSON configuration file"`
	//state
	config config
}

func (s *Server) Run() error {
	if err := s.ReloadConfigFile(); err != nil {
		return err
	}
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	h := http.Server{Addr: addr, Handler: http.HandlerFunc(s.handle)}
	log.Printf("listening on %d...", s.Port)
	return h.ListenAndServe()
}

func (s *Server) ReloadConfigFile() error {
	b, err := ioutil.ReadFile(s.ConfigFile)
	if err != nil {
		return err
	}

	c := config{}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return err
	}

	//replace config
	s.config = c
	return nil
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	for path, e := range s.config {
		if r.URL.Path == string(path) {
			s.execute(e, w, r)
			return
		}
	}
	w.WriteHeader(404)
	w.Write([]byte("Not found"))
}

func (s *Server) execute(e *endpoint, w http.ResponseWriter, r *http.Request) {

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

	log.Printf("GET %s => %s", url, resp.Status)

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
		sel.Find(e.List).Each(func(i int, sel *goquery.Selection) {
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
