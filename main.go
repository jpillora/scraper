package main

import (
	"log"

	"github.com/jpillora/opts"
	"github.com/jpillora/scraper/lib"
)

func main() {
	s := &scraper.Server{Host: "0.0.0.0", Port: 3000}
	opts.Parse(s)
	log.Fatal(s.Run())
}
