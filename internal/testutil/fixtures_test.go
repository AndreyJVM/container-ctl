package testutil

import (
	"testing"
)

func TestFixtures_AContainer(t *testing.T) {
	cDefault := AContainer()
	if cDefault.ID != "c123456789ab" || cDefault.Name != "test-container" {
		t.Errorf("unexpected default container: %+v", cDefault)
	}

	cCustom := AContainer(
		WithID("custom-id"),
		WithName("custom-name"),
		WithImage("alpine:3.18"),
		WithState("exited", "Exited (0) 5m ago"),
		WithHealth("unhealthy"),
		WithMetrics(25.0, 128.0, 512.0),
		WithNetworkIO(100, 200),
		WithBlockIO(300, 400),
		WithCompose("project-x", "service-y"),
		WithPorts("8080:80/tcp"),
		WithMounts("/host:/container:ro"),
		WithNetwork("10.0.0.2", "custom-net"),
		WithRestartPolicy("unless-stopped"),
		WithCreated("2026-11-20"),
	)

	if cCustom.ID != "custom-id" || cCustom.Name != "custom-name" || cCustom.Image != "alpine:3.18" {
		t.Errorf("unexpected custom identity: %+v", cCustom)
	}
	if cCustom.State != "exited" || cCustom.Status != "Exited (0) 5m ago" || cCustom.Health != "unhealthy" {
		t.Errorf("unexpected custom status: %+v", cCustom)
	}
	if cCustom.CPUPerc != 25.0 || cCustom.MemUsageMB != 128.0 || cCustom.MemPerc != 25.0 {
		t.Errorf("unexpected custom metrics: %+v", cCustom)
	}
	if cCustom.NetRxBytes != 100 || cCustom.NetTxBytes != 200 || cCustom.BlockReadBytes != 300 || cCustom.BlockWriteBytes != 400 {
		t.Errorf("unexpected custom IO: %+v", cCustom)
	}
	if cCustom.ComposeProject != "project-x" || cCustom.ComposeService != "service-y" {
		t.Errorf("unexpected compose fields: %+v", cCustom)
	}
	if cCustom.Ports != "8080:80/tcp" || len(cCustom.Mounts) != 1 || cCustom.Mounts[0] != "/host:/container:ro" {
		t.Errorf("unexpected ports/mounts: %+v", cCustom)
	}
	if cCustom.IPAddress != "10.0.0.2" || cCustom.NetworkName != "custom-net" || cCustom.RestartPolicy != "unless-stopped" || cCustom.Created != "2026-11-20" {
		t.Errorf("unexpected network/policy/created: %+v", cCustom)
	}

	// Zero limit branch in WithMetrics
	cZeroLimit := AContainer(WithMetrics(10.0, 50.0, 0))
	if cZeroLimit.MemPerc != 0 {
		t.Errorf("expected 0 MemPerc with 0 limit, got %f", cZeroLimit.MemPerc)
	}
}

func TestFixtures_AnImage(t *testing.T) {
	imgDefault := AnImage()
	if imgDefault.ID != "img123456789" || imgDefault.Repository != "test-repo" {
		t.Errorf("unexpected default image: %+v", imgDefault)
	}

	imgCustom := AnImage(
		WithImageID("img-custom", "sha256:img-custom-full"),
		WithRepoTag("my-repo", "v1.0"),
		WithAllTags("my-repo:v1.0", "my-repo:latest"),
		WithImageStatus(false, true, 0),
	)

	if imgCustom.ID != "img-custom" || imgCustom.FullID != "sha256:img-custom-full" {
		t.Errorf("unexpected custom image ID: %+v", imgCustom)
	}
	if imgCustom.Repository != "my-repo" || imgCustom.Tag != "v1.0" || len(imgCustom.AllTags) != 2 {
		t.Errorf("unexpected custom repo/tags: %+v", imgCustom)
	}
	if imgCustom.InUse || !imgCustom.Dangling || imgCustom.Containers != 0 {
		t.Errorf("unexpected custom status: %+v", imgCustom)
	}

	// Branch with <none> tag
	imgNone := AnImage(WithRepoTag("<none>", "<none>"))
	if imgNone.Repository != "<none>" || imgNone.Tag != "<none>" {
		t.Errorf("unexpected <none> repo tag: %+v", imgNone)
	}
}

func TestFixtures_ANetwork(t *testing.T) {
	net := ANetwork()
	if net.ID != "net123456789" || net.Name != "test-net" || !net.Attachable {
		t.Errorf("unexpected default network: %+v", net)
	}
}

func TestFixtures_AVolume(t *testing.T) {
	vol := AVolume()
	if vol.Name != "test-volume" || vol.Driver != "local" || vol.RefCount != 1 {
		t.Errorf("unexpected default volume: %+v", vol)
	}
}
