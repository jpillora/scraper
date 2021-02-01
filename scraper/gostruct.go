package scraper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

//Execute builds an Endpoint with Extractors using the given
//struct and executes it
func Execute(gostruct interface{}) error {
	//must be struct pointer
	v := reflect.ValueOf(gostruct)
	if v.Type().Kind() != reflect.Ptr && v.Type().Elem().Kind() != reflect.Struct {
		return errors.New("expected struct pointer")
	}
	v = v.Elem()
	//generate endpoint, params and put results back into struct
	endpoint, err := newEndpointFromStruct(v)
	if err != nil {
		return err
	}
	if endpoint.Debug {
		j, _ := json.MarshalIndent(endpoint, "", "  ")
		logf("computed endpoint: %s", j)
	}
	params := newParamsFromStruct(v)
	results, err := endpoint.Execute(params)
	if err != nil {
		return err
	}
	return setResultsToStruct(results, v)
}

func newEndpointFromStruct(v reflect.Value) (*Endpoint, error) {
	//build up endpoint
	e := &Endpoint{}
	//method
	parseMethod(v, e)
	parseHeaders(v, e)
	parseBody(v, e)
	parseDebug(v, e)
	//url
	if err := parseURL(v, e); err != nil {
		return nil, err
	}
	//list/extractors
	if err := parseListExtractors(v, e); err != nil {
		return nil, err
	}
	return e, nil

}

func parseMethod(v reflect.Value, e *Endpoint) {
	t := v.Type()
	u, ok := t.FieldByName("Method")
	if ok && u.Type.Kind() == reflect.String {
		if s := v.FieldByName("Method").Interface().(string); s != "" {
			e.Method = s
		}
	}
}

func parseHeaders(v reflect.Value, e *Endpoint) {
	h := http.Header{}
	hv := reflect.ValueOf(h)
	t := v.Type()
	u, ok := t.FieldByName("Headers")
	if ok && u.Type == hv.Type() {
		h, ok = v.FieldByName("Headers").Interface().(http.Header)
		if ok && len(h) > 0 {
			m := map[string]string{}
			for k := range h {
				m[k] = h.Get(k)
			}
			e.Headers = m
		}
	}
}

func parseBody(v reflect.Value, e *Endpoint) {
	t := v.Type()
	u, ok := t.FieldByName("Body")
	if ok && u.Type.Kind() == reflect.String {
		e.Body = v.FieldByName("Body").Interface().(string)
	}
}

func parseDebug(v reflect.Value, e *Endpoint) {
	t := v.Type()
	u, ok := t.FieldByName("Debug")
	if ok && u.Type.Kind() == reflect.Bool {
		e.Debug = v.FieldByName("Debug").Interface().(bool)
	}
}

func parseURL(v reflect.Value, e *Endpoint) error {
	t := v.Type()
	u, ok := t.FieldByName("URL")
	if !ok || u.Type.Kind() != reflect.String {
		return errors.New("expected URL string field")
	}
	url := u.Tag.Get("scraper")
	if s := v.FieldByName("URL").Interface().(string); s != "" {
		url = s
	}
	e.URL = url
	return nil
}

func parseListExtractors(v reflect.Value, e *Endpoint) error {
	t := v.Type()
	r, ok := t.FieldByName("Result")
	if !ok {
		return errors.New("expected Result struct or []struct")
	}
	rt := r.Type
	if rt.Kind() == reflect.Slice {
		//extract list selector
		l := r.Tag.Get("scraper")
		if l == "" {
			return errors.New("expected slice field to have list selector")
		}
		e.List = l
		//elem is extractor set
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return errors.New("expected Result struct or []struct")
	}
	//result list struct fields are extractors
	extractors := map[string]Extractors{}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		s := f.Tag.Get("scraper")
		if s == "" {
			return fmt.Errorf("expected result field %s to have selector (scraper struct tag)", f.Name)
		}
		es := Extractors{}
		for _, sel := range strings.Split(s, " | ") {
			e, err := NewExtractor(sel)
			if err != nil {
				return fmt.Errorf("result field %s: %s: %s", f.Name, sel, err)
			}
			es = append(es, e)
		}
		extractors[f.Name] = es
	}
	e.Result = extractors
	return nil
}

func newParamsFromStruct(v reflect.Value) map[string]string {
	t := v.Type()
	params := map[string]string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		n := f.Name
		if n == "_" || n == "Method" || n == "URL" || n == "Body" || n == "Result" {
			continue
		}
		if f.Type.Kind() != reflect.String {
			continue
		}
		name := f.Tag.Get("scraper")
		if name == "" {
			name = strings.ToLower(n[:1]) + n[1:]
		}
		value := v.Field(i).Interface().(string)
		params[name] = value
	}
	return params
}

func setResultsToStruct(results []Result, v reflect.Value) error {
	t := v.Type()
	r, ok := t.FieldByName("Result")
	if !ok {
		panic("expected Result struct or []struct")
	}
	st := r.Type
	//single struct?
	if st.Kind() != reflect.Slice {
		return setResultToStruct(results, v)
	}
	//create empty slice
	sv := reflect.MakeSlice(st, len(results), len(results))
	//take type of given struct
	et := st.Elem()
	//loop results
	for i, kvs := range results {
		//create new struct per result
		ev := reflect.New(et).Elem()
		//set each kv
		for k, v := range kvs {
			evf := ev.FieldByName(k)
			evf.Set(reflect.ValueOf(v))
		}
		//add new struct to new slice
		sv.Index(i).Set(ev)
	}
	//set results onto original gostruct
	v.FieldByName("Result").Set(sv)
	return nil
}

func setResultToStruct(results []Result, v reflect.Value) error {
	return errors.New("only list implemented")
}
