# Tasks

## Completed
- [x] Create git branch "surf"
- [x] Upgrade to latest Go version in go.mod
- [x] Switch HTTP client to https://github.com/enetx/surf
- [x] Test it still works using CLI (main.go) + example/google.json
- [x] Add duckduckgo.json example
- [x] Delete cache.json

## JSON API Mode Implementation
- [x] Write doc/mode-design.md documenting the new mode system
- [x] Add "mode" property to Endpoint struct (defaults to "html")
- [x] Install gojq dependency (https://github.com/itchyny/gojq)
- [x] Implement "json" mode using jq selectors for list and result fields
- [x] Create new cache.json using api.example.com JSON API
- [x] Test JSON mode with cache.json
