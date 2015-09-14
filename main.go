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

var VERSION = "0.0.0"

type config struct {
	ConfigFile string `type:"arg" help:"Path to JSON configuration file"`
	Host       string `help:"Listening interface"`
	Port       int    `help:"Listening port"`
}

func main() {

	c := config{
		Host: "0.0.0.0",
		Port: 3000,
	}

	opts.New(&c).
		Repo("github.com/jpillora/scraper").
		Version(VERSION).
		Parse()

	h := &scraper.Handler{Log: true}

	go func() {
		for {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGHUP)
			<-sig
			if err := h.LoadConfigFile(c.ConfigFile); err != nil {
				log.Printf("failed to load configuration: %s", err)
			} else {
				log.Printf("successfully loaded new configuration")
			}
		}
	}()

	if err := h.LoadConfigFile(c.ConfigFile); err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %d...", c.Port)
	log.Fatal(http.ListenAndServe(c.Host+":"+strconv.Itoa(c.Port), h))
}
