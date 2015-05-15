package scraper

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type config map[string]*endpoint

type endpoint struct {
	URL  string
	List string
	Item map[string]interface{}
}

type Server struct {
	//config
	Host       string `help:"Listening interface"`
	Port       int    `help:"Listening port"`
	ConfigFile string `type:"arg" help:"Path to JSON configuration file"`
	//state
	config *config
}

func (s *Server) Run() error {
	if err := s.ReloadConfigFile(); err != nil {
		return nil
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

	for path, e := range c {
		log.Printf("%s = %+v", path, e)
	}

	return nil
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}
