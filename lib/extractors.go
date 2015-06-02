package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

type extractor func(string, *goquery.Selection) (string, *goquery.Selection)

type extractorGenerator func(string) (extractor, error)

type extractors []extractor

//execute all extractors on the query
func (ex extractors) execute(s *goquery.Selection) string {
	v := ""
	for _, e := range ex {
		v, s = e(v, s)
	}
	return v
}

func (ex *extractors) UnmarshalJSON(data []byte) error {
	//force array
	if bytes.IndexRune(data, '[') != 0 {
		data = append([]byte{'['}, append(data, ']')...)
	}
	//parse strings
	strs := []string{}
	if err := json.Unmarshal(data, &strs); err != nil {
		return err
	}
	//reset extractors
	*ex = make(extractors, len(strs))
	//convert all strings
	for i, s := range strs {
		fn, err := ex.convert(s)
		if err != nil {
			return err
		}
		(*ex)[i] = fn
	}
	return nil
}

//convert strings into extractor functions
func (e *extractors) convert(s string) (extractor, error) {
	for re, generator := range generators {
		m := re.FindStringSubmatch(s)
		if len(m) > 0 {
			if len(m) > 1 {
				s = m[1]
			}
			return generator(s)
		}
	}
	return defaultGenerator(s)
}

//selector extractor
var defaultGenerator = func(selstr string) (extractor, error) {
	if err := checkSelector(selstr); err != nil {
		return nil, fmt.Errorf("Invalid selector: %s", err)
	}
	return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
		s := sel.Find(selstr)
		if value == "" {
			value = s.Text()
		}
		return value, s
	}, nil
}

var generatorsPre = map[string]extractorGenerator{
	//attr generator
	`^@(.+)`: func(attr string) (extractor, error) {
		//make attribute extractor
		return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
			value, _ = sel.Attr(attr)
			// h, _ := sel.Html()
			// log.Printf("attr===\n%s\n\n", h)
			return value, sel
		}, nil
	},
	//regex generator
	`^\/(.+)\/$`: func(reStr string) (extractor, error) {
		re, err := regexp.Compile(reStr)
		if err != nil {
			return nil, fmt.Errorf("Invalid regex '%s': %s", reStr, err)
		}
		return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
			ctx := value
			if ctx == "" {
				ctx, _ = sel.Html()
			}
			m := re.FindStringSubmatch(ctx)
			if len(m) == 0 {
				value = ""
			} else if m[1] != "" {
				value = m[1]
			} else {
				value = m[0]
			}
			return value, sel
		}, nil
	},
	//first() generator
	`^first\(\)$`: func(_ string) (extractor, error) {
		return func(value string, sel *goquery.Selection) (string, *goquery.Selection) {
			return value, sel.First()
		}, nil
	},
}

//run-time generators
var generators = map[*regexp.Regexp]extractorGenerator{}

func init() {
	for str, gen := range generatorsPre {
		generators[regexp.MustCompile(str)] = gen
	}
}
