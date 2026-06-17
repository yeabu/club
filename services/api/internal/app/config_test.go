package app

import "testing"

func TestConfigPublicRedactsSecrets(t *testing.T) {
	t.Setenv("MYSQL_PASSWORD", "secret")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("MINIO_ACCESS_KEY", "access")
	t.Setenv("MINIO_SECRET_KEY", "secret")
	t.Setenv("OBS_ACCESS_KEY_ID", "access")
	t.Setenv("OBS_SECRET_ACCESS_KEY", "secret")

	cfg := LoadConfig()
	public := cfg.Public()

	mysql := public["mysql"].(map[string]any)
	if mysql["password"] != nil {
		t.Fatal("mysql password must not be exposed")
	}
	if mysql["passwordProvided"] != true {
		t.Fatal("mysql password presence should be exposed as a boolean")
	}

	obs := public["obs"].(map[string]any)
	if obs["secretAccessKey"] != nil {
		t.Fatal("obs secret access key must not be exposed")
	}
	if obs["secretProvided"] != true {
		t.Fatal("obs secret presence should be exposed as a boolean")
	}
}

func TestEndpointAddressAddsDefaultPort(t *testing.T) {
	got := endpointAddress("https://example.com/", "443")
	if got != "example.com:443" {
		t.Fatalf("expected example.com:443, got %s", got)
	}
}

func TestEndpointAddressKeepsExplicitPort(t *testing.T) {
	got := endpointAddress("127.0.0.1:9000", "443")
	if got != "127.0.0.1:9000" {
		t.Fatalf("expected explicit port to be preserved, got %s", got)
	}
}
