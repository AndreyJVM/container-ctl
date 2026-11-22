package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Interval != DefaultInterval {
		t.Errorf("expected interval %v, got %v", DefaultInterval, cfg.Interval)
	}
	if cfg.InitialTab != DefaultTab {
		t.Errorf("expected initial tab %d, got %d", DefaultTab, cfg.InitialTab)
	}
	if cfg.ShowSwarm {
		t.Errorf("expected show_swarm to be false")
	}
	if cfg.Docker.SocketPath != "" {
		t.Errorf("expected empty socket path, got %q", cfg.Docker.SocketPath)
	}
	if cfg.UI.OnlyUnusedImages {
		t.Errorf("expected only_unused_images false")
	}
	if cfg.UI.DefaultSort != "default" {
		t.Errorf("expected default sort %q, got %q", "default", cfg.UI.DefaultSort)
	}
}

func TestParseTab(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"1", 1, false},
		{"docker", 1, false},
		{"containers", 1, false},
		{"2", 2, false},
		{"images", 2, false},
		{"image", 2, false},
		{"3", 3, false},
		{"compose", 3, false},
		{"4", 4, false},
		{"networks", 4, false},
		{"network", 4, false},
		{"5", 5, false},
		{"volumes", 5, false},
		{"volume", 5, false},
		{"6", 6, false},
		{"swarm", 6, false},
		{"0", 0, true},
		{"7", 0, true},
		{"unknown", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseTab(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTab(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("ParseTab(%q) = %d, want = %d", tt.input, got, tt.want)
		}
	}
}

func TestTabName(t *testing.T) {
	if got := TabName(1); got != "docker" {
		t.Errorf("TabName(1) = %q, want 'docker'", got)
	}
	if got := TabName(6); got != "swarm" {
		t.Errorf("TabName(6) = %q, want 'swarm'", got)
	}
	if got := TabName(99); got != "unknown" {
		t.Errorf("TabName(99) = %q, want 'unknown'", got)
	}
}

func TestParseSortMode(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"", SortDefault, false},
		{"default", SortDefault, false},
		{"cpu", SortCPU, false},
		{"mem", SortMem, false},
		{"memory", SortMem, false},
		{"name", SortName, false},
		{"status", SortStatus, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseSortMode(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSortMode(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("ParseSortMode(%q) = %d, want = %d", tt.input, got, tt.want)
		}
	}
}

func TestSortModeName(t *testing.T) {
	if SortModeName(SortCPU) != "cpu" {
		t.Errorf("expected 'cpu', got %s", SortModeName(SortCPU))
	}
	if SortModeName(SortMem) != "mem" {
		t.Errorf("expected 'mem', got %s", SortModeName(SortMem))
	}
	if SortModeName(SortName) != "name" {
		t.Errorf("expected 'name', got %s", SortModeName(SortName))
	}
	if SortModeName(SortStatus) != "status" {
		t.Errorf("expected 'status', got %s", SortModeName(SortStatus))
	}
	if SortModeName(SortDefault) != "default" {
		t.Errorf("expected 'default', got %s", SortModeName(SortDefault))
	}
}

func TestLoadConfig_YAMLFile(t *testing.T) {
	yamlContent := `
interval: 1500ms
initial_tab: compose
show_swarm: true
docker:
  socket_path: /custom/docker.sock
ui:
  only_unused_images: true
  default_sort: cpu
`
	tmpFile, err := os.CreateTemp("", "container-ctl-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected LoadConfig error: %v", err)
	}

	if cfg.Interval != 1500*time.Millisecond {
		t.Errorf("expected 1500ms interval, got %v", cfg.Interval)
	}
	if cfg.InitialTab != 3 {
		t.Errorf("expected tab 3 (compose), got %d", cfg.InitialTab)
	}
	if !cfg.ShowSwarm {
		t.Errorf("expected show_swarm true")
	}
	if cfg.Docker.SocketPath != "/custom/docker.sock" {
		t.Errorf("expected /custom/docker.sock, got %q", cfg.Docker.SocketPath)
	}
	if !cfg.UI.OnlyUnusedImages {
		t.Errorf("expected only_unused_images true")
	}
	if cfg.UI.DefaultSort != "cpu" {
		t.Errorf("expected default_sort 'cpu', got %q", cfg.UI.DefaultSort)
	}
	if cfg.LoadedFrom != tmpFile.Name() {
		t.Errorf("expected LoadedFrom %q, got %q", tmpFile.Name(), cfg.LoadedFrom)
	}
}

func TestLoadConfig_NumericTabAndInterval(t *testing.T) {
	yamlContent := `
interval: 4
initial_tab: 2
show_swarm: false
`
	tmpFile, err := os.CreateTemp("", "container-ctl-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected LoadConfig error: %v", err)
	}

	if cfg.Interval != 4*time.Second {
		t.Errorf("expected 4s interval, got %v", cfg.Interval)
	}
	if cfg.InitialTab != 2 {
		t.Errorf("expected tab 2, got %d", cfg.InitialTab)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	yamlContent := `
interval: 5s
initial_tab: 1
show_swarm: false
docker:
  socket_path: /file.sock
ui:
  only_unused_images: false
  default_sort: name
`
	tmpFile, err := os.CreateTemp("", "container-ctl-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Set environment variables to override file
	origInterval := os.Getenv("DOCKPULSE_INTERVAL")
	origTab := os.Getenv("DOCKPULSE_TAB")
	origSwarm := os.Getenv("DOCKPULSE_SHOW_SWARM")
	origSocket := os.Getenv("DOCKPULSE_SOCKET")
	origUnused := os.Getenv("DOCKPULSE_UNUSED_IMAGES")
	origSort := os.Getenv("DOCKPULSE_SORT")
	defer func() {
		os.Setenv("DOCKPULSE_INTERVAL", origInterval)
		os.Setenv("DOCKPULSE_TAB", origTab)
		os.Setenv("DOCKPULSE_SHOW_SWARM", origSwarm)
		os.Setenv("DOCKPULSE_SOCKET", origSocket)
		os.Setenv("DOCKPULSE_UNUSED_IMAGES", origUnused)
		os.Setenv("DOCKPULSE_SORT", origSort)
	}()

	os.Setenv("DOCKPULSE_INTERVAL", "1s")
	os.Setenv("DOCKPULSE_TAB", "volumes")
	os.Setenv("DOCKPULSE_SHOW_SWARM", "true")
	os.Setenv("DOCKPULSE_SOCKET", "/env.sock")
	os.Setenv("DOCKPULSE_UNUSED_IMAGES", "true")
	os.Setenv("DOCKPULSE_SORT", "status")

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected LoadConfig error: %v", err)
	}

	if cfg.Interval != 1*time.Second {
		t.Errorf("expected 1s from env, got %v", cfg.Interval)
	}
	if cfg.InitialTab != 5 {
		t.Errorf("expected tab 5 (volumes) from env, got %d", cfg.InitialTab)
	}
	if !cfg.ShowSwarm {
		t.Errorf("expected show_swarm true from env")
	}
	if cfg.Docker.SocketPath != "/env.sock" {
		t.Errorf("expected /env.sock from env, got %q", cfg.Docker.SocketPath)
	}
	if !cfg.UI.OnlyUnusedImages {
		t.Errorf("expected only_unused_images true from env")
	}
	if cfg.UI.DefaultSort != "status" {
		t.Errorf("expected default_sort 'status' from env, got %q", cfg.UI.DefaultSort)
	}
}

func TestLoadConfig_Errors(t *testing.T) {
	t.Run("non-existent explicit file", func(t *testing.T) {
		_, err := LoadConfig("/non/existent/path/config.yaml")
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid YAML syntax", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "bad-yaml-*.yaml")
		if err != nil {
			t.Fatalf("failed to create temp: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("invalid: [yaml: broken")
		tmpFile.Close()

		_, err = LoadConfig(tmpFile.Name())
		if err == nil {
			t.Fatal("expected YAML parse error, got nil")
		}
	})

	t.Run("invalid interval", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "bad-interval-*.yaml")
		if err != nil {
			t.Fatalf("failed to create temp: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("interval: not-a-duration\n")
		tmpFile.Close()

		_, err = LoadConfig(tmpFile.Name())
		if err == nil {
			t.Fatal("expected error for invalid interval, got nil")
		}
	})

	t.Run("interval below minimum", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "low-interval-*.yaml")
		if err != nil {
			t.Fatalf("failed to create temp: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("interval: 20ms\n")
		tmpFile.Close()

		_, err = LoadConfig(tmpFile.Name())
		if err == nil {
			t.Fatal("expected validation error for interval < 100ms, got nil")
		}
	})
}

func TestInitConfigFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "container-ctl-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	target := filepath.Join(tempDir, "sub", "config.yaml")

	// 1. First creation should succeed
	createdPath, err := InitConfigFile(target, false)
	if err != nil {
		t.Fatalf("InitConfigFile failed: %v", err)
	}
	if createdPath != target {
		t.Errorf("expected %s, got %s", target, createdPath)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if !strings.Contains(string(data), "Dockpulse Configuration File") {
		t.Errorf("file does not contain expected header")
	}

	// 2. Second creation without overwrite should error
	_, err = InitConfigFile(target, false)
	if err == nil {
		t.Fatal("expected error when file already exists, got nil")
	}

	// 3. Second creation with overwrite should succeed
	_, err = InitConfigFile(target, true)
	if err != nil {
		t.Fatalf("InitConfigFile with overwrite failed: %v", err)
	}
}

func TestFindConfigFile(t *testing.T) {
	// Explicit path that doesn't exist
	if got := FindConfigFile("/custom/path.yaml"); got != "/custom/path.yaml" {
		t.Errorf("expected explicit path returned, got %s", got)
	}

	// DOCKPULSE_CONFIG env var
	tmpFile, err := os.CreateTemp("", "container-ctl-env-*.yaml")
	if err != nil {
		t.Fatalf("temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	origEnv := os.Getenv("DOCKPULSE_CONFIG")
	defer os.Setenv("DOCKPULSE_CONFIG", origEnv)
	os.Setenv("DOCKPULSE_CONFIG", tmpFile.Name())

	if got := FindConfigFile(""); got != tmpFile.Name() {
		t.Errorf("expected %s from DOCKPULSE_CONFIG, got %s", tmpFile.Name(), got)
	}
}

func TestValidate_Errors(t *testing.T) {
	if err := Validate(nil); err == nil {
		t.Errorf("expected error on nil config")
	}

	cfg := DefaultConfig()
	cfg.InitialTab = 0
	if err := Validate(cfg); err == nil {
		t.Errorf("expected error on initial tab 0")
	}

	cfg.InitialTab = 7
	if err := Validate(cfg); err == nil {
		t.Errorf("expected error on initial tab 7")
	}

	cfg.InitialTab = 1
	cfg.UI.DefaultSort = "invalid_sort_mode"
	if err := Validate(cfg); err == nil {
		t.Errorf("expected error on invalid sort mode")
	}
}

func TestTabName_Unknown(t *testing.T) {
	if name := TabName(99); name != "unknown" {
		t.Errorf("expected 'unknown' for invalid tab ID, got %s", name)
	}
	if name := TabName(1); name != "docker" {
		t.Errorf("expected 'docker', got %s", name)
	}
	if name := TabName(4); name != "networks" {
		t.Errorf("expected 'networks', got %s", name)
	}
	if name := TabName(6); name != "swarm" {
		t.Errorf("expected 'swarm', got %s", name)
	}
}

func TestLoadConfig_TypesAndShowSwarm(t *testing.T) {
	yamlContent := `
interval: [unsupported, list]
`
	tmpFile, err := os.CreateTemp("", "bad-type-*.yaml")
	if err != nil {
		t.Fatalf("temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yamlContent)
	tmpFile.Close()

	if _, err := LoadConfig(tmpFile.Name()); err == nil {
		t.Errorf("expected error on unsupported interval type")
	}

	yamlTabBadType := `
initial_tab: [bad]
`
	tmpFile2, _ := os.CreateTemp("", "bad-tab-*.yaml")
	defer os.Remove(tmpFile2.Name())
	tmpFile2.WriteString(yamlTabBadType)
	tmpFile2.Close()

	if _, err := LoadConfig(tmpFile2.Name()); err == nil {
		t.Errorf("expected error on unsupported initial_tab type")
	}

	yamlTabBadVal := `
initial_tab: "not_a_valid_tab"
`
	tmpFile3, _ := os.CreateTemp("", "bad-tab-val-*.yaml")
	defer os.Remove(tmpFile3.Name())
	tmpFile3.WriteString(yamlTabBadVal)
	tmpFile3.Close()

	if _, err := LoadConfig(tmpFile3.Name()); err == nil {
		t.Errorf("expected error on invalid initial_tab string")
	}

	yamlBadSort := `
ui:
  default_sort: "unknown-sort-mode"
`
	tmpFile4, _ := os.CreateTemp("", "bad-sort-*.yaml")
	defer os.Remove(tmpFile4.Name())
	tmpFile4.WriteString(yamlBadSort)
	tmpFile4.Close()

	if _, err := LoadConfig(tmpFile4.Name()); err == nil {
		t.Errorf("expected error on invalid default_sort")
	}

	yamlTabSwarm := `
initial_tab: swarm
`
	tmpFile5, _ := os.CreateTemp("", "swarm-tab-*.yaml")
	defer os.Remove(tmpFile5.Name())
	tmpFile5.WriteString(yamlTabSwarm)
	tmpFile5.Close()

	cfg, err := LoadConfig(tmpFile5.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.ShowSwarm {
		t.Errorf("expected ShowSwarm to be true when initial_tab is swarm (6)")
	}
}

func TestFindConfigFile_IgnoresDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "container-ctl-dir-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origEnv := os.Getenv("DOCKPULSE_CONFIG")
	defer os.Setenv("DOCKPULSE_CONFIG", origEnv)
	os.Setenv("DOCKPULSE_CONFIG", tempDir)

	if got := FindConfigFile(""); got == tempDir {
		t.Errorf("FindConfigFile should not return a directory: got %s", got)
	}
}

func TestLoadConfig_FloatAndInt64(t *testing.T) {
	yamlContent := `
interval: 1.5
initial_tab: 2
`
	tmpFile, err := os.CreateTemp("", "float-interval-*.yaml")
	if err != nil {
		t.Fatalf("temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yamlContent)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error loading float interval: %v", err)
	}
	if cfg.Interval != 1500*time.Millisecond {
		t.Errorf("expected interval 1.5s (1500ms), got: %v", cfg.Interval)
	}
	if cfg.InitialTab != 2 {
		t.Errorf("expected initial tab 2, got: %d", cfg.InitialTab)
	}
}

