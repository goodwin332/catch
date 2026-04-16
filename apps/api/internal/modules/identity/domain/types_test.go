package domain

import "testing"

func TestNewEmailNormalizesValue(t *testing.T) {
	email, err := NewEmail("  User@Example.COM ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if email.String() != "user@example.com" {
		t.Fatalf("email = %q, want %q", email.String(), "user@example.com")
	}
}

func TestNewEmailRejectsInvalidValue(t *testing.T) {
	if _, err := NewEmail("not-email"); err == nil {
		t.Fatal("expected invalid email to be rejected")
	}
}
