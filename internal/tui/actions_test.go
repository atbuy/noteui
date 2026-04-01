package tui

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTagInputEmpty(t *testing.T) {
	got := parseTagInput("")
	if len(got) != 0 {
		require.Failf(t, "assertion failed", "expected empty slice, got %v", got)
	}
}

func TestParseTagInputSingleTag(t *testing.T) {
	got := parseTagInput("foo")
	if !reflect.DeepEqual(got, []string{"foo"}) {
		require.Failf(t, "assertion failed", "expected [foo], got %v", got)
	}
}

func TestParseTagInputMultipleTags(t *testing.T) {
	got := parseTagInput("foo, bar, baz")
	if !reflect.DeepEqual(got, []string{"foo", "bar", "baz"}) {
		require.Failf(t, "assertion failed", "expected [foo bar baz], got %v", got)
	}
}

func TestParseTagInputStripsHash(t *testing.T) {
	got := parseTagInput("#foo, #bar")
	if !reflect.DeepEqual(got, []string{"foo", "bar"}) {
		require.Failf(t, "assertion failed", "expected [foo bar], got %v", got)
	}
}

func TestParseTagInputTrimsWhitespace(t *testing.T) {
	got := parseTagInput("  foo  ,  bar  ")
	if !reflect.DeepEqual(got, []string{"foo", "bar"}) {
		require.Failf(t, "assertion failed", "expected [foo bar], got %v", got)
	}
}

func TestParseTagInputDeduplicatesCaseInsensitive(t *testing.T) {
	got := parseTagInput("Foo, foo, FOO")
	if len(got) != 1 || got[0] != "Foo" {
		require.Failf(t, "assertion failed", "expected [Foo] (first wins), got %v", got)
	}
}

func TestParseTagInputSkipsEmptyParts(t *testing.T) {
	got := parseTagInput("foo,,bar")
	if !reflect.DeepEqual(got, []string{"foo", "bar"}) {
		require.Failf(t, "assertion failed", "expected [foo bar], got %v", got)
	}
}

func TestParseTagInputOnlyCommas(t *testing.T) {
	got := parseTagInput(",,,,")
	if len(got) != 0 {
		require.Failf(t, "assertion failed", "expected empty slice, got %v", got)
	}
}

func TestParseTagInputHashOnly(t *testing.T) {
	got := parseTagInput("#")
	if len(got) != 0 {
		require.Failf(t, "assertion failed", "expected empty slice for bare #, got %v", got)
	}
}

func TestParseTagInputMixedHashAndPlain(t *testing.T) {
	got := parseTagInput("#work, personal, #urgent")
	if !reflect.DeepEqual(got, []string{"work", "personal", "urgent"}) {
		require.Failf(t, "assertion failed", "expected [work personal urgent], got %v", got)
	}
}
