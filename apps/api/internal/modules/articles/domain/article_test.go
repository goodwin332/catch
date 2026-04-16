package domain

import (
	"encoding/json"
	"testing"
)

func TestValidateDocument(t *testing.T) {
	doc, err := ValidateDocument(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(doc) {
		t.Fatal("default document must be valid json")
	}

	if _, err := ValidateDocument(json.RawMessage(`[]`)); err == nil {
		t.Fatal("array document must be rejected")
	}
}

func TestNormalizeTags(t *testing.T) {
	tags, err := NormalizeTags([]string{"#Спиннинг", " спиннинг ", "Лодка"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("tags len = %d, want 2", len(tags))
	}
}
