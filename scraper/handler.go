package scraper

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Result represents a result
type Result map[string]string

// Config is a path → endpoint mapping
type Config map[string]*Endpoint

type Handler struct {
	Config  Config            `opts:"-"`
	Headers map[string]string `opts:"-"`
	Auth    string            `help:"Basic auth credentials <user>:<pass>"`
	Log     bool              `opts:"-"`
	Debug   bool              `help:"Enable debug output"`
}

func (h *Handler) LoadConfigFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return h.LoadConfig(b)
}

func (h *Handler) LoadConfig(b []byte) error {
	c := Config{}
	// json unmarshal performs selector validation
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	for k, e := range c {
		// normalise path: lookup later strips the leading slash
		if strings.HasPrefix(k, "/") {
			delete(c, k)
			k = strings.TrimPrefix(k, "/")
			c[k] = e
		}
		if h.Log {
			logf("Loaded endpoint: /%s", k)
		}
		// inherit handler-level Debug + Headers (per-endpoint values win)
		e.Debug = h.Debug
		if e.Headers == nil {
			e.Headers = h.Headers
		} else {
			for k, v := range h.Headers {
				if _, ok := e.Headers[k]; !ok {
					e.Headers[k] = v
				}
			}
		}
	}
	if h.Debug {
		logf("Enabled debug mode")
	}
	h.Config = c
	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// basic auth
	if h.Auth != "" {
		u, p, _ := r.BasicAuth()
		if subtle.ConstantTimeCompare([]byte(u+":"+p), []byte(h.Auth)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="scraper"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Access Denied"))
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	// admin actions on root
	if r.URL.Path == "" || r.URL.Path == "/" {
		switch r.Method {
		case http.MethodGet:
			// fall through to write config
		case http.MethodPost:
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write(jsonerr(err))
				return
			}
			if err := h.LoadConfig(b); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write(jsonerr(err))
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write(jsonerr(errors.New("use GET or POST")))
			return
		}
		b, err := json.MarshalIndent(h.Config, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(jsonerr(err))
			return
		}
		w.Write(b)
		return
	}
	// endpoint id (excludes root slash)
	id := r.URL.Path[1:]
	endpoint := h.Endpoint(id)
	if endpoint == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write(jsonerr(fmt.Errorf("endpoint /%s not found", id)))
		return
	}
	// Repeated query params (?tag=a&tag=b) collapse to a comma-joined value
	// (?tag=a,b). The template engine handles URL-escaping when the param sits
	// after the URL's `?`, so the resulting string is still a valid query.
	values := map[string]string{}
	for k, v := range r.URL.Query() {
		values[k] = strings.Join(v, ",")
	}
	res, err := endpoint.Execute(values)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonerr(err))
		return
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	var v any
	if endpoint.List == "" && len(res) == 1 {
		v = res[0]
	} else {
		v = res
	}
	if err := enc.Encode(v); err != nil {
		w.Write([]byte("JSON Error: " + err.Error()))
	}
}

// Endpoint returns the endpoint registered at path, or nil if missing.
func (h *Handler) Endpoint(path string) *Endpoint {
	if e, ok := h.Config[path]; ok {
		return e
	}
	return nil
}
