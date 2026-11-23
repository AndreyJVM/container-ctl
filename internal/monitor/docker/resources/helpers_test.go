package resources

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/engine"
)

func TestTruncateResourceID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"1234567890", "1234567890"},
		{"123456789012", "123456789012"},
		{"1234567890123456", "123456789012"},
		{"  abcdef123456789  ", "abcdef123456"},
	}

	for _, tt := range tests {
		got := TruncateResourceID(tt.input)
		if got != tt.expected {
			t.Errorf("TruncateResourceID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"t", true},
		{"false", false},
		{"0", false},
		{"", false},
		{"notabool", false},
		{"  true  ", true},
	}

	for _, tt := range tests {
		got := ParseBool(tt.input)
		if got != tt.expected {
			t.Errorf("ParseBool(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestNonEmptyLines(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"   \n\n  \t  \n", nil},
		{"line1\nline2", []string{"line1", "line2"}},
		{" line1 \n\n  line2  \n line3\n", []string{"line1", "line2", "line3"}},
	}

	for _, tt := range tests {
		got := NonEmptyLines(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("NonEmptyLines(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestLabelValue(t *testing.T) {
	tests := []struct {
		labels   string
		key      string
		expected string
	}{
		{"", "app", ""},
		{"app=frontend,env=prod", "app", "frontend"},
		{"app=frontend,env=prod", "env", "prod"},
		{"app=frontend,env=prod", "missing", ""},
		{"key_only,foo=bar", "foo", "bar"},
		{"key_only", "key_only", ""},
		{"  app=api , env=stage  ", "env", "stage"},
	}

	for _, tt := range tests {
		got := LabelValue(tt.labels, tt.key)
		if got != tt.expected {
			t.Errorf("LabelValue(%q, %q) = %q, want %q", tt.labels, tt.key, got, tt.expected)
		}
	}
}

func TestFetchDockerResourceJSON(t *testing.T) {
	origFactory := engine.SocketClientFactory
	defer func() { engine.SocketClientFactory = origFactory }()

	t.Run("successful decode", func(t *testing.T) {
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/test" {
					return mockStringResponse(http.StatusOK, `{"name":"test-resource"}`), nil
				}
				return mockStringResponse(http.StatusNotFound, "not found"), nil
			})
		}

		var target struct {
			Name string `json:"name"`
		}
		err := engine.FetchDockerResourceJSON("/tmp/test.sock", "/test", &target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if target.Name != "test-resource" {
			t.Errorf("expected target.Name 'test-resource', got %q", target.Name)
		}
	})

	t.Run("http client error", func(t *testing.T) {
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("socket dial error")
			})
		}

		var target map[string]interface{}
		err := engine.FetchDockerResourceJSON("/tmp/test.sock", "/test", &target)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("non-200 status code", func(t *testing.T) {
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return mockStringResponse(http.StatusInternalServerError, "error"), nil
			})
		}

		var target map[string]interface{}
		err := engine.FetchDockerResourceJSON("/tmp/test.sock", "/test", &target)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
