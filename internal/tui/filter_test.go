package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterContainers(t *testing.T) {
	t.Parallel()

	containers := []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web-gateway"), testutil.WithImage("nginx:alpine"), testutil.WithCompose("infra", "")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db-postgres"), testutil.WithImage("postgres:16"), testutil.WithCompose("backend", "")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("redis-cache"), testutil.WithImage("redis:7-alpine"), testutil.WithCompose("cache", "")),
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "empty query returns all containers",
			query:         "",
			expectedCount: 3,
			expectedNames: []string{"web-gateway", "db-postgres", "redis-cache"},
		},
		{
			name:          "whitespace only query returns all containers",
			query:         "   ",
			expectedCount: 3,
			expectedNames: []string{"web-gateway", "db-postgres", "redis-cache"},
		},
		{
			name:          "filter by container name",
			query:         "web",
			expectedCount: 1,
			expectedNames: []string{"web-gateway"},
		},
		{
			name:          "case-insensitive match on image",
			query:         "POSTGRES",
			expectedCount: 1,
			expectedNames: []string{"db-postgres"},
		},
		{
			name:          "match by container ID",
			query:         "c3",
			expectedCount: 1,
			expectedNames: []string{"redis-cache"},
		},
		{
			name:          "match by compose project name",
			query:         "infra",
			expectedCount: 1,
			expectedNames: []string{"web-gateway"},
		},
		{
			name:          "no match returns empty slice",
			query:         "non-existent-service",
			expectedCount: 0,
			expectedNames: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newFastTestModel()
			m.dockerInfo.Containers = containers
			m.filterQuery = tc.query

			filtered := m.getFilteredContainers()
			if len(filtered) != tc.expectedCount {
				t.Fatalf("expected %d containers, got %d", tc.expectedCount, len(filtered))
			}
			for i, name := range tc.expectedNames {
				if filtered[i].Name != name {
					t.Errorf("expected container %d to be %q, got %q", i, name, filtered[i].Name)
				}
			}
		})
	}
}

func TestFilterContainersByHealth(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithImage("nginx"), testutil.WithHealth("healthy")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db"), testutil.WithImage("postgres"), testutil.WithHealth("unhealthy")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("cache"), testutil.WithImage("redis"), testutil.WithHealth("starting")),
	}

	t.Run("filter by unhealthy", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.filterQuery = "unhealthy"
		filtered := subM.getFilteredContainers()
		if len(filtered) != 1 || filtered[0].Name != "db" {
			t.Errorf("expected 1 result matching 'unhealthy' (db), got %v", filtered)
		}
	})

	t.Run("filter by healthy matches substring in unhealthy too", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.filterQuery = "healthy"
		filtered := subM.getFilteredContainers()
		if len(filtered) != 2 {
			t.Errorf("expected 2 results matching 'healthy', got %d", len(filtered))
		}
	})

	t.Run("filter by starting", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.filterQuery = "starting"
		filtered := subM.getFilteredContainers()
		if len(filtered) != 1 || filtered[0].Name != "cache" {
			t.Errorf("expected 1 result matching 'starting', got %v", filtered)
		}
	})
}

func TestFilterModeKeys(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web server"), testutil.WithImage("nginx")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("worker"), testutil.WithImage("app")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("database"), testutil.WithImage("postgres")),
	}
	m.cursorIdx = 2 // Point to last container

	// 1. Activate search mode with '/'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering=true after '/'")
	}

	// 2. Type "web" with Space and "server"
	for _, r := range []rune("web server") {
		if r == ' ' {
			res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
		} else {
			res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		m = res.(Model)
	}
	if m.filterQuery != "web server" {
		t.Errorf("expected filterQuery='web server', got %q", m.filterQuery)
	}
	if len(m.getFilteredContainers()) != 1 {
		t.Errorf("expected 1 container matching 'web server', got %d", len(m.getFilteredContainers()))
	}
	// Cursor must clamp to 0 since only 1 container matches
	if m.cursorIdx != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursorIdx)
	}

	// 3. Backspace deletes character
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "web serve" {
		t.Errorf("expected filterQuery='web serve' after backspace, got %q", m.filterQuery)
	}

	// 4. Enter applies filter and exits typing mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering=false after Enter")
	}
	if m.filterQuery != "web serve" {
		t.Errorf("expected filterQuery to persist after Enter, got %q", m.filterQuery)
	}

	// 5. Esc in normal mode clears filter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery='' after Esc, got %q", m.filterQuery)
	}

	// 6. Backspace on empty filterQuery does not panic
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery to remain empty after backspace on empty string")
	}

	// 7. Esc during active typing clears query and exits filtering
	m.isFiltering = true
	m.filterQuery = "search"
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.isFiltering || m.filterQuery != "" {
		t.Errorf("expected isFiltering=false and filterQuery='' after Esc during typing")
	}
}

func TestFilterModelDelegation(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()

	m.images = []monitor.DockerImage{
		{ID: "img1", Repository: "nginx", Tag: "latest", InUse: true},
		{ID: "img2", Repository: "redis", Tag: "alpine", InUse: false},
		{ID: "img3", Repository: "<none>", Tag: "<none>", InUse: false, Dangling: true},
	}
	m.compose = []monitor.ComposeProject{
		{Name: "app-prod"},
		{Name: "app-dev"},
	}
	m.networks = []monitor.DockerNetwork{
		{ID: "n1", Name: "bridge-net", Driver: "bridge"},
		{ID: "n2", Name: "custom-net", Driver: "overlay"},
	}
	m.volumes = []monitor.DockerVolume{
		{Name: "db-data", Driver: "local"},
		{Name: "cache-data", Driver: "local"},
	}

	// Compose delegation
	m.filterQuery = "prod"
	if len(m.getFilteredCompose()) != 1 || m.getFilteredCompose()[0].Name != "app-prod" {
		t.Errorf("expected 1 compose project matching 'prod'")
	}

	// Networks delegation
	m.filterQuery = "overlay"
	if len(m.getFilteredNetworks()) != 1 || m.getFilteredNetworks()[0].Name != "custom-net" {
		t.Errorf("expected 1 network matching 'overlay'")
	}

	// Volumes delegation
	m.filterQuery = "cache"
	if len(m.getFilteredVolumes()) != 1 || m.getFilteredVolumes()[0].Name != "cache-data" {
		t.Errorf("expected 1 volume matching 'cache'")
	}

	// Images delegation with query and onlyUnused
	m.filterQuery = "redis"
	if len(m.getFilteredImages()) != 1 || m.getFilteredImages()[0].Repository != "redis" {
		t.Errorf("expected 1 image matching 'redis'")
	}

	m.filterQuery = ""
	m.SetOnlyUnusedImages(true)
	if !m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be true")
	}
	if len(m.getFilteredImages()) != 2 {
		t.Errorf("expected 2 unused images (redis and dangling), got %d", len(m.getFilteredImages()))
	}

	m.SetOnlyUnusedImages(false)
	if m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be false")
	}
	if len(m.getFilteredImages()) != 3 {
		t.Errorf("expected all 3 images when onlyUnused is false")
	}

	// Sort mode setters and boundary checks
	m.SetContainerSortMode(SortName)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode SortName, got %d", m.containerSortMode)
	}
	// Invalid modes out of range should be ignored
	m.SetContainerSortMode(999)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode to remain SortName when 999 passed, got %d", m.containerSortMode)
	}
	m.SetContainerSortMode(-1)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode to remain SortName when -1 passed, got %d", m.containerSortMode)
	}

	// Container status score evaluation
	runningHealthy := containerStatusScore(monitor.ContainerItem{State: "running", Health: "healthy"})
	runningUnhealthy := containerStatusScore(monitor.ContainerItem{State: "running", Health: "unhealthy"})
	if runningUnhealthy >= runningHealthy {
		t.Errorf("expected unhealthy containers to have higher priority (lower score) than healthy")
	}
}

func TestContainerSortingViaModel(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithName("beta"), testutil.WithMetrics(10.0, 500.0, 1024.0)),
		testutil.AContainer(testutil.WithName("alpha"), testutil.WithMetrics(85.0, 100.0, 1024.0)),
		testutil.AContainer(testutil.WithName("gamma"), testutil.WithMetrics(45.0, 900.0, 1024.0)),
	}

	t.Run("sort by CPU descending", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.SetContainerSortMode(SortCPU)
		sorted := subM.getFilteredContainers()
		if sorted[0].Name != "alpha" || sorted[1].Name != "gamma" || sorted[2].Name != "beta" {
			t.Errorf("unexpected CPU sort order: %v, %v, %v", sorted[0].Name, sorted[1].Name, sorted[2].Name)
		}
	})

	t.Run("sort by memory descending", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.SetContainerSortMode(SortMem)
		sorted := subM.getFilteredContainers()
		if sorted[0].Name != "gamma" || sorted[1].Name != "beta" || sorted[2].Name != "alpha" {
			t.Errorf("unexpected memory sort order: %v, %v, %v", sorted[0].Name, sorted[1].Name, sorted[2].Name)
		}
	})

	t.Run("sort by name ascending", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.SetContainerSortMode(SortName)
		sorted := subM.getFilteredContainers()
		if sorted[0].Name != "alpha" || sorted[1].Name != "beta" || sorted[2].Name != "gamma" {
			t.Errorf("unexpected name sort order: %v, %v, %v", sorted[0].Name, sorted[1].Name, sorted[2].Name)
		}
	})
}

func TestComposeNetworksVolumesSearchFilter(t *testing.T) {
	t.Parallel()

	m := Model{
		version:         "v0.18.1",
		activeTab:       TabCompose,
		refreshInterval: 2 * time.Second,
		compose: []monitor.ComposeProject{
			{Name: "web-stack", Status: "running", Services: []monitor.ComposeService{{Name: "web-srv", Image: "nginx"}}},
			{Name: "db-stack", Status: "running", Services: []monitor.ComposeService{{Name: "db-srv", Image: "postgres"}}},
		},
		networks: []monitor.DockerNetwork{
			{ID: "n1", Name: "bridge-net", Driver: "bridge", Subnet: "172.18.0.0/16"},
			{ID: "n2", Name: "custom-net", Driver: "overlay", Subnet: "10.0.0.0/24"},
		},
		volumes: []monitor.DockerVolume{
			{Name: "db-data", Driver: "local"},
			{Name: "cache-data", Driver: "local"},
		},
	}

	// 1. Open search on TabCompose with "/"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering true on Compose tab")
	}

	// Type "db"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = res.(Model)

	filteredCompose := m.getFilteredCompose()
	if len(filteredCompose) != 1 || filteredCompose[0].Name != "db-stack" {
		t.Errorf("expected db-stack matching 'db', got %+v", filteredCompose)
	}

	// Apply filter with Enter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering false after Enter")
	}

	// View rendering contains filter badge
	v := m.View()
	if !strings.Contains(v, "Filter: db") {
		t.Errorf("expected filter badge in compose view, got: %s", v)
	}

	// Reset with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery empty after Esc, got %q", m.filterQuery)
	}
	if len(m.getFilteredCompose()) != 2 {
		t.Errorf("expected all compose projects restored")
	}

	// 2. Search on TabNetworks
	m.activeTab = TabNetworks
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering true on Networks tab")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o', 'v', 'e', 'r', 'l', 'a', 'y'}})
	m = res.(Model)
	filteredNets := m.getFilteredNetworks()
	if len(filteredNets) != 1 || filteredNets[0].Name != "custom-net" {
		t.Errorf("expected custom-net matching overlay driver, got %+v", filteredNets)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery reset after Esc")
	}

	// 3. Search on TabVolumes
	m.activeTab = TabVolumes
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering true on Volumes tab")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c', 'a', 'c', 'h', 'e'}})
	m = res.(Model)
	filteredVols := m.getFilteredVolumes()
	if len(filteredVols) != 1 || filteredVols[0].Name != "cache-data" {
		t.Errorf("expected cache-data matching 'cache', got %+v", filteredVols)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery reset after Esc")
	}
}
