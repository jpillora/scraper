package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/jpillora/opts"
	"github.com/jpillora/scraper/scraper"
)

var version = "0.0.0"

type config struct {
	scraper.Handler
	ConfigFile string `opts:"mode=arg" help:"Path to JSON <config-file>"`
	Host       string `help:"Listening interface"`
	Port       int    `help:"Listening port"`
	NoLog      bool   `help:"Disable access logs"`
}

func main() {

	c := config{
		Handler: scraper.Handler{Log: true},
		Host:    "0.0.0.0",
		Port:    3000,
	}

	h := &c.Handler

	opts.New(&c).
		Repo("github.com/jpillora/scraper").
		Version(version).
		Parse()

	go func() {
		for {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGHUP)
			<-sig
			if err := h.LoadConfigFile(c.ConfigFile); err != nil {
				log.Printf("[scraper] Failed to load configuration: %s", err)
			} else {
				log.Printf("[scraper] Successfully loaded new configuration")
			}
		}
	}()

	h.Log = !c.NoLog
	if err := h.LoadConfigFile(c.ConfigFile); err != nil {
		log.Fatal(err)
	}

	log.Printf("[scraper] Listening on %d...", c.Port)
	log.Fatal(http.ListenAndServe(c.Host+":"+strconv.Itoa(c.Port), h))
}
