package testutil

import (
	"testing"
)

type mockTB struct {
	testing.TB
	failed bool
	msg    string
}

func (m *mockTB) Helper() {}

func (m *mockTB) Fatalf(format string, _ ...any) {
	m.failed = true
	m.msg = format
}

func TestAssertContainsAll(t *testing.T) {
	mock := &mockTB{}
	AssertContainsAll(mock, "hello beautiful world", "hello", "world")
	if mock.failed {
		t.Errorf("expected pass, but mock failed: %s", mock.msg)
	}

	mockFail := &mockTB{}
	AssertContainsAll(mockFail, "hello beautiful world", "hello", "missing")
	if !mockFail.failed {
		t.Errorf("expected failure when substring missing, but passed")
	}
}

func TestAssertNotContains(t *testing.T) {
	mock := &mockTB{}
	AssertNotContains(mock, "hello world", "foo", "bar")
	if mock.failed {
		t.Errorf("expected pass, but mock failed: %s", mock.msg)
	}

	mockFail := &mockTB{}
	AssertNotContains(mockFail, "hello world", "foo", "world")
	if !mockFail.failed {
		t.Errorf("expected failure when forbidden substring present, but passed")
	}
}

func TestAssertContainsInOrder(t *testing.T) {
	mock := &mockTB{}
	AssertContainsInOrder(mock, "first middle last", "first", "middle", "last")
	if mock.failed {
		t.Errorf("expected pass, but mock failed: %s", mock.msg)
	}

	mockFailOrder := &mockTB{}
	AssertContainsInOrder(mockFailOrder, "first middle last", "last", "first")
	if !mockFailOrder.failed {
		t.Errorf("expected failure when order is reversed, but passed")
	}

	mockFailMissing := &mockTB{}
	AssertContainsInOrder(mockFailMissing, "first middle last", "first", "missing")
	if !mockFailMissing.failed {
		t.Errorf("expected failure when item missing, but passed")
	}
}
