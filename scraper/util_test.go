package scraper

import (
	"strings"
	"testing"
)

func TestTemplate(t *testing.T) {
	tests := []struct {
		name    string
		isURL   bool
		str     string
		vars    map[string]string
		want    string
		wantErr string
	}{
		{
			name: "basic substitution",
			str:  "hello {{name}}",
			vars: map[string]string{"name": "world"},
			want: "hello world",
		},
		{
			name: "default value when var missing",
			str:  "hi {{name:stranger}}",
			vars: map[string]string{},
			want: "hi stranger",
		},
		{
			name: "explicit value beats default",
			str:  "hi {{name:stranger}}",
			vars: map[string]string{"name": "alice"},
			want: "hi alice",
		},
		{
			name:    "missing required var",
			str:     "hi {{name}}",
			vars:    map[string]string{},
			wantErr: "missing param: name",
		},
		{
			name:  "url-escape after question mark",
			isURL: true,
			str:   "https://example.com/search?q={{q}}",
			vars:  map[string]string{"q": "hello world"},
			want:  "https://example.com/search?q=hello+world",
		},
		{
			name:  "no escape before question mark",
			isURL: true,
			str:   "https://example.com/{{path}}?q=x",
			vars:  map[string]string{"path": "a/b"},
			want:  "https://example.com/a/b?q=x",
		},
		{
			name: "first missing param wins",
			str:  "{{a}} and {{b}}",
			vars: map[string]string{},
			// no value substituted on error
			wantErr: "missing param: a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := template(tt.isURL, tt.str, tt.vars)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				if got != "" {
					t.Fatalf("want empty output on error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckSelector(t *testing.T) {
	if err := checkSelector("div.foo > a[href]"); err != nil {
		t.Errorf("valid selector rejected: %v", err)
	}
	// goquery is permissive but should at least not panic on garbage
	_ = checkSelector("[[[")
}
