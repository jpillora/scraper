# scraper

A configuration based HTML to JSON API server

:warning: In progress

---

Similar project https://github.com/ernesto-jimenez/scraperboard

---

### Features

* Single binary
* Simple configuration
* Zero-downtime config reload with `SIGHUP`

### Quick Example

Given `google.json`

``` json
{
  "/search": {
    "url": "https://www.google.com/search?q={{q}}",
    "list": "#search ol > li",
    "item": {
      "title": "h3 a",
      "url": ["h3 a","@href"]
    }
  }
}
```

``` sh
$ scraper google.json
listening on 3000...
```

``` sh
$ curl localhost:3000/search?q=helloworld
# [
#  json results...
# ]
```

### Configuration

``` plain
{
  <path>: {
    "url": <url>
    "list": <selector>,
    "item": {
      <field>: <selector>,
      <field>: <selector>,
      ...
    }
  }
}
```

* `<path>` - **Required** The URI path of this scraper GET endpoint
  * You may define path variables like: `/my/path/:var` when set to `/my/path/foo` then `:var = "foo"`
* `<url>` - **Required** The URL of the remote server
  * It may contain template variables in the form `{{ var }}`, scraper will look for a `var` path variable, if not found, it will then look for a query parameter `var`
* `list` - **Optional** When defined, a JSON array will be returned with each `item`, where each element selected is used as the DOM root.
* `item` - **Required** Returns a JSON object with each field defined using their corresponding selectors.
  * Uses the first item from the selection
* `<selector>` - Returns a string by parsing the provided config
  * array selectors must contain a series of string selectors
  * string selectors are parsed in the following order:
    * `/abc/` is viewed as a regexp
    * `abc` is viewed as a CSS selector
    * `@abc` is viewed as an attribute name
    * resulting DOM nodes are converted to strings using their text value


#### MIT License

Copyright © 2015 &lt;dev@jpillora.com&gt;

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
