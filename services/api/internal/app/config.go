package app

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv        string      `json:"appEnv"`
	Port          string      `json:"port"`
	MySQL         MySQLConfig `json:"mysql"`
	Redis         RedisConfig `json:"redis"`
	StorageDriver string      `json:"storageDriver"`
	MinIO         MinIOConfig `json:"minio"`
	OBS           OBSConfig   `json:"obs"`
}

type MySQLConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"-"`
	Database string `json:"database"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"-"`
	DB       int    `json:"db"`
}

type MinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"-"`
	SecretKey string `json:"-"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"useSSL"`
}

type OBSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"-"`
	SecretAccessKey string `json:"-"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
}

func LoadConfig() Config {
	loadDotEnv(".env.local")

	return Config{
		AppEnv: env("APP_ENV", "development"),
		Port:   env("PORT", "8080"),
		MySQL: MySQLConfig{
			Host:     env("MYSQL_HOST", "127.0.0.1"),
			Port:     env("MYSQL_PORT", "3306"),
			User:     env("MYSQL_USER", "root"),
			Password: env("MYSQL_PASSWORD", ""),
			Database: env("MYSQL_DATABASE", "club"),
		},
		Redis: RedisConfig{
			Addr:     env("REDIS_ADDR", "127.0.0.1:6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		StorageDriver: env("STORAGE_DRIVER", "minio"),
		MinIO: MinIOConfig{
			Endpoint:  env("MINIO_ENDPOINT", "127.0.0.1:9000"),
			AccessKey: env("MINIO_ACCESS_KEY", ""),
			SecretKey: env("MINIO_SECRET_KEY", ""),
			Bucket:    env("MINIO_BUCKET", "club-dev"),
			UseSSL:    envBool("MINIO_USE_SSL", false),
		},
		OBS: OBSConfig{
			Endpoint:        env("OBS_ENDPOINT", ""),
			AccessKeyID:     env("OBS_ACCESS_KEY_ID", ""),
			SecretAccessKey: env("OBS_SECRET_ACCESS_KEY", ""),
			Bucket:          env("OBS_BUCKET", ""),
			Region:          env("OBS_REGION", ""),
		},
	}
}

func (c Config) Public() map[string]any {
	return map[string]any{
		"appEnv":        c.AppEnv,
		"port":          c.Port,
		"storageDriver": c.StorageDriver,
		"mysql": map[string]any{
			"host":             c.MySQL.Host,
			"port":             c.MySQL.Port,
			"user":             c.MySQL.User,
			"database":         c.MySQL.Database,
			"passwordProvided": c.MySQL.Password != "",
		},
		"redis": map[string]any{
			"addr":             c.Redis.Addr,
			"db":               c.Redis.DB,
			"passwordProvided": c.Redis.Password != "",
		},
		"minio": map[string]any{
			"endpoint":          c.MinIO.Endpoint,
			"bucket":            c.MinIO.Bucket,
			"useSSL":            c.MinIO.UseSSL,
			"accessKeyProvided": c.MinIO.AccessKey != "",
			"secretProvided":    c.MinIO.SecretKey != "",
		},
		"obs": map[string]any{
			"endpoint":          c.OBS.Endpoint,
			"bucket":            c.OBS.Bucket,
			"region":            c.OBS.Region,
			"accessKeyProvided": c.OBS.AccessKeyID != "",
			"secretProvided":    c.OBS.SecretAccessKey != "",
		},
	}
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(env(key, ""))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}

func loadDotEnv(name string) {
	path, ok := findUp(name)
	if !ok {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func findUp(name string) (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
