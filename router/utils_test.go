package router

import (
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Test@example.com  ", "test@example.com"},
		{"USER@DOMAIN.COM", "user@domain.com"},
		{"simple@example.com", "simple@example.com"},
		{"MixedCase@ExAmPlE.CoM", "mixedcase@example.com"},
		{"  spaces  ", "spaces"},
	}

	for _, test := range tests {
		result := normalizeEmail(test.input)
		if result != test.expected {
			t.Errorf("normalizeEmail(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
