package main

import (
	"fmt"
	"log"

	"github.com/jpillora/scraper/scraper"
)

func main() {

	type result struct {
		Title string `scraper:"h3 span"`
		URL   string `scraper:"a[href] | @href"`
	}

	type google struct {
		URL    string   `scraper:"https://www.google.com/search?q={{query}}"`
		Result []result `scraper:"#rso div[class=g]"`
		Debug  bool
		Query  string `scraper:"query"`
	}

	g := google{Query: "hello world", Debug: true}

	if err := scraper.Execute(&g); err != nil {
		log.Fatal(err)
	}

	for i, r := range g.Result {
		fmt.Printf("#%d: '%s' => %s\n", i+1, r.Title, r.URL)
	}
}
