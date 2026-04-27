package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	h.Log = !c.NoLog
	if err := h.LoadConfigFile(c.ConfigFile); err != nil {
		log.Fatal(err)
	}

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for range hup {
			if err := h.LoadConfigFile(c.ConfigFile); err != nil {
				log.Printf("[scraper] Failed to load configuration: %s", err)
			} else {
				log.Printf("[scraper] Successfully loaded new configuration")
			}
		}
	}()

	srv := &http.Server{
		Addr:              c.Host + ":" + strconv.Itoa(c.Port),
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		log.Printf("[scraper] Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[scraper] Shutdown error: %s", err)
		}
	}()

	log.Printf("[scraper] Listening on %d...", c.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
