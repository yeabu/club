package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromYAMLAndEnvironmentOverride(t *testing.T) {
	for _, key := range []string{
		"APP_ENV", "PORT", "MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB", "STORAGE_DRIVER", "MINIO_ENDPOINT", "MINIO_ACCESS_KEY",
		"MINIO_SECRET_KEY", "MINIO_BUCKET", "MINIO_USE_SSL", "OBS_ENDPOINT", "OBS_ACCESS_KEY_ID",
		"OBS_SECRET_ACCESS_KEY", "OBS_BUCKET", "OBS_REGION",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("PORT", "9090")

	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`
appEnv: test
port: "8088"
mysql:
  host: db.example.com
  port: "3307"
  user: club
  password: mysql-secret
  database: club_test
redis:
  addr: cache.example.com:6380
  password: redis-secret
  db: 2
storageDriver: obs
minio:
  endpoint: minio.example.com:9000
  bucket: test-bucket
  useSSL: true
obs:
  endpoint: https://obs.example.com
  accessKeyId: access
  secretAccessKey: secret
  bucket: reports
  region: cn-test-1
`)
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "cmd", "server")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	cfg := LoadConfig()
	if cfg.AppEnv != "test" || cfg.Port != "9090" {
		t.Fatalf("unexpected app config: %+v", cfg)
	}
	if cfg.MySQL.Host != "db.example.com" || cfg.MySQL.Port != "3307" || cfg.MySQL.Password != "mysql-secret" {
		t.Fatalf("unexpected mysql config: %+v", cfg.MySQL)
	}
	if cfg.Redis.Addr != "cache.example.com:6380" || cfg.Redis.DB != 2 || cfg.Redis.Password != "redis-secret" {
		t.Fatalf("unexpected redis config: %+v", cfg.Redis)
	}
	if cfg.StorageDriver != "obs" || cfg.OBS.Bucket != "reports" || !cfg.MinIO.UseSSL {
		t.Fatalf("unexpected storage config: %+v", cfg)
	}
}

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
