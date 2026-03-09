package semver_test

import (
	"reflect"
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/semver"
)

func TestResolve(t *testing.T) {
	available := []string{"1.24.3", "1.25.0", "1.25.2", "2.0.1"}

	t.Run("major resolves latest patch", func(t *testing.T) {
		got, err := semver.Resolve("1", available)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got != "1.25.2" {
			t.Fatalf("got %q, want 1.25.2", got)
		}
	})

	t.Run("minor resolves latest patch", func(t *testing.T) {
		got, err := semver.Resolve("1.25", available)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got != "1.25.2" {
			t.Fatalf("got %q, want 1.25.2", got)
		}
	})

	t.Run("exact resolves exact", func(t *testing.T) {
		got, err := semver.Resolve("1.24.3", available)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got != "1.24.3" {
			t.Fatalf("got %q, want 1.24.3", got)
		}
	})

	t.Run("nonexistent exact", func(t *testing.T) {
		_, err := semver.Resolve("1.24.4", available)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("nonexistent partial", func(t *testing.T) {
		_, err := semver.Resolve("3.1", available)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestResolve_InvalidInput(t *testing.T) {
	cases := []string{"", "1.", "1..2", "v1.2.3", "1.2.3.4", "1.2.x", "1.2.3+meta"}
	for _, c := range cases {
		if _, err := semver.Resolve(c, []string{"1.2.3"}); err == nil {
			t.Fatalf("Resolve(%q): expected error", c)
		}
	}
}

func TestSortStringsDesc(t *testing.T) {
	input := []string{"1.2.9", "1.11.0", "2.0.0", "1.2.10"}
	semver.SortStringsDesc(input)
	want := []string{"2.0.0", "1.11.0", "1.2.10", "1.2.9"}
	if !reflect.DeepEqual(input, want) {
		t.Fatalf("sorted = %v, want %v", input, want)
	}
}
