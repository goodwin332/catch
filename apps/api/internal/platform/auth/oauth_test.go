package auth

import (
	"testing"
	"time"
)

func TestOAuthStateIsSignedAndVerified(t *testing.T) {
	manager := NewManager("test-secret", "session", "csrf", "X-CSRF-Token", false)
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	state, err := manager.NewOAuthState("google", "/articles", time.Minute, now)
	if err != nil {
		t.Fatalf("create state: %v", err)
	}
	value, err := manager.SignOAuthState(state)
	if err != nil {
		t.Fatalf("sign state: %v", err)
	}

	verified, err := manager.VerifyOAuthState(value, "google", state.State, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("verify state: %v", err)
	}
	if verified.Provider != "google" || verified.ReturnTo != "/articles" || verified.CodeVerifier == "" || verified.CodeChallenge == "" {
		t.Fatalf("verified state = %+v", verified)
	}
}

func TestOAuthStateRejectsTampering(t *testing.T) {
	manager := NewManager("test-secret", "session", "csrf", "X-CSRF-Token", false)
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	state, err := manager.NewOAuthState("google", "/", time.Minute, now)
	if err != nil {
		t.Fatalf("create state: %v", err)
	}
	value, err := manager.SignOAuthState(state)
	if err != nil {
		t.Fatalf("sign state: %v", err)
	}

	if _, err := manager.VerifyOAuthState(value+"x", "google", state.State, now); err == nil {
		t.Fatal("tampered state must be rejected")
	}
	if _, err := manager.VerifyOAuthState(value, "yandex", state.State, now); err == nil {
		t.Fatal("provider mismatch must be rejected")
	}
	if _, err := manager.VerifyOAuthState(value, "google", "other-state", now); err == nil {
		t.Fatal("state token mismatch must be rejected")
	}
	if _, err := manager.VerifyOAuthState(value, "google", state.State, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expired state must be rejected")
	}
}
