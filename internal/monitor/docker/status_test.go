package docker

import "testing"

func TestParseHealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"exact healthy", "healthy", "healthy"},
		{"exact unhealthy", "unhealthy", "unhealthy"},
		{"exact starting", "starting", "starting"},
		{"status with (healthy)", "Up 2 hours (healthy)", "healthy"},
		{"status with (health: healthy)", "Up 2 hours (health: healthy)", "healthy"},
		{"status with (unhealthy)", "Up 35 minutes (unhealthy)", "unhealthy"},
		{"status with (health: unhealthy)", "Up 35 minutes (health: unhealthy)", "unhealthy"},
		{"status with (health: starting)", "Up 12 seconds (health: starting)", "starting"},
		{"status with (starting)", "Up 12 seconds (starting)", "starting"},
		{"status without healthcheck", "Up 4 days", ""},
		{"empty string", "", ""},
		{"whitespace string", "   ", ""},
		{"case insensitive HEALTHY", "HEALTHY", "healthy"},
		{"case insensitive (HEALTHY)", "Up 1 hour (HEALTHY)", "healthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHealthStatus(tt.input)
			if got != tt.expected {
				t.Errorf("ParseHealthStatus(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseExitCode(t *testing.T) {
	tests := []struct {
		input    string
		hasCode  bool
		expected int
	}{
		{"Exited (0) 5 minutes ago", true, 0},
		{"Exited (1) 2 minutes ago", true, 1},
		{"Exited (137) 10 seconds ago", true, 137},
		{"exited (255) 1 hour ago", true, 255},
		{"exited (ExitCode: 137)", true, 137},
		{"stopped (ExitCode: 1)", true, 1},
		{"exited (ExitCode: 0)", true, 0},
		{"Up 2 hours", false, 0},
		{"Dead", false, 0},
		{"", false, 0},
		{"Exited ()", false, 0},
		{"Exited (abc)", false, 0},
		{"exitcode: 143", true, 143},
		{"exitcode:", false, 0},
		{"exitcode: notanumber", false, 0},
		{"Exited (notanumber)", false, 0},
		{"exited (exitcode: 99)", true, 99},
	}

	for _, tt := range tests {
		has, code := ParseExitCode(tt.input)
		if has != tt.hasCode || code != tt.expected {
			t.Errorf("ParseExitCode(%q) = (%v, %d); want (%v, %d)", tt.input, has, code, tt.hasCode, tt.expected)
		}
	}
}

func TestIsContainerFailed(t *testing.T) {
	tests := []struct {
		state    string
		status   string
		failed   bool
		exitCode int
	}{
		{"dead", "", true, -1},
		{"DEAD", "Dead", true, -1},
		{"exited", "Exited (1) 5 seconds ago", true, 1},
		{"exited", "Exited (137) 10 seconds ago", true, 137},
		{"exited", "exited (ExitCode: 137)", true, 137},
		{"exited", "exited (ExitCode: 0)", false, 0},
		{"exited", "Exited (0) 10 minutes ago", false, 0},
		{"running", "Up 2 hours", false, 0},
		{"paused", "Paused", false, 0},
		{"restarting", "Restarting (1) 2s ago", false, 0},
	}

	for _, tt := range tests {
		failed, code := IsContainerFailed(tt.state, tt.status)
		if failed != tt.failed || code != tt.exitCode {
			t.Errorf("IsContainerFailed(%q, %q) = (%v, %d); want (%v, %d)", tt.state, tt.status, failed, code, tt.failed, tt.exitCode)
		}
	}
}
