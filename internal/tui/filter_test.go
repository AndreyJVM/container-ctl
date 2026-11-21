package tui

import (
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterContainers(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web-gateway"), testutil.WithImage("nginx:alpine"), testutil.WithCompose("infra", "")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db-postgres"), testutil.WithImage("postgres:16"), testutil.WithCompose("backend", "")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("redis-cache"), testutil.WithImage("redis:7-alpine"), testutil.WithCompose("cache", "")),
	}

	// Empty query
	m.filterQuery = ""
	if len(m.getFilteredContainers()) != 3 {
		t.Errorf("expected all 3 containers for empty filter query")
	}

	// Filter by Name
	m.filterQuery = "web"
	filtered := m.getFilteredContainers()
	if len(filtered) != 1 || filtered[0].Name != "web-gateway" {
		t.Errorf("expected 1 result 'web-gateway', got %v", filtered)
	}

	// Case-insensitive match on Image
	m.filterQuery = "POSTGRES"
	filtered = m.getFilteredContainers()
	if len(filtered) != 1 || filtered[0].Name != "db-postgres" {
		t.Errorf("expected 1 result 'db-postgres', got %v", filtered)
	}

	// Match on ID
	m.filterQuery = "c3"
	filtered = m.getFilteredContainers()
	if len(filtered) != 1 || filtered[0].Name != "redis-cache" {
		t.Errorf("expected 1 result 'redis-cache', got %v", filtered)
	}

	// Match on ComposeProject
	m.filterQuery = "infra"
	filtered = m.getFilteredContainers()
	if len(filtered) != 1 || filtered[0].Name != "web-gateway" {
		t.Errorf("expected 1 result for compose project 'infra', got %v", filtered)
	}

	// No match
	m.filterQuery = "non-existent"
	if len(m.getFilteredContainers()) != 0 {
		t.Errorf("expected 0 results for non-matching query")
	}
}

func TestFilterModeKeys(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithImage("nginx")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("worker"), testutil.WithImage("app")),
	}

	// 1. Activate search mode with '/'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering=true after '/'")
	}

	// 2. Type "w", "e", "b"
	for _, r := range []rune("web") {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	if m.filterQuery != "web" {
		t.Errorf("expected filterQuery='web', got %q", m.filterQuery)
	}
	if len(m.getFilteredContainers()) != 1 {
		t.Errorf("expected 1 container matching 'web', got %d", len(m.getFilteredContainers()))
	}

	// 3. Backspace
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "we" {
		t.Errorf("expected filterQuery='we' after backspace, got %q", m.filterQuery)
	}

	// 4. Enter to apply filter and exit typing mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering=false after Enter")
	}
	if m.filterQuery != "we" {
		t.Errorf("expected filterQuery to persist after Enter")
	}

	// 5. Esc in normal mode clears filter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery='' after Esc, got %q", m.filterQuery)
	}
}

func TestFilterContainersByHealth(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithImage("nginx"), testutil.WithHealth("healthy")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db"), testutil.WithImage("postgres"), testutil.WithHealth("unhealthy")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("cache"), testutil.WithImage("redis"), testutil.WithHealth("starting")),
	}

	m.filterQuery = "unhealthy"
	filtered := m.getFilteredContainers()
	if len(filtered) != 1 || filtered[0].Name != "db" {
		t.Errorf("expected 1 result matching health 'unhealthy', got %v", filtered)
	}

	m.filterQuery = "healthy"
	filtered = m.getFilteredContainers()
	// "healthy" is substring of "unhealthy" as well, so matches c1 and c2
	if len(filtered) != 2 {
		t.Errorf("expected 2 results matching 'healthy', got %d", len(filtered))
	}
}

func TestFilterModelDelegation(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")

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

	m.filterQuery = "prod"
	if len(m.getFilteredCompose()) != 1 {
		t.Errorf("expected 1 compose project matching 'prod'")
	}

	m.filterQuery = "overlay"
	if len(m.getFilteredNetworks()) != 1 {
		t.Errorf("expected 1 network matching 'overlay'")
	}

	m.filterQuery = "cache"
	if len(m.getFilteredVolumes()) != 1 {
		t.Errorf("expected 1 volume matching 'cache'")
	}

	m.SetOnlyUnusedImages(true)
	if !m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be true")
	}
	m.SetOnlyUnusedImages(false)
	if m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be false")
	}

	m.SetContainerSortMode(SortName)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode SortName, got %d", m.containerSortMode)
	}
	// Invalid mode should be ignored
	m.SetContainerSortMode(999)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode to remain SortName, got %d", m.containerSortMode)
	}

	score := containerStatusScore(monitor.ContainerItem{State: "running", Health: "healthy"})
	if score == 0 {
		t.Errorf("expected non-zero container status score")
	}
}
