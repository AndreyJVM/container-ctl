package testutil

import (
	"testing"
)

type mockTB struct {
	testing.TB
	failed bool
	msg    string
}

func (m *mockTB) Helper() {}

func (m *mockTB) Fatalf(format string, _ ...any) {
	m.failed = true
	m.msg = format
}

func TestAssertContainsAll(t *testing.T) {
	mock := &mockTB{}
	AssertContainsAll(mock, "hello beautiful world", "hello", "world")
	if mock.failed {
		t.Errorf("expected pass, but mock failed: %s", mock.msg)
	}

	mockFail := &mockTB{}
	AssertContainsAll(mockFail, "hello beautiful world", "hello", "missing")
	if !mockFail.failed {
		t.Errorf("expected failure when substring missing, but passed")
	}
}

func TestAssertNotContains(t *testing.T) {
	mock := &mockTB{}
	AssertNotContains(mock, "hello world", "foo", "bar")
	if mock.failed {
		t.Errorf("expected pass, but mock failed: %s", mock.msg)
	}

	mockFail := &mockTB{}
	AssertNotContains(mockFail, "hello world", "foo", "world")
	if !mockFail.failed {
		t.Errorf("expected failure when forbidden substring present, but passed")
	}
}

func TestAssertContainsInOrder(t *testing.T) {
	mock := &mockTB{}
	AssertContainsInOrder(mock, "first middle last", "first", "middle", "last")
	if mock.failed {
		t.Errorf("expected pass, but mock failed: %s", mock.msg)
	}

	mockFailOrder := &mockTB{}
	AssertContainsInOrder(mockFailOrder, "first middle last", "last", "first")
	if !mockFailOrder.failed {
		t.Errorf("expected failure when order is reversed, but passed")
	}

	mockFailMissing := &mockTB{}
	AssertContainsInOrder(mockFailMissing, "first middle last", "first", "missing")
	if !mockFailMissing.failed {
		t.Errorf("expected failure when item missing, but passed")
	}
}

func TestFixturesCoverage(t *testing.T) {
	c := AContainer(
		WithID("c-100"),
		WithName("test-cnt"),
		WithImage("alpine:latest"),
		WithState("running", "Up 10 hours"),
		WithHealth("healthy"),
		WithMetrics(25.5, 50.0, 100.0),
		WithNetworkIO(1024, 2048),
		WithBlockIO(4096, 8192),
		WithCompose("my-proj", "web"),
		WithPorts("0.0.0.0:8080->80/tcp"),
		WithMounts("/data:/data"),
		WithNetwork("172.17.0.2", "bridge"),
		WithRestartPolicy("always"),
		WithCreated("2026-11-20T10:00:00Z"),
	)
	if c.ID != "c-100" || c.Name != "test-cnt" || c.Image != "alpine:latest" {
		t.Errorf("unexpected container fixture: %+v", c)
	}

	img := AnImage(
		WithImageID("img-100", "sha256:img-100"),
		WithRepoTag("nginx", "1.25"),
		WithAllTags("nginx:1.25", "nginx:latest"),
		WithImageStatus(true, false, 2),
	)
	if img.ID != "img-100" || img.Repository != "nginx" || img.Tag != "1.25" {
		t.Errorf("unexpected image fixture: %+v", img)
	}

	net := ANetwork()
	if net.ID == "" || net.Name == "" {
		t.Errorf("unexpected network fixture: %+v", net)
	}

	vol := AVolume()
	if vol.Name == "" || vol.Driver == "" {
		t.Errorf("unexpected volume fixture: %+v", vol)
	}
}
