package auth

import "testing"

func TestManagerCreatesDistinctSessionTokens(t *testing.T) {
	manager := NewManager("secret", "session", "csrf", "X-CSRF-Token", false)

	first, err := manager.NewSessionTokens()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := manager.NewSessionTokens()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.SessionToken == second.SessionToken {
		t.Fatal("session tokens must be distinct")
	}
	if string(first.SessionTokenHash) == string(second.SessionTokenHash) {
		t.Fatal("session token hashes must be distinct")
	}
}

func TestManagerCreatesNumericEmailCode(t *testing.T) {
	manager := NewManager("secret", "session", "csrf", "X-CSRF-Token", false)

	code, err := manager.NewEmailCode(6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("code contains non-digit rune %q", r)
		}
	}
}
