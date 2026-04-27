package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Extractor struct {
	val string
	fn  extractorFn
}

func NewExtractor(value string) (*Extractor, error) {
	e := &Extractor{}
	if err := e.Set(value); err != nil {
		return nil, err
	}
	return e, nil
}

func MustExtractor(value string) *Extractor {
	e := &Extractor{}
	if err := e.Set(value); err != nil {
		panic(err)
	}
	return e
}

//sets the current string Value as the Extractor function
func (e *Extractor) Set(value string) (err error) {
	for _, g := range generators {
		if g.match(value) {
			e.val = value
			e.fn, err = g.generate(value)
			return
		}
	}
	e.val = value
	e.fn, err = defaultGenerator(value)
	return
}

type extractorFn func(string, *goquery.Selection) (string, *goquery.Selection)

type extractorGenerator func(string) (extractorFn, error)

type Extractors []*Extractor

//execute all Extractors on the query
func (ex Extractors) execute(s *goquery.Selection) string {
	v := ""
	for _, e := range ex {
		v, s = e.fn(v, s)
	}
	return v
}

func (ex *Extractors) UnmarshalJSON(data []byte) error {
	//force array
	if bytes.IndexRune(data, '[') != 0 {
		data = append([]byte{'['}, append(data, ']')...)
	}
	//parse strings
	strs := []string{}
	if err := json.Unmarshal(data, &strs); err != nil {
		return err
	}
	//reset Extractors
	*ex = make(Extractors, len(strs))
	//convert all strings
	for i, s := range strs {
		e := &Extractor{}
		if err := e.Set(s); err != nil {
			return err
		}
		(*ex)[i] = e
	}
	return nil
}

func (ex Extractors) MarshalJSON() ([]byte, error) {
	strs := make([]string, len(ex))
	for i, e := range ex {
		strs[i] = e.val
	}
	return json.Marshal(strs)
}

type sedExpr struct {
	match  string
	repl   string
	global bool
}

// parseSed accepts sed-style "s<delim>match<delim>repl<delim>[g]". The
// delimiter is taken from the second character. Match cannot be empty.
// Flags must be empty or "g". Returns ok=false on any deviation.
func parseSed(s string) (sedExpr, bool) {
	var z sedExpr
	if len(s) < 5 || s[0] != 's' {
		return z, false
	}
	delim := s[1]
	if delim == 0 || delim == ' ' {
		return z, false
	}
	parts := strings.Split(s, string(delim))
	if len(parts) != 4 {
		return z, false
	}
	if parts[1] == "" {
		return z, false
	}
	flags := parts[3]
	if flags != "" && flags != "g" {
		return z, false
	}
	return sedExpr{match: parts[1], repl: parts[2], global: flags == "g"}, true
}

//selector Extractor
var defaultGenerator = func(selstr string) (extractorFn, error) {
	if err := checkSelector(selstr); err != nil {
		return nil, fmt.Errorf("invalid selector: %s", err)
	}
	return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
		s := sel.Find(selstr)
		if value == "" {
			if l := s.Length(); l == 1 {
				value = s.Text()
			} else if l > 1 {
				strs := make([]string, l)
				s.Each(func(i int, s *goquery.Selection) {
					strs[i] = s.Text()
				})
				value = strings.Join(strs, ",")
			}
		}
		return value, s
	}, nil
}

//custom Extractor functions
var generators = []struct {
	match    func(extractor string) bool
	generate extractorGenerator
}{
	//attr generator
	{
		match: func(extractor string) bool {
			return strings.HasPrefix(extractor, "@")
		},
		generate: func(extractor string) (extractorFn, error) {
			attr := strings.TrimPrefix(extractor, "@")
			//make attribute Extractor
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				value, _ = sel.Attr(attr)
				return value, sel
			}, nil
		},
	},
	//regex match generator
	{
		match: func(extractor string) bool {
			return strings.HasPrefix(extractor, "/") && strings.HasSuffix(extractor, "/")
		},
		generate: func(extractor string) (extractorFn, error) {
			reStr := strings.TrimSuffix(strings.TrimPrefix(extractor, "/"), "/")
			re, err := regexp.Compile(reStr)
			if err != nil {
				return nil, fmt.Errorf("invalid regex '%s': %s", reStr, err)
			}
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				ctx := value
				if ctx == "" {
					ctx, _ = sel.Html() //force text
				}
				m := re.FindStringSubmatch(ctx)
				if len(m) == 0 {
					value = ""
				} else if len(m) >= 2 && m[1] != "" {
					value = m[1]
				} else {
					value = m[0]
				}
				return value, sel
			}, nil
		},
	},
	//regex (sed syntax) replace generator: s<delim>match<delim>repl<delim>[g]
	//repl supports Go regexp expansion syntax ($1, ${name}, $$ for literal $)
	{
		match: func(extractor string) bool {
			_, ok := parseSed(extractor)
			return ok
		},
		generate: func(extractor string) (extractorFn, error) {
			p, ok := parseSed(extractor)
			if !ok {
				return nil, fmt.Errorf("invalid s/.../.../ expression: %q", extractor)
			}
			re, err := regexp.Compile(p.match)
			if err != nil {
				return nil, fmt.Errorf("invalid regex '%s' (%s)", p.match, err)
			}
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				ctx := value
				if ctx == "" {
					ctx, _ = sel.Html()
				}
				if p.global {
					value = re.ReplaceAllString(ctx, p.repl)
					return value, sel
				}
				// non-global: replace only the first match, preserving the rest verbatim
				m := re.FindStringSubmatchIndex(ctx)
				if m == nil {
					value = ctx
					return value, sel
				}
				expanded := re.ExpandString(nil, p.repl, ctx, m)
				value = ctx[:m[0]] + string(expanded) + ctx[m[1]:]
				return value, sel
			}, nil
		},
	},
	//first generator
	{
		match: func(extractor string) bool {
			return extractor == "first()"
		},
		generate: func(_ string) (extractorFn, error) {
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				return value, sel.First()
			}, nil
		},
	},
	//html generator
	{
		match: func(extractor string) bool {
			return extractor == "html()"
		},
		generate: func(_ string) (extractorFn, error) {
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				html, _ := sel.Html()
				return html, sel
			}, nil
		},
	},
	//trim generator
	{
		match: func(extractor string) bool {
			return extractor == "trim()"
		},
		generate: func(_ string) (extractorFn, error) {
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				return strings.TrimSpace(value), sel
			}, nil
		},
	},
	//query param generator
	{
		match: func(extractor string) bool {
			return strings.HasPrefix(extractor, "query-param(") && strings.HasSuffix(extractor, ")")
		},
		generate: func(extractor string) (extractorFn, error) {
			param := strings.TrimSuffix(strings.TrimPrefix(extractor, "query-param("), ")")
			return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
				ctx := value
				if ctx == "" {
					ctx, _ = sel.Html() //force text
				}
				u, err := url.Parse(ctx)
				if err != nil {
					return "", sel
				}
				return u.Query().Get(param), sel
			}, nil
		},
	},
	//join generator: join(sep) joins each matched element's text with sep.
	//Replaces the default comma-join when applied to a multi-match selection.
	//`sep` may be quoted ("|", "\n") or a bare token ( - , : ).
	{
		match: func(extractor string) bool {
			return strings.HasPrefix(extractor, "join(") && strings.HasSuffix(extractor, ")")
		},
		generate: func(extractor string) (extractorFn, error) {
			raw := strings.TrimSuffix(strings.TrimPrefix(extractor, "join("), ")")
			sep, err := unquoteJoinSep(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid join separator %q: %s", raw, err)
			}
			return func(_ string, sel *goquery.Selection) (string, *goquery.Selection) {
				parts := make([]string, 0, sel.Length())
				sel.Each(func(_ int, s *goquery.Selection) {
					parts = append(parts, s.Text())
				})
				return strings.Join(parts, sep), sel
			}, nil
		},
	},
}

// unquoteJoinSep accepts a Go-quoted string ("\n", "|") or a bare separator.
func unquoteJoinSep(s string) (string, error) {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`') {
		return strconv.Unquote(s)
	}
	return s, nil
}
