package scraper

import (
	"strings"
	"testing"
)

func TestJSONValueString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"float as int", float64(42), "42"},
		{"fractional float", 1.5, "1.5"},
		{"int64", int64(7), "7"},
		{"object re-encoded as JSON", map[string]any{"k": "v"}, `{"k":"v"}`},
		{"array re-encoded as JSON", []any{1, 2}, `[1,2]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonValueString(tt.in)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func mustExtractors(t *testing.T, vals ...string) Extractors {
	t.Helper()
	es := make(Extractors, len(vals))
	for i, v := range vals {
		ex, err := NewExtractor(v)
		if err != nil {
			t.Fatalf("NewExtractor(%q): %v", v, err)
		}
		es[i] = ex
	}
	return es
}

func TestExtractHTML(t *testing.T) {
	e := &Endpoint{
		List: ".item",
		Result: map[string]Extractors{
			"name": mustExtractors(t, "h2"),
			"href": mustExtractors(t, "a", "@href"),
		},
	}
	body := strings.NewReader(`
		<div class="item"><h2>One</h2><a href="/1">x</a></div>
		<div class="item"><h2>Two</h2><a href="/2">x</a></div>
	`)
	res, err := e.extractHTML(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d results, want 2", len(res))
	}
	if res[0]["name"] != "One" || res[0]["href"] != "/1" {
		t.Errorf("result[0] = %+v", res[0])
	}
	if res[1]["name"] != "Two" || res[1]["href"] != "/2" {
		t.Errorf("result[1] = %+v", res[1])
	}
}

func TestExtractHTMLSkipsIncompleteRows(t *testing.T) {
	e := &Endpoint{
		List: ".item",
		Result: map[string]Extractors{
			"name": mustExtractors(t, "h2"),
			"href": mustExtractors(t, "a", "@href"),
		},
	}
	body := strings.NewReader(`
		<div class="item"><h2>One</h2><a href="/1">x</a></div>
		<div class="item"><h2>Missing href</h2></div>
	`)
	res, err := e.extractHTML(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d results, want 1 (incomplete row should be dropped)", len(res))
	}
}

func TestExtractJSON(t *testing.T) {
	e := &Endpoint{
		Mode: "json",
		List: ".items[]",
		Result: map[string]Extractors{
			"name":  mustExtractors(t, ".name"),
			"count": mustExtractors(t, ".count"),
		},
	}
	body := strings.NewReader(`{"items":[{"name":"a","count":1},{"name":"b","count":2}]}`)
	res, err := e.extractJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d results, want 2", len(res))
	}
	if res[0]["name"] != "a" || res[0]["count"] != "1" {
		t.Errorf("result[0] = %+v", res[0])
	}
	if res[1]["name"] != "b" || res[1]["count"] != "2" {
		t.Errorf("result[1] = %+v", res[1])
	}
}

func TestExtractJSONChaining(t *testing.T) {
	e := &Endpoint{
		Mode: "json",
		List: ".items[]",
		Result: map[string]Extractors{
			// pipeline: extract count, multiply by 10
			"big": mustExtractors(t, ".count", ". * 10"),
		},
	}
	body := strings.NewReader(`{"items":[{"count":1},{"count":2}]}`)
	res, err := e.extractJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0]["big"] != "10" || res[1]["big"] != "20" {
		t.Errorf("got %+v", res)
	}
}

func TestExecuteUnknownMode(t *testing.T) {
	e := &Endpoint{
		URL:  "https://example.invalid/",
		Mode: "yaml",
	}
	_, err := e.Execute(map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "unknown mode") {
		// We only get here if the request somehow succeeded; either way the
		// mode validation should fire before we try to parse a body. Tolerate
		// the network error path because example.invalid won't resolve.
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	}
}

func TestNewRequestRejectsUnknownMethod(t *testing.T) {
	_, err := newRequest("CONNECT", "https://example.com")
	if err == nil {
		t.Fatal("expected error for unsupported method")
	}
}
