package scraper

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func parseHTML(t *testing.T, html string) *goquery.Selection {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc.Selection
}

func runExtractor(t *testing.T, html, selector string) string {
	t.Helper()
	ex, err := NewExtractor(selector)
	if err != nil {
		t.Fatalf("NewExtractor(%q): %v", selector, err)
	}
	v, _ := ex.fn("", parseHTML(t, html))
	return v
}

// runChain narrows context with a CSS selector first, then runs the chain
// — mirrors how Endpoint applies a list of Extractors.
func runChain(t *testing.T, html string, specs ...string) string {
	t.Helper()
	exs := make(Extractors, len(specs))
	for i, s := range specs {
		ex, err := NewExtractor(s)
		if err != nil {
			t.Fatalf("NewExtractor(%q): %v", s, err)
		}
		exs[i] = ex
	}
	return exs.execute(parseHTML(t, html))
}

func TestDefaultSelectorExtractor(t *testing.T) {
	got := runExtractor(t, `<h1>Hello</h1>`, "h1")
	if got != "Hello" {
		t.Errorf("got %q, want Hello", got)
	}
}

func TestDefaultSelectorMultiMatchCommaJoin(t *testing.T) {
	got := runExtractor(t, `<ul><li>a</li><li>b</li><li>c</li></ul>`, "li")
	if got != "a,b,c" {
		t.Errorf("got %q, want a,b,c", got)
	}
}

func TestAttrExtractor(t *testing.T) {
	ex, err := NewExtractor("@href")
	if err != nil {
		t.Fatal(err)
	}
	sel := parseHTML(t, `<a href="https://example.com">x</a>`).Find("a")
	got, _ := ex.fn("", sel)
	if got != "https://example.com" {
		t.Errorf("got %q, want https://example.com", got)
	}
}

func TestRegexMatchExtractor(t *testing.T) {
	got := runExtractor(t, `<p>price is $42.50 today</p>`, `/\$(\d+\.\d+)/`)
	if got != "42.50" {
		t.Errorf("got %q, want 42.50", got)
	}
}

func TestRegexMatchExtractorNoCapture(t *testing.T) {
	got := runExtractor(t, `<p>foo bar baz</p>`, `/bar/`)
	if got != "bar" {
		t.Errorf("got %q, want bar", got)
	}
}

func TestSedReplaceFirstOnly(t *testing.T) {
	got := runChain(t, `<p>aaaa</p>`, "p", `s/a/b/`)
	if got != "baaa" {
		t.Errorf("got %q, want baaa", got)
	}
}

func TestSedReplaceGlobal(t *testing.T) {
	got := runChain(t, `<p>aaaa</p>`, "p", `s/a/b/g`)
	if got != "bbbb" {
		t.Errorf("got %q, want bbbb", got)
	}
}

func TestSedReplaceBackref(t *testing.T) {
	got := runChain(t, `<p>v1.2.3</p>`, "p", `s/v(\d+)\.(\d+)\.(\d+)/$1-$2-$3/`)
	if got != "1-2-3" {
		t.Errorf("got %q, want 1-2-3", got)
	}
}

func TestSedReplaceCustomDelimiter(t *testing.T) {
	got := runChain(t, `<p>a/b/c</p>`, "p", `s|/|-|g`)
	if got != "a-b-c" {
		t.Errorf("got %q, want a-b-c", got)
	}
}

func TestParseSed(t *testing.T) {
	tests := []struct {
		in     string
		ok     bool
		match  string
		repl   string
		global bool
	}{
		{"s/a/b/", true, "a", "b", false},
		{"s/a/b/g", true, "a", "b", true},
		{"s|x|y|g", true, "x", "y", true},
		{"s/a/b/X", false, "", "", false}, // bad flag
		{"s//b/", false, "", "", false},   // empty match
		{"s/a/b", false, "", "", false},   // missing trailing delim
		{"s/a/b/g/extra", false, "", "", false},
		{"x/a/b/", false, "", "", false}, // not s
		{"s", false, "", "", false},      // too short
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseSed(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok=%v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got.match != tt.match || got.repl != tt.repl || got.global != tt.global {
				t.Errorf("got %+v, want match=%q repl=%q global=%v",
					got, tt.match, tt.repl, tt.global)
			}
		})
	}
}

func TestFirstExtractorNarrowsSelection(t *testing.T) {
	ex1, _ := NewExtractor("li")
	ex2, _ := NewExtractor("first()")
	sel := parseHTML(t, `<ul><li>a</li><li>b</li></ul>`)
	v, sel := ex1.fn("", sel)
	if v != "a,b" {
		t.Fatalf("after li: value=%q, want a,b", v)
	}
	_, sel = ex2.fn(v, sel)
	if sel.Length() != 1 {
		t.Errorf("after first(): selection length=%d, want 1", sel.Length())
	}
	if sel.Text() != "a" {
		t.Errorf("after first(): text=%q, want a", sel.Text())
	}
}

func TestHTMLExtractor(t *testing.T) {
	got := runExtractor(t, `<div><span>x</span></div>`, "html()")
	if !strings.Contains(got, "<span>") {
		t.Errorf("got %q, want HTML containing <span>", got)
	}
}

func TestTrimExtractor(t *testing.T) {
	exSel, _ := NewExtractor("p")
	exTrim, _ := NewExtractor("trim()")
	sel := parseHTML(t, `<p>   spaced   </p>`)
	v, sel := exSel.fn("", sel)
	v, _ = exTrim.fn(v, sel)
	if v != "spaced" {
		t.Errorf("got %q, want spaced", v)
	}
}

func TestQueryParamExtractor(t *testing.T) {
	exAttr, _ := NewExtractor("@href")
	exQP, _ := NewExtractor("query-param(q)")
	sel := parseHTML(t, `<a href="/url?q=hello&r=other">x</a>`).Find("a")
	v, sel := exAttr.fn("", sel)
	v, _ = exQP.fn(v, sel)
	if v != "hello" {
		t.Errorf("got %q, want hello", v)
	}
}

func TestJoinExtractor(t *testing.T) {
	tests := []struct {
		spec string
		want string
	}{
		{`join(|)`, "a|b|c"},
		{`join("|")`, "a|b|c"},
		{`join(", ")`, "a, b, c"},
		{`join("\n")`, "a\nb\nc"},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			ex, err := NewExtractor("li")
			if err != nil {
				t.Fatal(err)
			}
			exJoin, err := NewExtractor(tt.spec)
			if err != nil {
				t.Fatalf("NewExtractor(%q): %v", tt.spec, err)
			}
			sel := parseHTML(t, `<ul><li>a</li><li>b</li><li>c</li></ul>`)
			_, sel = ex.fn("", sel)
			v, _ := exJoin.fn("", sel)
			if v != tt.want {
				t.Errorf("got %q, want %q", v, tt.want)
			}
		})
	}
}

func TestExtractorsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"single string", `"h1"`, []string{"h1"}},
		{"array", `["a[href]", "@href"]`, []string{"a[href]", "@href"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ex Extractors
			if err := ex.UnmarshalJSON([]byte(tt.in)); err != nil {
				t.Fatal(err)
			}
			if len(ex) != len(tt.want) {
				t.Fatalf("got %d extractors, want %d", len(ex), len(tt.want))
			}
			for i, e := range ex {
				if e.val != tt.want[i] {
					t.Errorf("[%d] got %q, want %q", i, e.val, tt.want[i])
				}
			}
		})
	}
}

func TestExtractorChaining(t *testing.T) {
	// Equivalent to ["a[href]", "@href"] from JSON config
	ex1, _ := NewExtractor("a[href]")
	ex2, _ := NewExtractor("@href")
	es := Extractors{ex1, ex2}
	sel := parseHTML(t, `<div><a href="/x">click</a></div>`)
	got := es.execute(sel)
	if got != "/x" {
		t.Errorf("got %q, want /x", got)
	}
}
