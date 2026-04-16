package mail

import (
	"bytes"
	"testing"
)

func TestBuildMessageEncodesUTF8SubjectAndBody(t *testing.T) {
	data := buildMessage("Catch <noreply@catch.local>", "user@example.com", Message{
		Subject: "Код входа в Catch",
		Text:    "Ваш код: 123456",
	})
	if !bytes.Contains(data, []byte("Content-Type: text/plain; charset=utf-8")) {
		t.Fatalf("message = %s, want utf-8 text content type", data)
	}
	if !bytes.Contains(data, []byte("=?utf-8?")) {
		t.Fatalf("message = %s, want encoded subject", data)
	}
	if !bytes.Contains(data, []byte("Ваш код: 123456")) {
		t.Fatalf("message = %s, want body", data)
	}
}
