package domain

import "testing"

func TestNormalizeBody(t *testing.T) {
	body, err := NormalizeBody("  привет  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "привет" {
		t.Fatalf("body = %q, want %q", body, "привет")
	}
	if _, err := NormalizeBody(""); err == nil {
		t.Fatal("empty body must be rejected")
	}
}
