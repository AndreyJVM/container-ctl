package e2e

import (
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

func TestE2E_NetworkFilterAndBatchOperations(t *testing.T) {
	m := tui.NewModelWithOptions(
		2*time.Second,
		int(tui.TabNetworks),
		"v0.22.6",
	)

	initialData := actions.DataFetchedMsg{
		Networks: []monitor.DockerNetwork{
			{
				ID:             "net-fe-10",
				Name:           "net-frontend",
				Driver:         "bridge",
				Scope:          "local",
				Subnet:         "172.20.0.0/16",
				Gateway:        "172.20.0.1",
				Containers:     2,
				ComposeProject: "ecommerce",
			},
			{
				ID:             "net-be-20",
				Name:           "net-backend",
				Driver:         "bridge",
				Scope:          "local",
				Subnet:         "172.21.0.0/16",
				Gateway:        "172.21.0.1",
				Containers:     3,
				ComposeProject: "ecommerce",
			},
			{
				ID:             "net-ov-30",
				Name:           "net-overlay-swarm",
				Driver:         "overlay",
				Scope:          "swarm",
				Subnet:         "10.0.1.0/24",
				Gateway:        "10.0.1.1",
				Containers:     5,
				ComposeProject: "production-swarm",
			},
			{
				ID:         "net-mac-40",
				Name:       "net-macvlan",
				Driver:     "macvlan",
				Scope:      "local",
				Subnet:     "192.168.1.0/24",
				Gateway:    "192.168.1.1",
				Containers: 1,
			},
		},
		Timestamp: "15:00:00",
	}

	updated, _ := m.Update(initialData)
	m = updated.(tui.Model)
	assertTab(t, m, tui.TabNetworks)

	if len(m.FilteredNetworks()) != 4 {
		t.Fatalf("expected 4 networks, got %d", len(m.FilteredNetworks()))
	}

	// 1. Filter by Driver: "overlay"
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "overlay")
	if len(m.FilteredNetworks()) != 1 || m.FilteredNetworks()[0].Name != "net-overlay-swarm" {
		t.Fatalf("expected 1 network 'net-overlay-swarm' matching 'overlay', got %d", len(m.FilteredNetworks()))
	}

	// Clear filter
	m, _ = sendKey(m, "esc")
	if len(m.FilteredNetworks()) != 4 {
		t.Fatalf("expected 4 networks restored, got %d", len(m.FilteredNetworks()))
	}

	// 2. Filter by Subnet: "172.21."
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "172.21.")
	if len(m.FilteredNetworks()) != 1 || m.FilteredNetworks()[0].Name != "net-backend" {
		t.Fatalf("expected 1 network 'net-backend' matching subnet, got %d", len(m.FilteredNetworks()))
	}

	// Clear filter
	m, _ = sendKey(m, "esc")

	// 3. Filter by Compose Project: "ecommerce"
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "ecommerce")
	if len(m.FilteredNetworks()) != 2 {
		t.Fatalf("expected 2 networks for project 'ecommerce', got %d", len(m.FilteredNetworks()))
	}

	// Multi-select all filtered networks with 'a'
	// First press Enter to finalize filter input so 'a' acts as select all
	m, _ = sendKey(m, "enter")
	if m.IsFiltering() {
		t.Fatal("expected IsFiltering to be false after Enter")
	}
	m, _ = sendKey(m, "a")
	if m.SelectedNetworksCount() != 2 {
		t.Fatalf("expected 2 networks selected, got %d", m.SelectedNetworksCount())
	}

	// Clear filter query with Esc: all 4 networks visible, but 2 remain selected
	m, _ = sendKey(m, "esc")
	if m.FilterQuery() != "" {
		t.Fatalf("expected filter query cleared, got %q", m.FilterQuery())
	}
	if len(m.FilteredNetworks()) != 4 {
		t.Fatalf("expected 4 networks visible, got %d", len(m.FilteredNetworks()))
	}
	if m.SelectedNetworksCount() != 2 {
		t.Fatalf("expected selection of 2 networks to persist after clearing filter, got %d", m.SelectedNetworksCount())
	}

	// 4. Open Batch Remove Modal with 'd'
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalRemoveNetwork)
	assertContains(t, m.View(), "CONFIRM REMOVAL OF 2 NETWORKS")

	// Confirm batch deletion with 'y'
	m, removeCmd := sendKey(m, "y")
	assertModalClosed(t, m)
	if removeCmd == nil {
		t.Fatal("expected non-nil removeCmd upon batch remove confirmation")
	}
	if m.SelectedNetworksCount() != 0 {
		t.Fatalf("expected selected networks cleared after batch remove confirmation, got %d", m.SelectedNetworksCount())
	}
	assertContains(t, m.View(), "Removing 2 selected network(s)...")
}

func TestE2E_VolumeFilterAndBatchOperations(t *testing.T) {
	m := tui.NewModelWithOptions(
		2*time.Second,
		int(tui.TabVolumes),
		"v0.22.6",
	)

	initialData := actions.DataFetchedMsg{
		Volumes: []monitor.DockerVolume{
			{
				Name:           "pg_data_prod",
				Driver:         "local",
				Mountpoint:     "/var/lib/docker/volumes/pg_data_prod/_data",
				ComposeProject: "database-stack",
				RefCount:       1,
			},
			{
				Name:           "redis_cache_prod",
				Driver:         "local",
				Mountpoint:     "/var/lib/docker/volumes/redis_cache_prod/_data",
				ComposeProject: "cache-stack",
				RefCount:       0,
			},
			{
				Name:           "nfs_shared_storage",
				Driver:         "nfs",
				Mountpoint:     "/mnt/nfs/shared",
				ComposeProject: "storage-stack",
				RefCount:       0,
			},
		},
		Timestamp: "15:10:00",
	}

	updated, _ := m.Update(initialData)
	m = updated.(tui.Model)
	assertTab(t, m, tui.TabVolumes)

	if len(m.FilteredVolumes()) != 3 {
		t.Fatalf("expected 3 volumes, got %d", len(m.FilteredVolumes()))
	}

	// 1. Filter by Driver: "nfs"
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "nfs")
	if len(m.FilteredVolumes()) != 1 || m.FilteredVolumes()[0].Name != "nfs_shared_storage" {
		t.Fatalf("expected 1 volume 'nfs_shared_storage' matching driver 'nfs', got %d", len(m.FilteredVolumes()))
	}
	m, _ = sendKey(m, "esc")

	// 2. Filter by Mountpoint: "_data"
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "_data")
	if len(m.FilteredVolumes()) != 2 {
		t.Fatalf("expected 2 volumes matching mountpoint '_data', got %d", len(m.FilteredVolumes()))
	}
	m, _ = sendKey(m, "esc")

	// 3. Multi-Select across distinct filtered subsets
	// Subset A: search "cache" -> select item with space
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "cache")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "space")
	if m.SelectedVolumesCount() != 1 {
		t.Fatalf("expected 1 volume selected, got %d", m.SelectedVolumesCount())
	}

	// Clear filter: search "storage" -> select item with space
	m, _ = sendKey(m, "esc")
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "storage")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "space")
	if m.SelectedVolumesCount() != 2 {
		t.Fatalf("expected 2 volumes selected across filters, got %d", m.SelectedVolumesCount())
	}

	// Clear filter to see all 3 volumes
	m, _ = sendKey(m, "esc")
	if len(m.FilteredVolumes()) != 3 {
		t.Fatalf("expected 3 volumes restored, got %d", len(m.FilteredVolumes()))
	}
	if m.SelectedVolumesCount() != 2 {
		t.Fatalf("expected 2 selected volumes preserved, got %d", m.SelectedVolumesCount())
	}

	// 4. Batch Remove Modal: open with 'x', cancel with 'esc'
	m, _ = sendKey(m, "x")
	assertModal(t, m, tui.ModalRemoveVolume)
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
	if m.SelectedVolumesCount() != 2 {
		t.Fatalf("expected selections kept after cancelling modal, got %d", m.SelectedVolumesCount())
	}

	// Open again with 'd', confirm with 'y'
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalRemoveVolume)
	m, removeCmd := sendKey(m, "y")
	assertModalClosed(t, m)
	if removeCmd == nil {
		t.Fatal("expected non-nil removeCmd upon volume batch remove confirmation")
	}
	if m.SelectedVolumesCount() != 0 {
		t.Fatalf("expected volume selections cleared after confirmation, got %d", m.SelectedVolumesCount())
	}
	assertContains(t, m.View(), "Removing 2 selected volume(s)...")
}

func TestE2E_ConcurrentBackgroundRefreshDuringActiveFilter(t *testing.T) {
	m := createComprehensiveTestModel()
	assertTab(t, m, tui.TabDocker)

	// User initiates search with '/' and types partial prefix "web"
	m, _ = sendKey(m, "/")
	if !m.IsFiltering() {
		t.Fatal("expected IsFiltering to be true")
	}
	m = sendTypeString(m, "web")
	if m.FilterQuery() != "web" {
		t.Fatalf("expected FilterQuery 'web', got %q", m.FilterQuery())
	}

	initialFiltered := m.FilteredContainers()
	if len(initialFiltered) != 1 || initialFiltered[0].Name != "web-gateway" {
		t.Fatalf("expected 1 container 'web-gateway' filtered, got %d", len(initialFiltered))
	}

	// WHILE the user is actively typing in the filter input, a background refresh occurs!
	// New data contains an additional web container: "web-worker"
	backgroundData := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status:        "running",
			StatusDisplay: "Running",
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c101"), testutil.WithName("web-gateway"), testutil.WithState("running", "Up 5 hours")),
				testutil.AContainer(testutil.WithID("c104"), testutil.WithName("web-worker"), testutil.WithState("running", "Up 10m")),
				testutil.AContainer(testutil.WithID("c102"), testutil.WithName("api-backend"), testutil.WithState("running", "Up 3 hours")),
			},
		},
		Timestamp: "15:20:00",
	}

	// Update with background DataFetchedMsg
	updated, _ := m.Update(backgroundData)
	m = updated.(tui.Model)

	// CRITICAL UX INVARIANT CHECKS:
	// 1. IsFiltering must remain true - user's input mode is NOT interrupted!
	if !m.IsFiltering() {
		t.Fatal("expected IsFiltering to remain true during background refresh")
	}

	// 2. FilterQuery must be untouched - user's typed characters are NOT lost!
	if m.FilterQuery() != "web" {
		t.Fatalf("expected FilterQuery 'web' preserved, got %q", m.FilterQuery())
	}

	// 3. Filtered items must dynamically reflect the new data matching the query!
	newFiltered := m.FilteredContainers()
	if len(newFiltered) != 2 {
		t.Fatalf("expected 2 containers matching 'web' after background refresh, got %d: %+v", len(newFiltered), newFiltered)
	}

	// 4. Cursor must be valid and within range
	if m.ActiveCursor() >= len(newFiltered) {
		t.Fatalf("cursor %d out of bounds for %d items", m.ActiveCursor(), len(newFiltered))
	}

	// 5. User continues typing "-gateway" smoothly
	m = sendTypeString(m, "-gateway")
	if m.FilterQuery() != "web-gateway" {
		t.Fatalf("expected FilterQuery 'web-gateway', got %q", m.FilterQuery())
	}

	finalFiltered := m.FilteredContainers()
	if len(finalFiltered) != 1 || finalFiltered[0].Name != "web-gateway" {
		t.Fatalf("expected exactly 1 container 'web-gateway', got %d", len(finalFiltered))
	}

	// Press Enter to confirm filter
	m, _ = sendKey(m, "enter")
	if m.IsFiltering() {
		t.Fatal("expected IsFiltering to be false after Enter")
	}
	if m.FilterQuery() != "web-gateway" {
		t.Fatalf("expected query 'web-gateway' preserved after Enter, got %q", m.FilterQuery())
	}

	// Press Esc to clear filter
	m, _ = sendKey(m, "esc")
	if m.FilterQuery() != "" {
		t.Fatalf("expected filter query cleared after Esc, got %q", m.FilterQuery())
	}
	if len(m.FilteredContainers()) != 3 {
		t.Fatalf("expected all 3 containers restored, got %d", len(m.FilteredContainers()))
	}
}
