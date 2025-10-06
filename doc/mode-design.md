# Mode Design

## Overview

The scraper supports two modes for extracting data from endpoints: **HTML mode** and **JSON mode**.

## Mode Property

Each endpoint can specify a `mode` property in its configuration. If not specified, the mode defaults to `"html"`.

```json
{
  "/endpoint": {
    "mode": "html",  // or "json"
    "url": "...",
    ...
  }
}
```

## HTML Mode (Default)

HTML mode is the original scraping mode that uses CSS selectors to extract data from HTML responses.

### How it works:

1. Makes HTTP request to the URL
2. Parses the response as HTML
3. Uses CSS selector in `list` field to find list elements
4. For each list element, uses CSS selectors in `result` fields to extract data

### Example:

```json
{
  "/search": {
    "mode": "html",
    "url": "https://html.duckduckgo.com/html/?q={{query}}",
    "list": ".result",
    "result": {
      "title": ".result__title a",
      "url": [".result__url", "@href"]
    }
  }
}
```

### Selector Format:

- Simple selector: `"h1"` - extracts text content
- Attribute selector: `["a.detLink", "@href"]` - extracts href attribute
- Regex selector: `"/Size (\\d+(\\.\\d+).[KMG]iB)/"` - extracts using regex

## JSON Mode

JSON mode is designed for scraping JSON APIs using jq-style selectors.

### How it works:

1. Makes HTTP request to the URL
2. Parses the response as JSON
3. Uses jq selector in `list` field to find array of items
4. For each item, uses jq selectors in `result` fields to extract data

### Example:

```json
{
  "/search": {
    "mode": "json",
    "url": "https://api.example.com/q.php?q={{query}}",
    "list": ".",
    "result": {
      "name": ".name",
      "size": ".size",
      "stars": ".stars | tonumber",
      "forks": ".forks | tonumber"
    }
  }
}
```

### JQ Selector Format:

Uses [gojq](https://github.com/itchyny/gojq) library for jq-compatible selectors:

- `.field` - extract field value
- `.field.nested` - extract nested field
- `.field | tonumber` - extract and convert to number
- `.field | tostring` - extract and convert to string
- `.[0]` - extract first element from array
- `.[]` - iterate over array elements

## Implementation

### Endpoint Struct

```go
type Endpoint struct {
    Mode    string                `json:"mode,omitempty"`    // "html" or "json"
    URL     string                `json:"url"`
    List    string                `json:"list,omitempty"`
    Result  map[string]Extractors `json:"result"`
    // ... other fields
}
```

### Execution Flow

```go
func (e *Endpoint) Execute(params map[string]string) ([]Result, error) {
    // Make HTTP request
    resp := client.Get(url).Do()

    // Choose extraction method based on mode
    switch e.Mode {
    case "json":
        return e.extractJSON(resp.Body)
    default: // "html" or empty
        return e.extractHTML(resp.Body)
    }
}
```

## Migration

Existing configurations without a `mode` field will automatically use HTML mode, ensuring backward compatibility.
