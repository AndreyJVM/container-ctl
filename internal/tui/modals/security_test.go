package modals

import (
	"strings"
	"testing"
)

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
			env:      "APP_PWD=user_password_42",
			leaksVal: "user_password_42",
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
