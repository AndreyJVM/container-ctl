package testutil

import (
	"strings"
	"testing"
)

// AssertContainsAll verifies that the target string contains all provided substrings.
func AssertContainsAll(t testing.TB, target string, substrings ...string) {
	t.Helper()
	var missing []string
	for _, sub := range substrings {
		if !strings.Contains(target, sub) {
			missing = append(missing, sub)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("target output does not contain expected substring(s):\nMissing: %q\nFull target:\n%s", missing, target)
	}
}

// AssertNotContains verifies that the target string contains none of the provided substrings.
func AssertNotContains(t testing.TB, target string, substrings ...string) {
	t.Helper()
	var unexpected []string
	for _, sub := range substrings {
		if strings.Contains(target, sub) {
			unexpected = append(unexpected, sub)
		}
	}
	if len(unexpected) > 0 {
		t.Fatalf("target output contains unexpected substring(s):\nUnexpected: %q\nFull target:\n%s", unexpected, target)
	}
}

// AssertContainsInOrder verifies that all substrings appear in target in sequential order.
func AssertContainsInOrder(t testing.TB, target string, orderedSubstrings ...string) {
	t.Helper()
	lastIdx := 0
	for _, sub := range orderedSubstrings {
		idx := strings.Index(target[lastIdx:], sub)
		if idx == -1 {
			t.Fatalf("expected substring %q after position %d in order, but was not found\nTarget:\n%s", sub, lastIdx, target)
		}
		lastIdx += idx + len(sub)
	}
}
