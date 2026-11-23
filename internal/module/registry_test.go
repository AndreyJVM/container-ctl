package module

import (
	"testing"
)

func TestDefaultRegistry_StandardVisibility(t *testing.T) {
	r := DefaultRegistry()

	if r.TotalCount() != 6 {
		t.Fatalf("expected 6 registered standard modules, got %d", r.TotalCount())
	}

	// 1. Swarm inactive -> 5 visible modules
	ctxInactive := Context{ShowSwarm: false, IsSwarmActive: false}
	if r.VisibleCount(ctxInactive) != 5 {
		t.Errorf("expected 5 visible modules without swarm, got %d", r.VisibleCount(ctxInactive))
	}

	visible := r.VisibleModules(ctxInactive)
	expectedIDs := []ID{IDContainers, IDImages, IDCompose, IDNetworks, IDVolumes}
	expectedLabels := []string{"1: Containers", "2: Images", "3: Compose", "4: Networks", "5: Volumes"}

	for i, v := range visible {
		if v.ID != expectedIDs[i] {
			t.Errorf("pos %d: expected ID %s, got %s", i, expectedIDs[i], v.ID)
		}
		if v.Index != i+1 {
			t.Errorf("pos %d: expected Index %d, got %d", i, i+1, v.Index)
		}
		if v.TabLabel != expectedLabels[i] {
			t.Errorf("pos %d: expected TabLabel %q, got %q", i, expectedLabels[i], v.TabLabel)
		}
	}

	// 2. Swarm active -> 6 visible modules
	ctxActive := Context{ShowSwarm: false, IsSwarmActive: true}
	if r.VisibleCount(ctxActive) != 6 {
		t.Errorf("expected 6 visible modules with swarm active, got %d", r.VisibleCount(ctxActive))
	}
	vSwarm, found := r.FindVisibleByID(IDSwarm, ctxActive)
	if !found {
		t.Fatalf("expected Swarm module visible when active")
	}
	if vSwarm.Index != 6 || vSwarm.TabLabel != "6: Swarm" {
		t.Errorf("unexpected Swarm VisibleModule: %+v", vSwarm)
	}

	// 3. FindVisibleByIndex
	v1, found1 := r.FindVisibleByIndex(1, ctxInactive)
	if !found1 || v1.ID != IDContainers {
		t.Errorf("expected Index 1 to be Containers, got: %+v (found=%v)", v1, found1)
	}

	_, foundOutOfBounds := r.FindVisibleByIndex(99, ctxInactive)
	if foundOutOfBounds {
		t.Errorf("expected out-of-bounds index to return false")
	}
}

func TestRegistry_CustomOrderingAndDisable(t *testing.T) {
	r := DefaultRegistry()

	disabled := false
	r.SetModuleConfig(IDImages, Config{Enabled: &disabled})

	// Change order of Compose to 1 and custom title
	r.SetModuleConfig(IDCompose, Config{Order: 1, Title: "Stacks"})
	// Change Containers to order 2
	r.SetModuleConfig(IDContainers, Config{Order: 2, Title: "DockerApps"})

	ctx := Context{ShowSwarm: false, IsSwarmActive: false}
	visible := r.VisibleModules(ctx)

	// Expected:
	// 1. Compose ("1: Stacks")
	// 2. Containers ("2: DockerApps")
	// 3. Networks ("3: Networks")
	// 4. Volumes ("4: Volumes")
	// Images should be omitted because enabled=false.
	if len(visible) != 4 {
		t.Fatalf("expected 4 visible modules, got %d", len(visible))
	}

	if visible[0].ID != IDCompose || visible[0].TabLabel != "1: Stacks" {
		t.Errorf("expected 1st visible module to be Compose / Stacks, got %+v", visible[0])
	}
	if visible[1].ID != IDContainers || visible[1].TabLabel != "2: DockerApps" {
		t.Errorf("expected 2nd visible module to be Containers / DockerApps, got %+v", visible[1])
	}
	if visible[2].ID != IDNetworks || visible[2].TabLabel != "3: Networks" {
		t.Errorf("expected 3rd visible module to be Networks, got %+v", visible[2])
	}
	if visible[3].ID != IDVolumes || visible[3].TabLabel != "4: Volumes" {
		t.Errorf("expected 4th visible module to be Volumes, got %+v", visible[3])
	}

	if r.IsModuleVisible(IDImages, ctx) {
		t.Errorf("expected Images to be not visible")
	}
}

func TestRegistry_CustomPluggableModule(t *testing.T) {
	r := DefaultRegistry()

	// Register a new custom module: Host Metrics
	customHostModule := NewBaseModule(
		ID("host_metrics"),
		"Host System Metrics",
		"Host",
		7,
		"Monitor CPU and RAM of the host system",
		func(_ Context, _ Config) bool { return true },
	)

	r.Register(customHostModule)

	if r.TotalCount() != 7 {
		t.Fatalf("expected 7 registered modules, got %d", r.TotalCount())
	}

	ctx := Context{ShowSwarm: false, IsSwarmActive: false}
	visible := r.VisibleModules(ctx)
	// 5 standard + 1 custom = 6
	if len(visible) != 6 {
		t.Fatalf("expected 6 visible modules, got %d", len(visible))
	}

	last := visible[len(visible)-1]
	if last.ID != ID("host_metrics") || last.TabLabel != "6: Host" {
		t.Errorf("expected last module to be custom host_metrics, got: %+v", last)
	}

	mod, ok := r.GetModule(ID("host_metrics"))
	if !ok || mod.Name() != "Host System Metrics" {
		t.Errorf("unexpected GetModule result: %+v", mod)
	}
}

func TestRegistry_ApplyConfigs(t *testing.T) {
	r := DefaultRegistry()

	enabled := true
	disabled := false
	r.ApplyConfigs(map[string]Config{
		"swarm":  {Enabled: &enabled},
		"images": {Enabled: &disabled},
	})

	ctx := Context{ShowSwarm: false, IsSwarmActive: false}
	if !r.IsModuleVisible(IDSwarm, ctx) {
		t.Errorf("expected Swarm to be visible via ApplyConfigs")
	}
	if r.IsModuleVisible(IDImages, ctx) {
		t.Errorf("expected Images to be hidden via ApplyConfigs")
	}

	cfg := r.GetModuleConfig(IDSwarm)
	if cfg.Enabled == nil || !*cfg.Enabled {
		t.Errorf("expected GetModuleConfig to reflect enabled swarm")
	}

	// Nil / empty map should do nothing
	r.ApplyConfigs(nil)
	r.ApplyConfigs(map[string]Config{})

	// Unknown / custom module ID string
	r.ApplyConfigs(map[string]Config{
		"custom_ext": {Order: 99},
	})
	if customCfg := r.GetModuleConfig(ID("custom_ext")); customCfg.Order != 99 {
		t.Errorf("expected custom_ext config order 99, got %d", customCfg.Order)
	}

	// Register nil should safely do nothing
	countBefore := r.TotalCount()
	r.Register(nil)
	if r.TotalCount() != countBefore {
		t.Errorf("expected TotalCount unchanged after Register(nil)")
	}
}
