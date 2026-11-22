package filter

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestFilterContainers(t *testing.T) {
	containers := []monitor.ContainerItem{
		{ID: "c1", Name: "app-web", Image: "nginx:latest", CPUPerc: 10.0, MemUsageMB: 100.0, State: "running", Health: "healthy"},
		{ID: "c2", Name: "app-db", Image: "postgres:15", CPUPerc: 50.0, MemUsageMB: 500.0, State: "running", Health: "unhealthy"},
		{ID: "c3", Name: "redis-cache", Image: "redis:alpine", CPUPerc: 5.0, MemUsageMB: 50.0, State: "paused"},
	}

	// 1. Empty query returns all
	all := Containers(containers, "", SortDefault)
	if len(all) != 3 {
		t.Fatalf("expected 3 items, got %d", len(all))
	}

	// 2. Query filter
	filtered := Containers(containers, "postgres", SortDefault)
	if len(filtered) != 1 || filtered[0].Name != "app-db" {
		t.Errorf("expected app-db, got %+v", filtered)
	}

	// 2a. Query filter by State
	stateFiltered := Containers(containers, "paused", SortDefault)
	if len(stateFiltered) != 1 || stateFiltered[0].Name != "redis-cache" {
		t.Errorf("expected redis-cache when filtering by state 'paused', got %+v", stateFiltered)
	}

	// 2b. Query filter by Status, Ports, ComposeService
	extraContainers := []monitor.ContainerItem{
		{ID: "c4", Name: "custom-api", Status: "Up 5 hours", Ports: "0.0.0.0:8080->8080/tcp", ComposeService: "web-srv"},
	}
	if len(Containers(extraContainers, "8080", SortDefault)) != 1 {
		t.Errorf("expected match by port 8080")
	}
	if len(Containers(extraContainers, "5 hours", SortDefault)) != 1 {
		t.Errorf("expected match by status '5 hours'")
	}
	if len(Containers(extraContainers, "web-srv", SortDefault)) != 1 {
		t.Errorf("expected match by compose service 'web-srv'")
	}

	// 3. Sort by CPU
	cpuSorted := Containers(containers, "", SortCPU)
	if cpuSorted[0].Name != "app-db" {
		t.Errorf("expected highest CPU app-db first, got %s", cpuSorted[0].Name)
	}

	// 4. Sort by Mem
	memSorted := Containers(containers, "", SortMem)
	if memSorted[0].Name != "app-db" {
		t.Errorf("expected highest Mem app-db first, got %s", memSorted[0].Name)
	}

	// 5. Sort by Name
	nameSorted := Containers(containers, "", SortName)
	if nameSorted[0].Name != "app-db" || nameSorted[1].Name != "app-web" || nameSorted[2].Name != "redis-cache" {
		t.Errorf("unexpected name sorting order: %+v", nameSorted)
	}

	// 6. Sort by Status (unhealthy first)
	statusSorted := Containers(containers, "", SortStatus)
	if statusSorted[0].Name != "app-db" {
		t.Errorf("expected unhealthy app-db first in status sort, got %s", statusSorted[0].Name)
	}
}

func TestFilterImages(t *testing.T) {
	images := []monitor.DockerImage{
		{ID: "img1", Repository: "alpine", Tag: "3.18", InUse: true},
		{ID: "img2", Repository: "golang", Tag: "1.24", InUse: false},
		{ID: "img3", Repository: "<none>", Tag: "<none>", Dangling: true},
	}

	// Filter only unused
	unused := Images(images, "", true)
	if len(unused) != 2 {
		t.Errorf("expected 2 unused images, got %d", len(unused))
	}

	// Query matching
	matched := Images(images, "golang", false)
	if len(matched) != 1 || matched[0].Repository != "golang" {
		t.Errorf("expected golang, got %+v", matched)
	}

	// Query keyword 'unused'
	keywordUnused := Images(images, "unused", false)
	if len(keywordUnused) != 2 {
		t.Errorf("expected 2 unused images, got %d", len(keywordUnused))
	}

	// Query keyword 'dangling'
	keywordDangling := Images(images, "dangling", false)
	if len(keywordDangling) != 1 || !keywordDangling[0].Dangling {
		t.Errorf("expected 1 dangling image, got %+v", keywordDangling)
	}
}

func TestFilterCompose(t *testing.T) {
	projects := []monitor.ComposeProject{
		{
			Name: "proj-alpha",
			Services: []monitor.ComposeService{
				{Name: "web", Image: "nginx"},
				{Name: "api", Image: "golang"},
			},
		},
		{
			Name: "proj-beta",
			Services: []monitor.ComposeService{
				{Name: "worker", Image: "python"},
			},
		},
	}

	// Search by service name
	filtered := Compose(projects, "worker")
	if len(filtered) != 1 || filtered[0].Name != "proj-beta" {
		t.Errorf("expected proj-beta, got %+v", filtered)
	}

	// Search by project name
	filtered = Compose(projects, "alpha")
	if len(filtered) != 1 || filtered[0].Name != "proj-alpha" {
		t.Errorf("expected proj-alpha, got %+v", filtered)
	}
}

func TestFilterNetworksAndVolumes(t *testing.T) {
	networks := []monitor.DockerNetwork{
		{Name: "bridge", Driver: "bridge", Subnet: "172.17.0.0/16"},
		{Name: "custom-net", Driver: "overlay", Subnet: "10.0.0.0/24"},
	}
	vols := []monitor.DockerVolume{
		{Name: "data-vol", Driver: "local", Mountpoint: "/var/lib/docker/volumes/data-vol"},
		{Name: "cache-vol", Driver: "local", Mountpoint: "/var/lib/docker/volumes/cache-vol"},
	}

	// Filter network by subnet
	fn := Networks(networks, "10.0.0")
	if len(fn) != 1 || fn[0].Name != "custom-net" {
		t.Errorf("expected custom-net, got %+v", fn)
	}

	// Filter volume by mountpoint
	fv := Volumes(vols, "cache")
	if len(fv) != 1 || fv[0].Name != "cache-vol" {
		t.Errorf("expected cache-vol, got %+v", fv)
	}
}

func TestContainerStatusScore(t *testing.T) {
	tests := []struct {
		state  string
		status string
		health string
		want   int
	}{
		{state: "exited", status: "Exited (1) 2m ago", want: 1},
		{state: "running", health: "unhealthy", want: 1},
		{state: "running", health: "starting", want: 2},
		{state: "running", health: "healthy", want: 3},
		{state: "restarting", want: 4},
		{state: "paused", want: 5},
		{state: "created", want: 6},
	}

	for _, tt := range tests {
		c := monitor.ContainerItem{
			State:  tt.state,
			Status: tt.status,
			Health: tt.health,
		}
		if score := ContainerStatusScore(c); score != tt.want {
			t.Errorf("ContainerStatusScore(%+v) = %d, want %d", c, score, tt.want)
		}
	}
}
