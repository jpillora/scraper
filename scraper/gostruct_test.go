package scraper

import (
	"strings"
	"testing"
)

func TestExecuteRejectsNonPointer(t *testing.T) {
	type s struct{}
	err := Execute(s{})
	if err == nil || !strings.Contains(err.Error(), "struct pointer") {
		t.Fatalf("want struct-pointer error, got %v", err)
	}
}

func TestExecuteRejectsNonStructPointer(t *testing.T) {
	x := 42
	err := Execute(&x)
	if err == nil || !strings.Contains(err.Error(), "struct pointer") {
		t.Fatalf("want struct-pointer error, got %v", err)
	}
}

func TestExecuteRejectsNilString(t *testing.T) {
	// Pre-fix: this would panic via reflect.Elem() on a string Kind.
	var s *string
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Execute panicked on nil pointer: %v", r)
		}
	}()
	_ = Execute(s) // should return an error, not panic
}
