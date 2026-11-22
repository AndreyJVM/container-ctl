package module

import (
	"testing"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		input       string
		expectedID  ID
		shouldError bool
	}{
		{"containers", IDContainers, false},
		{"docker", IDContainers, false},
		{"1", IDContainers, false},
		{"images", IDImages, false},
		{"image", IDImages, false},
		{"2", IDImages, false},
		{"compose", IDCompose, false},
		{"stacks", IDCompose, false},
		{"3", IDCompose, false},
		{"networks", IDNetworks, false},
		{"4", IDNetworks, false},
		{"volumes", IDVolumes, false},
		{"volume", IDVolumes, false},
		{"5", IDVolumes, false},
		{"swarm", IDSwarm, false},
		{"cluster", IDSwarm, false},
		{"6", IDSwarm, false},
		{"unknown_module", ID("unknown_module"), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, err := ParseID(tt.input)
			if tt.shouldError {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for %q: %v", tt.input, err)
				}
				if id != tt.expectedID {
					t.Errorf("expected %q, got %q", tt.expectedID, id)
				}
			}
		})
	}
}

func TestStandardModulesDefaults(t *testing.T) {
	ctx := Context{ShowSwarm: false, IsSwarmActive: false}

	if !ContainersModule.IsEnabled(ctx, Config{}) {
		t.Errorf("ContainersModule should be enabled by default")
	}
	if !ImagesModule.IsEnabled(ctx, Config{}) {
		t.Errorf("ImagesModule should be enabled by default")
	}
	if !ComposeModule.IsEnabled(ctx, Config{}) {
		t.Errorf("ComposeModule should be enabled by default")
	}
	if !NetworksModule.IsEnabled(ctx, Config{}) {
		t.Errorf("NetworksModule should be enabled by default")
	}
	if !VolumesModule.IsEnabled(ctx, Config{}) {
		t.Errorf("VolumesModule should be enabled by default")
	}

	// Swarm inactive and not forced: disabled
	if SwarmModule.IsEnabled(ctx, Config{}) {
		t.Errorf("SwarmModule should be disabled when swarm is inactive and ShowSwarm is false")
	}

	// Swarm with ShowSwarm: enabled
	ctxShowSwarm := Context{ShowSwarm: true, IsSwarmActive: false}
	if !SwarmModule.IsEnabled(ctxShowSwarm, Config{}) {
		t.Errorf("SwarmModule should be enabled when ShowSwarm is true")
	}

	// Swarm active: enabled
	ctxActiveSwarm := Context{ShowSwarm: false, IsSwarmActive: true}
	if !SwarmModule.IsEnabled(ctxActiveSwarm, Config{}) {
		t.Errorf("SwarmModule should be enabled when IsSwarmActive is true")
	}

	// Explicit override in Config overrides context
	disabled := false
	if SwarmModule.IsEnabled(ctxActiveSwarm, Config{Enabled: &disabled}) {
		t.Errorf("SwarmModule should be disabled when explicitly overridden with enabled=false")
	}

	enabled := true
	if !SwarmModule.IsEnabled(ctx, Config{Enabled: &enabled}) {
		t.Errorf("SwarmModule should be enabled when explicitly overridden with enabled=true")
	}
}

func TestBaseModuleGetters(t *testing.T) {
	bm := NewBaseModule(ID("custom"), "Custom Name", "Custom", 10, "Custom Description", nil)
	if bm.ID() != ID("custom") {
		t.Errorf("unexpected ID: %s", bm.ID())
	}
	if bm.Name() != "Custom Name" {
		t.Errorf("unexpected Name: %s", bm.Name())
	}
	if bm.DefaultTitle() != "Custom" {
		t.Errorf("unexpected DefaultTitle: %s", bm.DefaultTitle())
	}
	if bm.DefaultOrder() != 10 {
		t.Errorf("unexpected DefaultOrder: %d", bm.DefaultOrder())
	}
	if bm.Description() != "Custom Description" {
		t.Errorf("unexpected Description: %s", bm.Description())
	}
	if !bm.IsEnabled(Context{}, Config{}) {
		t.Errorf("BaseModule with nil enableCheck should default to true")
	}
}
