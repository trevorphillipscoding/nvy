package verutil_test

import (
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/verutil"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"20", "20.0.0"},
		{"1.26", "1.26.0"},
		{"3.12.5", "3.12.5"},
		{"3.12+20240814", "3.12.0+20240814"},
		{"1.22.1", "1.22.1"},
		{"20.11.1", "20.11.1"},
		{"3", "3.0.0"},
		{"1.0+tag", "1.0.0+tag"},
	}
	for _, c := range cases {
		got := verutil.Normalize(c.input)
		if got != c.want {
			t.Errorf("Normalize(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}
