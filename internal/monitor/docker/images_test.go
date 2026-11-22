package docker

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetDockerImagesViaSocket(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		t.Fatalf("CLI must not run after a successful socket request: %v", args)
		return nil, errors.New("unexpected CLI call")
	}
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/images/json" {
				return mockStringResponse(http.StatusOK, `[
					{
						"Id": "sha256:c557bd415052858081f023e8d17500073317df437ae79ea4d2c4077cd9a1a6ae",
						"RepoTags": ["sambryx-web:latest", "sambryx-web:v1.0"],
						"RepoDigests": ["sambryx-web@sha256:1234567890abcdef1234567890abcdef"],
						"Created": 1731760000,
						"Size": 359214108,
						"Containers": 1
					},
					{
						"Id": "sha256:a857ab57fe2d98a0a33432586f16a6b0d2e256ed9d915ec56ecfaaeb81ac36bd",
						"RepoTags": null,
						"RepoDigests": [],
						"Created": 1731750000,
						"Size": 1048576,
						"Containers": 0
					}
				]`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	images, err := GetDockerImages()
	if err != nil {
		t.Fatalf("GetDockerImages() returned an error: %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	// First should be non-dangling: sambryx-web
	img1 := images[0]
	if img1.ID != "c557bd415052" {
		t.Errorf("expected ID c557bd415052, got %q", img1.ID)
	}
	if img1.Repository != "sambryx-web" {
		t.Errorf("expected Repository sambryx-web, got %q", img1.Repository)
	}
	if img1.Tag != "latest (+1)" {
		t.Errorf("expected Tag 'latest (+1)', got %q", img1.Tag)
	}
	if !img1.InUse || img1.Containers != 1 {
		t.Errorf("expected InUse=true, Containers=1, got InUse=%t, Containers=%d", img1.InUse, img1.Containers)
	}
	if img1.Digest != "1234567890ab" {
		t.Errorf("expected clean short digest 1234567890ab, got %q", img1.Digest)
	}

	// Second should be dangling
	img2 := images[1]
	if !img2.Dangling {
		t.Errorf("expected Dangling=true for null RepoTags")
	}
	if img2.Repository != "<none>" || img2.Tag != "<none>" {
		t.Errorf("expected <none>:<none>, got %s:%s", img2.Repository, img2.Tag)
	}
}

func TestGetDockerImagesCLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] == "image" && args[1] == "ls" {
			output := "sha256:1122334455667788|nginx|alpine|94.5MB|2026-11-05 22:55:10 +0300 MSK|sha256:fed7e14be83cbd2661a75cdc5b872d936cf155d46c327084d30b8df19ae88d72\n" +
				"sha256:9988776655443322|<none>|<none>|120MB|2026-10-02 11:10:55 +0300 MSK|<none>\n"
			return []byte(output), nil
		}
		return nil, errors.New("unexpected command")
	}

	images, err := GetDockerImages()
	if err != nil {
		t.Fatalf("GetDockerImages() returned an error: %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].Repository != "nginx" || images[0].Tag != "alpine" {
		t.Errorf("unexpected first image: %#v", images[0])
	}
	if images[0].Size <= 0 {
		t.Errorf("expected positive parsed size, got %d", images[0].Size)
	}
	if images[1].Repository != "<none>" || !images[1].Dangling {
		t.Errorf("unexpected second image: %#v", images[1])
	}
}

func TestGetDockerImagesError(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return nil, errors.New("docker daemon is not running")
	}

	_, err := GetDockerImages()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveDockerImage_Socket(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	var requestedURL string
	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			requestedURL = req.URL.String()
			if req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/images/") {
				return mockStringResponse(http.StatusOK, `[{"Untagged":"test:latest"}]`), nil
			}
			return mockStringResponse(http.StatusBadRequest, "bad request"), nil
		})
	}

	if err := RemoveDockerImage("test-img-id", true); err != nil {
		t.Fatalf("RemoveDockerImage failed: %v", err)
	}
	if !strings.Contains(requestedURL, "/images/test-img-id?force=true") {
		t.Errorf("unexpected URL requested: %s", requestedURL)
	}
}

func TestRemoveDockerImage_CLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	var ranArgs []string
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		ranArgs = args
		return []byte("Untagged: nginx:alpine\nDeleted: sha256:123\n"), nil
	}

	if err := RemoveDockerImage("nginx:alpine", false); err != nil {
		t.Fatalf("RemoveDockerImage CLI failed: %v", err)
	}
	if len(ranArgs) != 2 || ranArgs[0] != "rmi" || ranArgs[1] != "nginx:alpine" {
		t.Errorf("unexpected CLI args: %v", ranArgs)
	}

	// With force
	if err := RemoveDockerImage("nginx:alpine", true); err != nil {
		t.Fatalf("RemoveDockerImage CLI with force failed: %v", err)
	}
	if len(ranArgs) != 3 || ranArgs[0] != "rmi" || ranArgs[1] != "-f" || ranArgs[2] != "nginx:alpine" {
		t.Errorf("unexpected CLI args with force: %v", ranArgs)
	}
}

func TestParseSizeToBytes(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"", -1},
		{"-", -1},
		{"invalid", -1},
		{"100B", 100},
		{"1KB", 1024},
		{"10MB", 10 * 1024 * 1024},
		{"2.5GB", int64(2.5 * 1024 * 1024 * 1024)},
		{"1TB", 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		got := parseSizeToBytes(tt.input)
		if got != tt.want {
			t.Errorf("parseSizeToBytes(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseRepoTags(t *testing.T) {
	tests := []struct {
		name         string
		tags         []string
		wantRepo     string
		wantTag      string
		wantDangling bool
	}{
		{
			name:         "empty tags",
			tags:         nil,
			wantRepo:     "<none>",
			wantTag:      "<none>",
			wantDangling: true,
		},
		{
			name:         "dangling tags",
			tags:         []string{"<none>:<none>"},
			wantRepo:     "<none>",
			wantTag:      "<none>",
			wantDangling: true,
		},
		{
			name:         "standard repo and tag",
			tags:         []string{"nginx:alpine"},
			wantRepo:     "nginx",
			wantTag:      "alpine",
			wantDangling: false,
		},
		{
			name:         "multiple tags",
			tags:         []string{"redis:7.0", "redis:latest"},
			wantRepo:     "redis",
			wantTag:      "7.0 (+1)",
			wantDangling: false,
		},
		{
			name:         "host with port and no tag",
			tags:         []string{"localhost:5000/my-app"},
			wantRepo:     "localhost:5000/my-app",
			wantTag:      "<none>",
			wantDangling: false,
		},
		{
			name:         "host with port and tag",
			tags:         []string{"registry.example.com:8443/team/service:v2.1.0"},
			wantRepo:     "registry.example.com:8443/team/service",
			wantTag:      "v2.1.0",
			wantDangling: false,
		},
		{
			name:         "repo without tag",
			tags:         []string{"my-service"},
			wantRepo:     "my-service",
			wantTag:      "<none>",
			wantDangling: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepo, gotTag, gotDangling := parseRepoTags(tt.tags)
			if gotRepo != tt.wantRepo || gotTag != tt.wantTag || gotDangling != tt.wantDangling {
				t.Errorf("parseRepoTags(%v) = (%q, %q, %v); want (%q, %q, %v)",
					tt.tags, gotRepo, gotTag, gotDangling, tt.wantRepo, tt.wantTag, tt.wantDangling)
			}
		})
	}
}

