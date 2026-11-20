package views

import (
	"strings"
	"testing"
)

func FuzzExtractPortURLs(f *testing.F) {
	// Seed corpus
	f.Add("0.0.0.0:80->80/tcp")
	f.Add("0.0.0.0:443->443/tcp")
	f.Add("0.0.0.0:8080->8080/tcp, :::8080->8080/tcp")
	f.Add("127.0.0.1:3000->3000/tcp, 0.0.0.0:53->53/udp")
	f.Add(":::80->80/tcp")
	f.Add("invalid")
	f.Add("")
	f.Add("0.0.0.0:999999->80/tcp")
	f.Add("[::1]:8080->8080/tcp")
	f.Add(strings.Repeat("0.0.0.0:80->80/tcp,", 50))

	f.Fuzz(func(t *testing.T, portsStr string) {
		urls := ExtractPortURLs(portsStr)
		for _, u := range urls {
			if !strings.HasPrefix(u, "http://localhost") && !strings.HasPrefix(u, "https://localhost") {
				t.Fatalf("ExtractPortURLs produced invalid URL prefix: %s", u)
			}
		}
	})
}

func FuzzFormatPortDisplay(f *testing.F) {
	f.Add("0.0.0.0:80->80/tcp")
	f.Add("0.0.0.0:8080->8080/tcp, :::8080->8080/tcp")
	f.Add(":::443->443/tcp")
	f.Add("invalid")
	f.Add("")
	f.Add("0.0.0.0:1000-1010->1000-1010/tcp")

	f.Fuzz(func(_ *testing.T, portsStr string) {
		// Must never panic on arbitrary input
		_ = FormatPortDisplay(portsStr)
	})
}
