# scraper

[![GoDoc](https://godoc.org/github.com/jpillora/scraper?status.svg)](https://godoc.org/github.com/jpillora/scraper) [![CI](https://github.com/jpillora/scraper/workflows/CI/badge.svg)](https://github.com/jpillora/scraper/actions?workflow=CI)

A dual interface Go module for building simple web scrapers
### Features

* Go struct-tag interface
* Command-line interface
  * HTML⇒JSON API server
  * Single binary
  * Simple configuration
  * Zero-downtime config reload with `kill -s SIGHUP <scraper-pid>`

### Install

**Binaries**

See [the latest release](https://github.com/jpillora/scraper/releases/latest) or download it with this one-liner: `curl https://i.jpillora.com/scraper | bash`

**Source**

``` sh
$ go get -v github.com/jpillora/scraper
```

### Go Example

```go
package main

import (
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
		Query  string   `scraper:"query"`
	}

	g := google{Query: "hello world"}

	if err := scraper.Execute(&g); err != nil {
		log.Fatal(err)
	}

	for i, r := range g.Result {
		fmt.Printf("#%d: '%s' => %s\n", i+1, r.Title, r.URL)
	}
}
```

```
#1: 'Helloworld Travel – Deals on Accommodation, Flights ...' => https://www.helloworld.com.au/
#2: '"Hello, World!" program - Wikipedia' => https://en.wikipedia.org/wiki/%22Hello,_World!%22_program
#3: 'Helloworld Travel - Wikipedia' => https://en.wikipedia.org/wiki/Helloworld_Travel
#4: 'Helloworld Travel Limited' => https://www.helloworldlimited.com.au/
#5: 'Total immersion, Serious fun! with Hello-World!' => https://www.hello-world.com/
#6: 'Helloworld Travel - Home | Facebook' => https://www.facebook.com/helloworldau/
```
### CLI Example

Given `google.json`

``` json
{
  "/search": {
    "url": "https://www.google.com/search?q={{query}}",
    "list": "#rso div[class=g]",
    "result": {
      "title": "h3 span",
      "url": ["a[href]", "@href"]
    }
  }
}
```

``` sh
$ scraper google.json
2015/05/16 20:10:46 listening on 3000...
```

``` sh
$ curl "localhost:3000/search?query=hellokitty"
[
  {
    "title": "Official Home of Hello Kitty \u0026 Friends | Hello Kitty Shop",
    "url": "http://www.sanrio.com/"
  },
  {
    "title": "Hello Kitty - Wikipedia, the free encyclopedia",
    "url": "http://en.wikipedia.org/wiki/Hello_Kitty"
  },
  ...
```

### JSON API

``` plain
{
  <path>: {
    "method": <method>
    "url": <url>
    "list": <selector>,
    "result": {
      <field>: <extractor>,
      <field>: [<extractor>, <extractor>, ...],
      ...
    }
  }
}
```

* `<path>` - **Required** The path of the scraper
  * Accessible at `http://<host>:port/<path>`
  * You may define path variables like: `my/path/:var` when set to `/my/path/foo` then `:var = "foo"`
* `<url>` - **Required** The URL of the remote server to scrape
  * It may contain template variables in the form `{{ var }}`, scraper will look for a `var` path variable, if not found, it will then look for a query parameter `var`
* `result` - **Required** represents the resulting JSON object, after executing the `<extractor>` on the current DOM context. A field may use sequence of `<extractor>`s to perform more complex queries.
* `<method>` - The HTTP request method (defaults to `GET`)
* `<extractor>` - A string in which must be one of:
  * a regex in form `/abc/` - searches the text of the current DOM context (extracts the first group when provided).
  * a regex in form `s/abc/xyz/` - searches the text of the current DOM context and replaces with the provided text (sed-like syntax).
  * an attribute in the form `@abc` - gets the attribute `abc` from the DOM context.
  * a function in the form `html()` - gets the DOM context as string
  * a function in the form `trim()` - trims space from the beginning and the end of the string
  * a query param in the form `query-param(abc)` - parses the current context as a URL and extracts the provided param
  * a css selector `abc` (if not in the forms above) alters the DOM context.
* `list` - **Optional** A css selector used to split the root DOM context into a set of DOM contexts. Useful for capturing search results.

### Go API

Replace `<variable>` with your configuration, documented above.

1. Define your endpoint struct:

```go
type endpoint struct {
  Method string   `scraper:"<method>"`
  URL    string   `scraper:"<url>"`
  Result []result `scraper:"<list>`
  <param>  string `scraper:"<param>"`
}
```

`Method`, `URL`, `Result` and `Debug` are special fields, the remaining **string** fields are treated as input parameters. Input parameters use the field name with first character lowercased by default.

2. Define your result struct:

```go
type result struct {
  <field> string `scraper:"<extractor>"`
  <field> string `scraper:"<extractor> | <extractor>"`
}
```

The result struct is used to define field to extractor mappings. All fields must be `string`s. Struct tags cannot contain arrays so instead we join multiple `extractor`s with ` | `.

3. Execute it:

```go
e := endpoint{MyParam: "hello world"}
if err := scraper.Execute(&e); err != nil {
  ...
}
// e.Result is now set
```

#### Similar projects

*  https://github.com/ernesto-jimenez/scraperboardR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
