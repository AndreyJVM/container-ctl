package modals

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestRenderInspect(t *testing.T) {
	detail := monitor.ContainerDetail{
		ID:              "c123456789",
		Name:            "web-service",
		Image:           "nginx:alpine",
		State:           "running",
		Status:          "Up 3 hours",
		Health:          "healthy",
		CPUPerc:         4.5,
		MemUsageMB:      120.0,
		MemPerc:         3.2,
		Created:         "2026-11-06",
		IP:              "172.17.0.2",
		Gateway:         "172.17.0.1",
		Command:         "nginx -g daemon off;",
		Mounts:          []string{"/host/data -> /data (ro)"},
		NetRxBytes:      5 * 1024 * 1024,
		NetTxBytes:      10 * 1024 * 1024,
		BlockReadBytes:  2 * 1024 * 1024,
		BlockWriteBytes: 4 * 1024 * 1024,
		RestartPolicy:   "unless-stopped",
	}

	view := RenderInspect(detail)
	if !strings.Contains(view, "web-service") || !strings.Contains(view, "[healthy]") {
		t.Errorf("expected container name and healthy badge, got: %s", view)
	}
	if !strings.Contains(view, "172.17.0.2") || !strings.Contains(view, "/host/data") {
		t.Errorf("expected IP address and mounts in inspect view, got: %s", view)
	}
	if !strings.Contains(view, "Net I/O:") || !strings.Contains(view, "5.0 MB / 10.0 MB") {
		t.Errorf("expected Net I/O in inspect view, got: %s", view)
	}
	if !strings.Contains(view, "Block I/O:") || !strings.Contains(view, "2.0 MB / 4.0 MB") {
		t.Errorf("expected Block I/O in inspect view, got: %s", view)
	}
	if !strings.Contains(view, "unless-stopped") || !strings.Contains(view, "Restart Policy:") {
		t.Errorf("expected Restart Policy in inspect view, got: %s", view)
	}

	// Environment rendering with masked secrets
	detailWithEnv := detail
	detailWithEnv.Env = []string{
		"APP_ENV=production",
		"DB_PASSWORD=supersecret",
		"API_KEY=abcdef123456",
		"PORT=8080",
	}
	envView := RenderInspect(detailWithEnv)
	if !strings.Contains(envView, "APP_ENV=production") || !strings.Contains(envView, "PORT=8080") {
		t.Errorf("expected regular env vars rendered, got: %s", envView)
	}
	if !strings.Contains(envView, "DB_PASSWORD=******") || strings.Contains(envView, "supersecret") {
		t.Errorf("expected DB_PASSWORD to be masked, got: %s", envView)
	}
	if !strings.Contains(envView, "API_KEY=******") || strings.Contains(envView, "abcdef123456") {
		t.Errorf("expected API_KEY to be masked, got: %s", envView)
	}
}

func TestMaskEnvVar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PORT=8080", "PORT=8080"},
		{"DB_PASS=mysecret", "DB_PASS=******"},
		{"AUTH_TOKEN=jwt1234", "AUTH_TOKEN=******"},
		{"AWS_SECRET_ACCESS_KEY=xyz", "AWS_SECRET_ACCESS_KEY=******"},
		{"CREDENTIALS_FILE=/tmp/cred", "CREDENTIALS_FILE=******"},
		{"NO_EQUALS", "NO_EQUALS"},
	}

	for _, tt := range tests {
		actual := MaskEnvVar(tt.input)
		if actual != tt.expected {
			t.Errorf("MaskEnvVar(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestRenderInspect_Scrolling(t *testing.T) {
	detail := monitor.ContainerDetail{
		ID:     "c1234567890abcdef",
		Name:   "huge-app",
		Image:  "my-app:v1",
		State:  "running",
		Status: "Up 10 hours",
		Health: "unhealthy",
		Mounts: []string{"/m1", "/m2", "/m3"},
		Env: []string{
			"VAR1=val1", "VAR2=val2", "VAR3=val3", "VAR4=val4",
			"VAR5=val5", "VAR6=val6", "VAR7=val7", "VAR8=val8",
			"VAR9=val9", "VAR10=val10", "VAR11=val11", "VAR12=val12",
		},
	}

	lines := BuildInspectLines(detail)
	if len(lines) <= DefaultMaxVisibleInspect {
		t.Fatalf("expected more than %d lines, got %d", DefaultMaxVisibleInspect, len(lines))
	}

	// Render page 1 (scroll = 0)
	v0 := RenderInspect(detail, 0)
	if !strings.Contains(v0, "[Lines 1-18 of") || !strings.Contains(v0, "[y] Copy ID") {
		t.Errorf("expected pagination footer in v0, got: %s", v0)
	}

	// Render scrolled
	vScroll := RenderInspect(detail, 5)
	if !strings.Contains(vScroll, "[Lines 6-") {
		t.Errorf("expected scrolled lines indicator, got: %s", vScroll)
	}
}

func TestBuildInspectLines_Variants(t *testing.T) {
	// Test starting status
	startingDetail := monitor.ContainerDetail{
		ID:     "starting-c",
		Name:   "starter",
		Health: "starting",
	}
	lines := BuildInspectLines(startingDetail)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "[starting]") {
		t.Errorf("expected [starting] health badge, got: %s", joined)
	}
}

func TestSecurity_MaskSensitiveEnvVars(t *testing.T) {
	vulnerableInputs := []struct {
		name     string
		env      string
		leaksVal string
	}{
		{
			name:     "Database URL with embedded password",
			env:      "DATABASE_URL=postgres://app_user:super_secret_pw@db.internal:5432/prod_db",
			leaksVal: "super_secret_pw",
		},
		{
			name:     "Redis URL with password",
			env:      "REDIS_URL=redis://:redis_secret_123@redis:6379/0",
			leaksVal: "redis_secret_123",
		},
		{
			name:     "MongoDB URI with credentials",
			env:      "MONGO_URI=mongodb://admin:admin_secret_99@mongo:27017/admin",
			leaksVal: "admin_secret_99",
		},
		{
			name:     "RabbitMQ AMQP URI with credentials",
			env:      "AMQP_URL=amqp://guest:guest_password@rabbitmq:5672/",
			leaksVal: "guest_password",
		},
		{
			name:     "DSN string",
			env:      "DSN=host=localhost user=root password=db_pass dbname=test",
			leaksVal: "db_pass",
		},
		{
			name:     "Sentry DSN",
			env:      "SENTRY_DSN=https://pub_key:priv_token_secret@sentry.io/12345",
			leaksVal: "priv_token_secret",
		},
		{
			name:     "Signing Salt",
			env:      "COOKIE_SALT=very_secret_signing_salt_hex",
			leaksVal: "very_secret_signing_salt_hex",
		},
		{
			name:     "Webhook URL",
			env:      "SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T00/B00/secret_token_12345",
			leaksVal: "secret_token_12345",
		},
		{
			name:     "JWT Secret / Signing key",
			env:      "JWT_SIGNATURE=hmac_secret_sha256_key",
			leaksVal: "hmac_secret_sha256_key",
		},
		{
			name:     "Standard PWD var",
			env:      "MY_DB_PWD=production_root_pass",
			leaksVal: "production_root_pass",
		},
	}

	for _, tt := range vulnerableInputs {
		t.Run(tt.name, func(t *testing.T) {
			masked := MaskEnvVar(tt.env)
			if strings.Contains(masked, tt.leaksVal) {
				t.Errorf("SECURITY LEAK in MaskEnvVar(%q): leaked secret %q, got output %q", tt.env, tt.leaksVal, masked)
			}
		})
	}
}

func FuzzMaskEnvVar(f *testing.F) {
	f.Add("KEY=value")
	f.Add("SECRET=mysecret")
	f.Add("DATABASE_URL=postgres://user:pass@localhost:5432/db")
	f.Add("NO_EQUALS_LINE")
	f.Add("=")
	f.Add("====")
	f.Add("A=B=C")
	f.Add("")

	f.Fuzz(func(_ *testing.T, entry string) {
		// Must never panic on arbitrary input
		_ = MaskEnvVar(entry)
	})
}
