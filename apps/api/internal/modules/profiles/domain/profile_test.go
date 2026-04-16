package domain

import "testing"

func TestNormalizeUsername(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "empty allowed", in: " ", want: "", ok: true},
		{name: "lowercases", in: "  User_Name-1 ", want: "user_name-1", ok: true},
		{name: "too short", in: "ab", want: "ab", ok: false},
		{name: "invalid chars", in: "bad.name", want: "bad.name", ok: false},
		{name: "cannot start with dash", in: "-bad", want: "-bad", ok: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeUsername(tt.in)
			if got != tt.want {
				t.Fatalf("got username %q, want %q", got, tt.want)
			}
			if ok != tt.ok {
				t.Fatalf("got ok %v, want %v", ok, tt.ok)
			}
		})
	}
}
