package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jpillora/opts"
	"github.com/jpillora/scraper/lib"
)

func main() {
	s := &scraper.Server{Host: "0.0.0.0", Port: 3000}
	opts.Parse(s)

	go func() {
		for {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGHUP)
			<-c
			if err := s.ReloadConfigFile(); err != nil {
				log.Printf("failed to load configuration: %s", err)
			} else {
				log.Printf("successfully loaded new configuration")
			}
		}
	}()

	log.Fatal(s.Run())
}
