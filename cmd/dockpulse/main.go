package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	configPkg "github.com/AndreyJVM/dockpulse/internal/config"
	"github.com/AndreyJVM/dockpulse/internal/monitor/docker"
	"github.com/AndreyJVM/dockpulse/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

var (
	version = "v0.18.0"
)

const (
	defaultInterval = configPkg.DefaultInterval
	defaultTab      = configPkg.DefaultTab
	minInterval     = configPkg.MinInterval
)

type config struct {
	interval    time.Duration
	initialTab  int
	showVersion bool
	showSwarm   bool
	configFile  string
	initConfig  bool
	printConfig bool
	appConfig   *configPkg.Config
}

var (
	runProgram = func(model tea.Model) error {
		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}
	isTerminal = func(fd uintptr) bool {
		return term.IsTerminal(fd)
	}
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "Dockpulse %s\n\n", version)
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  dockpulse [options]\n\n")
	fmt.Fprintf(w, "Options:\n")
	fmt.Fprintf(w, "  -c, --config <path>        Path to YAML config file (default: ~/.config/dockpulse/config.yaml)\n")
	fmt.Fprintf(w, "  -i, --interval <duration>  Auto-refresh interval (default: %v, env: DOCKPULSE_INTERVAL)\n", defaultInterval)
	fmt.Fprintf(w, "  -t, --tab <1-6|name>       Initial tab: 1|docker, 2|images, 3|compose, 4|networks, 5|volumes, 6|swarm (default: %d, env: DOCKPULSE_TAB)\n", defaultTab)
	fmt.Fprintf(w, "      --show-swarm           Always display Swarm tab even if inactive (env: DOCKPULSE_SHOW_SWARM)\n")
	fmt.Fprintf(w, "      --init-config          Generate a sample config file in ~/.config/dockpulse/config.yaml\n")
	fmt.Fprintf(w, "      --print-config         Print active configuration and exit\n")
	fmt.Fprintf(w, "  -v, --version              Show version\n")
	fmt.Fprintf(w, "  -h, --help                 Show this help\n\n")
	fmt.Fprintf(w, "TUI controls:\n")
	fmt.Fprintf(w, "  1..5 (or 6), Tab, ←/→      Switch tabs\n")
	fmt.Fprintf(w, "  r, R                       Refresh data immediately\n")
	fmt.Fprintf(w, "  ?                          Show keyboard shortcuts\n")
	fmt.Fprintf(w, "  q, Esc, Ctrl+C             Quit\n")
}

func parseFlags(args []string, stdout, stderr io.Writer) (*config, error) {
	// 1. Initial pass to check if -c / --config was specified before loading base config
	var configFilePath string
	var tabFlag string
	var intervalFlag time.Duration
	var showSwarmFlag bool

	flags := flag.NewFlagSet("dockpulse", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		printUsage(stdout)
	}

	cfg := &config{
		interval:   defaultInterval,
		initialTab: defaultTab,
	}

	flags.StringVar(&configFilePath, "config", "", "Path to YAML configuration file")
	flags.StringVar(&configFilePath, "c", "", "Path to YAML configuration file (short)")
	flags.DurationVar(&intervalFlag, "interval", defaultInterval, "Metrics refresh interval (for example: 1s, 2s, 500ms)")
	flags.DurationVar(&intervalFlag, "i", defaultInterval, "Metrics refresh interval (short flag)")
	flags.StringVar(&tabFlag, "tab", "", "Initial tab: 1|docker, 2|images, 3|compose, 4|networks, 5|volumes, 6|swarm")
	flags.StringVar(&tabFlag, "t", "", "Initial tab (short flag)")
	flags.BoolVar(&showSwarmFlag, "show-swarm", false, "Always show Swarm tab even if cluster is inactive")
	flags.BoolVar(&cfg.initConfig, "init-config", false, "Generate a default config file and exit")
	flags.BoolVar(&cfg.printConfig, "print-config", false, "Print active configuration and exit")
	flags.BoolVar(&cfg.showVersion, "version", false, "Show version")
	flags.BoolVar(&cfg.showVersion, "v", false, "Show version (short flag)")

	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	if len(flags.Args()) > 0 {
		return nil, fmt.Errorf("unknown positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	cfg.configFile = configFilePath

	if cfg.initConfig {
		return cfg, nil
	}

	// 2. Load layered config: Defaults -> File -> Environment Variables
	appCfg, err := configPkg.LoadConfig(configFilePath)
	if err != nil {
		return nil, err
	}

	// 3. Command line flags explicitly set by user override all earlier layers
	visited := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	if visited["interval"] || visited["i"] {
		appCfg.Interval = intervalFlag
	}

	if visited["tab"] || visited["t"] {
		tab, err := parseTab(tabFlag)
		if err != nil {
			return nil, err
		}
		appCfg.InitialTab = tab
	}

	if visited["show-swarm"] {
		appCfg.ShowSwarm = showSwarmFlag
	}

	if appCfg.InitialTab == 6 {
		appCfg.ShowSwarm = true
	}

	cfg.interval = appCfg.Interval
	cfg.initialTab = appCfg.InitialTab
	cfg.showSwarm = appCfg.ShowSwarm
	cfg.appConfig = appCfg

	return cfg, nil
}

func parseTab(val string) (int, error) {
	return configPkg.ParseTab(val)
}

func validateConfig(cfg *config) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	if cfg.interval < minInterval {
		return fmt.Errorf("refresh interval must be at least %v (got: %v)", minInterval, cfg.interval)
	}
	if cfg.initialTab < 1 || cfg.initialTab > 6 {
		return fmt.Errorf("initial tab must be between 1 and 6 (got: %d)", cfg.initialTab)
	}
	return nil
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg, err := parseFlags(args, stdout, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if cfg.showVersion {
		fmt.Fprintf(stdout, "dockpulse %s\n", version)
		return nil
	}

	if cfg.initConfig {
		path, initErr := configPkg.InitConfigFile(cfg.configFile, false)
		if initErr != nil {
			return initErr
		}
		fmt.Fprintf(stdout, "Configuration file initialized at: %s\n", path)
		return nil
	}

	if cfg.printConfig {
		if cfg.appConfig.LoadedFrom != "" {
			fmt.Fprintf(stdout, "# Configuration loaded from: %s\n", cfg.appConfig.LoadedFrom)
		} else {
			fmt.Fprintf(stdout, "# Configuration using defaults/environment\n")
		}
		fmt.Fprintf(stdout, "interval: %v\n", cfg.appConfig.Interval)
		fmt.Fprintf(stdout, "initial_tab: %s (%d)\n", configPkg.TabName(cfg.appConfig.InitialTab), cfg.appConfig.InitialTab)
		fmt.Fprintf(stdout, "show_swarm: %t\n", cfg.appConfig.ShowSwarm)
		if cfg.appConfig.Docker.SocketPath != "" {
			fmt.Fprintf(stdout, "docker.socket_path: %s\n", cfg.appConfig.Docker.SocketPath)
		}
		fmt.Fprintf(stdout, "ui.only_unused_images: %t\n", cfg.appConfig.UI.OnlyUnusedImages)
		fmt.Fprintf(stdout, "ui.default_sort: %s\n", cfg.appConfig.UI.DefaultSort)
		return nil
	}

	if err := validateConfig(cfg); err != nil {
		return err
	}

	if !isTerminal(os.Stdin.Fd()) || !isTerminal(os.Stdout.Fd()) {
		return errors.New("dockpulse requires an interactive terminal (TTY)")
	}

	if cfg.appConfig != nil && cfg.appConfig.Docker.SocketPath != "" {
		docker.SetCustomSocketPath(cfg.appConfig.Docker.SocketPath)
	}

	model := tui.NewModel(cfg.interval, cfg.initialTab, version, cfg.showSwarm)

	if cfg.appConfig != nil {
		if cfg.appConfig.UI.OnlyUnusedImages {
			model.SetOnlyUnusedImages(true)
		}
		if sortMode, sErr := configPkg.ParseSortMode(cfg.appConfig.UI.DefaultSort); sErr == nil && sortMode != 0 {
			model.SetContainerSortMode(sortMode)
		}
	}

	return runProgram(model)
}
