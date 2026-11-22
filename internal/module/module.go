package module

import (
	"fmt"
	"strings"
)

// ID represents a unique identifier for a module.
type ID string

const (
	// IDContainers identifies the Docker containers management module.
	IDContainers ID = "containers"
	// IDImages identifies the Docker images management module.
	IDImages ID = "images"
	// IDCompose identifies the Docker Compose projects module.
	IDCompose ID = "compose"
	// IDNetworks identifies the Docker networks inspection module.
	IDNetworks ID = "networks"
	// IDVolumes identifies the Docker volumes management module.
	IDVolumes ID = "volumes"
	// IDSwarm identifies the Docker Swarm cluster module.
	IDSwarm ID = "swarm"
)

// Config represents per-module user configuration options.
type Config struct {
	// Enabled controls whether the module is enabled.
	// nil = auto-detect/default behavior; true = always enabled; false = disabled.
	Enabled *bool `yaml:"enabled,omitempty"`
	// Order specifies the custom tab display order (lower values appear first).
	Order int `yaml:"order,omitempty"`
	// Title provides an optional custom label override for the tab.
	Title string `yaml:"title,omitempty"`
}

// Context carries runtime host and daemon state for dynamic visibility evaluation.
type Context struct {
	ShowSwarm     bool
	IsSwarmActive bool
}

// Module represents a pluggable functional unit (tab / monitor) in Dockpulse.
type Module interface {
	// ID returns the unique system identifier for this module.
	ID() ID
	// Name returns the human-readable display name.
	Name() string
	// DefaultTitle returns the default label displayed on the tab.
	DefaultTitle() string
	// DefaultOrder returns the natural position in the tab bar.
	DefaultOrder() int
	// Description returns a concise summary of what this module does.
	Description() string
	// IsEnabled evaluates whether the module should be visible and active.
	IsEnabled(ctx Context, cfg Config) bool
}

// BaseModule provides a standard implementation of the Module interface.
type BaseModule struct {
	id           ID
	name         string
	defaultTitle string
	defaultOrder int
	description  string
	enableCheck  func(ctx Context, cfg Config) bool
}

// NewBaseModule constructs a new BaseModule.
func NewBaseModule(id ID, name, defaultTitle string, defaultOrder int, description string, enableCheck func(ctx Context, cfg Config) bool) BaseModule {
	return BaseModule{
		id:           id,
		name:         name,
		defaultTitle: defaultTitle,
		defaultOrder: defaultOrder,
		description:  description,
		enableCheck:  enableCheck,
	}
}

// ID returns the module's unique ID.
func (b BaseModule) ID() ID { return b.id }

// Name returns the module's friendly name.
func (b BaseModule) Name() string { return b.name }

// DefaultTitle returns the default tab label.
func (b BaseModule) DefaultTitle() string { return b.defaultTitle }

// DefaultOrder returns the default tab position.
func (b BaseModule) DefaultOrder() int { return b.defaultOrder }

// Description returns a description of the module.
func (b BaseModule) Description() string { return b.description }

// IsEnabled evaluates module enablement, respecting explicit config overrides first.
func (b BaseModule) IsEnabled(ctx Context, cfg Config) bool {
	if cfg.Enabled != nil {
		return *cfg.Enabled
	}
	if b.enableCheck != nil {
		return b.enableCheck(ctx, cfg)
	}
	return true
}

// Built-in standard modules.
var (
	// ContainersModule handles Docker containers, statuses, and live lifecycle actions.
	ContainersModule = NewBaseModule(
		IDContainers,
		"Docker Containers",
		"Containers",
		1,
		"Manage and monitor Docker containers, inspect logs, and launch exec shells",
		func(_ Context, _ Config) bool { return true },
	)

	// ImagesModule handles Docker images, repository tags, and disk reclamation.
	ImagesModule = NewBaseModule(
		IDImages,
		"Docker Images",
		"Images",
		2,
		"Inspect Docker images, tags, sizes, and remove dangling images",
		func(_ Context, _ Config) bool { return true },
	)

	// ComposeModule manages Docker Compose project stacks and service groups.
	ComposeModule = NewBaseModule(
		IDCompose,
		"Docker Compose",
		"Compose",
		3,
		"Monitor Docker Compose stacks and aggregated service health",
		func(_ Context, _ Config) bool { return true },
	)

	// NetworksModule monitors Docker network drivers, subnets, and connected endpoints.
	NetworksModule = NewBaseModule(
		IDNetworks,
		"Docker Networks",
		"Networks",
		4,
		"Inspect bridge, overlay, and custom Docker networks",
		func(_ Context, _ Config) bool { return true },
	)

	// VolumesModule manages Docker volumes, mount points, and storage usage.
	VolumesModule = NewBaseModule(
		IDVolumes,
		"Docker Volumes",
		"Volumes",
		5,
		"Inspect and manage Docker storage volumes and reclaim disk space",
		func(_ Context, _ Config) bool { return true },
	)

	// SwarmModule displays Docker Swarm cluster nodes, managers, and service replicas.
	SwarmModule = NewBaseModule(
		IDSwarm,
		"Docker Swarm",
		"Swarm",
		6,
		"Monitor Swarm cluster topology, manager nodes, and service replicas",
		func(ctx Context, _ Config) bool {
			return ctx.ShowSwarm || ctx.IsSwarmActive
		},
	)
)

// ParseID resolves a string into a known module ID or returns an error.
func ParseID(val string) (ID, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "containers", "container", "docker", "1":
		return IDContainers, nil
	case "images", "image", "2":
		return IDImages, nil
	case "compose", "compose_projects", "stacks", "3":
		return IDCompose, nil
	case "networks", "network", "4":
		return IDNetworks, nil
	case "volumes", "volume", "5":
		return IDVolumes, nil
	case "swarm", "cluster", "6":
		return IDSwarm, nil
	default:
		return ID(val), fmt.Errorf("unknown module ID %q", val)
	}
}
