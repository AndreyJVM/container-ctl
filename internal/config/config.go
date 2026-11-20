package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultInterval is the standard metrics refresh interval.
	DefaultInterval = 2 * time.Second
	// MinInterval is the minimum allowed refresh interval to prevent excessive resource polling.
	MinInterval = 100 * time.Millisecond
	// DefaultTab is the initial active tab (1 = Docker Containers).
	DefaultTab = 1
	// ConfigFileName is the canonical configuration file name.
	ConfigFileName = "config.yaml"

	// Container sort modes.
	SortDefault = 0
	SortCPU     = 1
	SortMem     = 2
	SortName    = 3
	SortStatus  = 4
)

// Config represents complete application configuration settings.
type Config struct {
	Interval   time.Duration `yaml:"interval"`
	InitialTab int           `yaml:"initial_tab"`
	ShowSwarm  bool          `yaml:"show_swarm"`
	Docker     DockerConfig  `yaml:"docker"`
	UI         UIConfig      `yaml:"ui"`

	// LoadedFrom indicates the file path the configuration was read from, or empty if using defaults.
	LoadedFrom string `yaml:"-"`
}

// DockerConfig contains Docker daemon connection overrides.
type DockerConfig struct {
	SocketPath string `yaml:"socket_path"`
}

// UIConfig contains user interface preferences.
type UIConfig struct {
	OnlyUnusedImages bool   `yaml:"only_unused_images"`
	DefaultSort      string `yaml:"default_sort"`
}

// DefaultConfig returns a new Config with standard default values.
func DefaultConfig() *Config {
	return &Config{
		Interval:   DefaultInterval,
		InitialTab: DefaultTab,
		ShowSwarm:  false,
		Docker: DockerConfig{
			SocketPath: "",
		},
		UI: UIConfig{
			OnlyUnusedImages: false,
			DefaultSort:      "default",
		},
	}
}

// ParseTab resolves a numeric or named tab argument into a 1-indexed tab number.
func ParseTab(val string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "docker", "containers":
		return 1, nil
	case "2", "images", "image":
		return 2, nil
	case "3", "compose":
		return 3, nil
	case "4", "networks", "network":
		return 4, nil
	case "5", "volumes", "volume":
		return 5, nil
	case "6", "swarm":
		return 6, nil
	default:
		return 0, fmt.Errorf("invalid initial tab: %q (allowed: 1|docker, 2|images, 3|compose, 4|networks, 5|volumes, 6|swarm)", val)
	}
}

// TabName returns the canonical name for a given tab number.
func TabName(tab int) string {
	switch tab {
	case 1:
		return "docker"
	case 2:
		return "images"
	case 3:
		return "compose"
	case 4:
		return "networks"
	case 5:
		return "volumes"
	case 6:
		return "swarm"
	default:
		return "unknown"
	}
}

// ParseSortMode parses a string sort mode into integer constant.
func ParseSortMode(val string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "", "default":
		return SortDefault, nil
	case "cpu":
		return SortCPU, nil
	case "mem", "memory":
		return SortMem, nil
	case "name":
		return SortName, nil
	case "status":
		return SortStatus, nil
	default:
		return 0, fmt.Errorf("invalid sort mode %q (allowed: default, cpu, mem, name, status)", val)
	}
}

// SortModeName returns the string representation of a sort mode.
func SortModeName(mode int) string {
	switch mode {
	case SortCPU:
		return "cpu"
	case SortMem:
		return "mem"
	case SortName:
		return "name"
	case SortStatus:
		return "status"
	default:
		return "default"
	}
}

// FindConfigFile searches standard locations for a configuration file.
// If explicitPath is provided, it is returned if it exists.
func FindConfigFile(explicitPath string) string {
	if explicitPath != "" {
		cleaned := filepath.Clean(explicitPath)
		if _, err := os.Stat(cleaned); err == nil {
			return cleaned
		}
		return cleaned
	}

	if envPath := os.Getenv("DOCKPULSE_CONFIG"); envPath != "" {
		cleaned := filepath.Clean(envPath)
		if _, err := os.Stat(cleaned); err == nil { // #nosec G703 - path is cleaned and checking configuration file existence
			return cleaned
		}
	}

	var candidates []string

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "dockpulse", ConfigFileName))
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".config", "dockpulse", ConfigFileName),
			filepath.Join(home, ".dockpulse.yaml"),
		)
	}

	candidates = append(candidates, filepath.Join(".", "dockpulse.yaml"))

	for _, path := range candidates {
		cleaned := filepath.Clean(path)
		if info, err := os.Stat(cleaned); err == nil && !info.IsDir() { // #nosec G703 - path is cleaned and checked for regular file
			return cleaned
		}
	}

	return ""
}

type rawConfigFile struct {
	Interval   interface{}  `yaml:"interval"`
	InitialTab interface{}  `yaml:"initial_tab"`
	ShowSwarm  *bool        `yaml:"show_swarm"`
	Docker     DockerConfig `yaml:"docker"`
	UI         UIConfig     `yaml:"ui"`
}

// LoadConfig loads configuration with priority: Defaults -> File -> Environment Variables.
func LoadConfig(explicitPath string) (*Config, error) {
	cfg := DefaultConfig()

	targetFile := FindConfigFile(explicitPath)
	if targetFile != "" {
		data, err := os.ReadFile(filepath.Clean(targetFile)) // #nosec G304 - validated config file path
		if err != nil {
			if explicitPath != "" {
				return nil, fmt.Errorf("failed to read config file %s: %w", targetFile, err)
			}
		} else {
			var raw rawConfigFile
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", targetFile, err)
			}

			if raw.Interval != nil {
				switch v := raw.Interval.(type) {
				case string:
					d, err := time.ParseDuration(v)
					if err != nil {
						return nil, fmt.Errorf("invalid interval %q in config file: %w", v, err)
					}
					cfg.Interval = d
				case int:
					cfg.Interval = time.Duration(v) * time.Second
				default:
					return nil, fmt.Errorf("unsupported interval type in config: %T", raw.Interval)
				}
			}

			if raw.InitialTab != nil {
				var tabStr string
				switch v := raw.InitialTab.(type) {
				case string:
					tabStr = v
				case int:
					tabStr = strconv.Itoa(v)
				default:
					return nil, fmt.Errorf("unsupported initial_tab type in config: %T", raw.InitialTab)
				}
				tab, err := ParseTab(tabStr)
				if err != nil {
					return nil, fmt.Errorf("config error: %w", err)
				}
				cfg.InitialTab = tab
			}

			if raw.ShowSwarm != nil {
				cfg.ShowSwarm = *raw.ShowSwarm
			}

			if raw.Docker.SocketPath != "" {
				cfg.Docker.SocketPath = raw.Docker.SocketPath
			}

			if raw.UI.DefaultSort != "" {
				if _, err := ParseSortMode(raw.UI.DefaultSort); err != nil {
					return nil, fmt.Errorf("config error: %w", err)
				}
				cfg.UI.DefaultSort = strings.ToLower(raw.UI.DefaultSort)
			}
			cfg.UI.OnlyUnusedImages = raw.UI.OnlyUnusedImages

			cfg.LoadedFrom = targetFile
		}
	}

	// Environment variable overrides
	if envInterval := os.Getenv("DOCKPULSE_INTERVAL"); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			cfg.Interval = d
		}
	}

	if envTab := os.Getenv("DOCKPULSE_TAB"); envTab != "" {
		if tab, err := ParseTab(envTab); err == nil {
			cfg.InitialTab = tab
		}
	}

	if envShowSwarm := os.Getenv("DOCKPULSE_SHOW_SWARM"); envShowSwarm != "" {
		if b, err := strconv.ParseBool(envShowSwarm); err == nil {
			cfg.ShowSwarm = b
		}
	}

	if envSocket := os.Getenv("DOCKPULSE_SOCKET"); envSocket != "" {
		cfg.Docker.SocketPath = envSocket
	}

	if envUnused := os.Getenv("DOCKPULSE_UNUSED_IMAGES"); envUnused != "" {
		if b, err := strconv.ParseBool(envUnused); err == nil {
			cfg.UI.OnlyUnusedImages = b
		}
	}

	if envSort := os.Getenv("DOCKPULSE_SORT"); envSort != "" {
		if _, err := ParseSortMode(envSort); err == nil {
			cfg.UI.DefaultSort = strings.ToLower(envSort)
		}
	}

	if cfg.InitialTab == 6 {
		cfg.ShowSwarm = true
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks whether config values are within permissible ranges.
func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	if cfg.Interval < MinInterval {
		return fmt.Errorf("refresh interval must be at least %v (got: %v)", MinInterval, cfg.Interval)
	}
	if cfg.InitialTab < 1 || cfg.InitialTab > 6 {
		return fmt.Errorf("initial tab must be between 1 and 6 (got: %d)", cfg.InitialTab)
	}
	if _, err := ParseSortMode(cfg.UI.DefaultSort); err != nil {
		return err
	}
	return nil
}

// GenerateDefaultConfig returns a well-commented sample YAML configuration string.
func GenerateDefaultConfig() string {
	return `# Dockpulse Configuration File
# Save this file to ~/.config/dockpulse/config.yaml or ./dockpulse.yaml

# Metrics auto-refresh interval (for example: 1s, 2s, 500ms)
interval: 2s

# Initial active tab: 1|docker, 2|images, 3|compose, 4|networks, 5|volumes, 6|swarm
initial_tab: docker

# Always display the Swarm tab even if cluster is inactive
show_swarm: false

# Docker daemon connection options
docker:
  # Optional custom Unix socket path override (leave empty for auto-detection)
  # socket_path: /var/run/docker.sock
  socket_path: ""

# User interface preferences
ui:
  # Filter images tab to only unreferenced/dangling images by default
  only_unused_images: false

  # Default container sort mode: default, cpu, mem, name, status
  default_sort: default
`
}

// InitConfigFile writes a default configuration file to destination path.
// If destPath is empty, default ~/.config/dockpulse/config.yaml is used.
func InitConfigFile(destPath string, overwrite bool) (string, error) {
	if destPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home directory: %w", err)
		}
		destPath = filepath.Join(home, ".config", "dockpulse", ConfigFileName)
	}

	if _, err := os.Stat(destPath); err == nil && !overwrite {
		return destPath, fmt.Errorf("config file already exists at %s (use --force or remove it)", destPath)
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	content := GenerateDefaultConfig()
	if err := os.WriteFile(destPath, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("failed to write config file %s: %w", destPath, err)
	}

	return destPath, nil
}
