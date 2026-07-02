package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppEnv        string           `json:"appEnv" yaml:"appEnv"`
	Port          string           `json:"port" yaml:"port"`
	MySQL         MySQLConfig      `json:"mysql" yaml:"mysql"`
	Redis         RedisConfig      `json:"redis" yaml:"redis"`
	StorageDriver string           `json:"storageDriver" yaml:"storageDriver"`
	MinIO         MinIOConfig      `json:"minio" yaml:"minio"`
	OBS           OBSConfig        `json:"obs" yaml:"obs"`
	AIProvider    AIProviderConfig `json:"aiProvider" yaml:"aiProvider"`
}

type MySQLConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     string `json:"port" yaml:"port"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"-" yaml:"password"`
	Database string `json:"database" yaml:"database"`
}

type RedisConfig struct {
	Addr     string `json:"addr" yaml:"addr"`
	Password string `json:"-" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

type MinIOConfig struct {
	Endpoint  string `json:"endpoint" yaml:"endpoint"`
	AccessKey string `json:"-" yaml:"accessKey"`
	SecretKey string `json:"-" yaml:"secretKey"`
	Bucket    string `json:"bucket" yaml:"bucket"`
	UseSSL    bool   `json:"useSSL" yaml:"useSSL"`
}

type OBSConfig struct {
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"-" yaml:"accessKeyId"`
	SecretAccessKey string `json:"-" yaml:"secretAccessKey"`
	Bucket          string `json:"bucket" yaml:"bucket"`
	Region          string `json:"region" yaml:"region"`
}

type AIProviderConfig struct {
	Name           string `json:"name" yaml:"name"`
	BaseURL        string `json:"baseUrl" yaml:"baseUrl"`
	APIKey         string `json:"-" yaml:"apiKey"`
	TimeoutSeconds int    `json:"timeoutSeconds" yaml:"timeoutSeconds"`
	CallbackSecret string `json:"-" yaml:"callbackSecret"`
}

func LoadConfig() Config {
	config := Config{
		AppEnv: "development",
		Port:   "8080",
		MySQL: MySQLConfig{
			Host:     "127.0.0.1",
			Port:     "3306",
			User:     "root",
			Database: "club",
		},
		Redis: RedisConfig{
			Addr: "127.0.0.1:6379",
		},
		StorageDriver: "minio",
		MinIO: MinIOConfig{
			Endpoint: "127.0.0.1:9000",
			Bucket:   "club-dev",
		},
		AIProvider: AIProviderConfig{
			Name:           "generic-http",
			TimeoutSeconds: 30,
		},
	}
	loadYAMLConfig(filepath.Join("config", "config.yaml"), &config)

	config.AppEnv = env("APP_ENV", config.AppEnv)
	config.Port = env("PORT", config.Port)
	config.MySQL.Host = env("MYSQL_HOST", config.MySQL.Host)
	config.MySQL.Port = env("MYSQL_PORT", config.MySQL.Port)
	config.MySQL.User = env("MYSQL_USER", config.MySQL.User)
	config.MySQL.Password = env("MYSQL_PASSWORD", config.MySQL.Password)
	config.MySQL.Database = env("MYSQL_DATABASE", config.MySQL.Database)
	config.Redis.Addr = env("REDIS_ADDR", config.Redis.Addr)
	config.Redis.Password = env("REDIS_PASSWORD", config.Redis.Password)
	config.Redis.DB = envInt("REDIS_DB", config.Redis.DB)
	config.StorageDriver = env("STORAGE_DRIVER", config.StorageDriver)
	config.MinIO.Endpoint = env("MINIO_ENDPOINT", config.MinIO.Endpoint)
	config.MinIO.AccessKey = env("MINIO_ACCESS_KEY", config.MinIO.AccessKey)
	config.MinIO.SecretKey = env("MINIO_SECRET_KEY", config.MinIO.SecretKey)
	config.MinIO.Bucket = env("MINIO_BUCKET", config.MinIO.Bucket)
	config.MinIO.UseSSL = envBool("MINIO_USE_SSL", config.MinIO.UseSSL)
	config.OBS.Endpoint = env("OBS_ENDPOINT", config.OBS.Endpoint)
	config.OBS.AccessKeyID = env("OBS_ACCESS_KEY_ID", config.OBS.AccessKeyID)
	config.OBS.SecretAccessKey = env("OBS_SECRET_ACCESS_KEY", config.OBS.SecretAccessKey)
	config.OBS.Bucket = env("OBS_BUCKET", config.OBS.Bucket)
	config.OBS.Region = env("OBS_REGION", config.OBS.Region)
	config.AIProvider.Name = env("AI_PROVIDER_NAME", config.AIProvider.Name)
	config.AIProvider.BaseURL = env("AI_PROVIDER_BASE_URL", config.AIProvider.BaseURL)
	config.AIProvider.APIKey = env("AI_PROVIDER_API_KEY", config.AIProvider.APIKey)
	config.AIProvider.TimeoutSeconds = envInt("AI_PROVIDER_TIMEOUT_SECONDS", config.AIProvider.TimeoutSeconds)
	config.AIProvider.CallbackSecret = env("AI_PROVIDER_CALLBACK_SECRET", config.AIProvider.CallbackSecret)
	if config.AIProvider.Name == "" {
		config.AIProvider.Name = "generic-http"
	}
	if config.AIProvider.TimeoutSeconds <= 0 {
		config.AIProvider.TimeoutSeconds = 30
	}
	return config
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
		"aiProvider": map[string]any{
			"name":                   c.AIProvider.Name,
			"baseUrl":                c.AIProvider.BaseURL,
			"timeoutSeconds":         c.AIProvider.TimeoutSeconds,
			"apiKeyProvided":         c.AIProvider.APIKey != "",
			"callbackSecretProvided": c.AIProvider.CallbackSecret != "",
			"configured":             c.AIProvider.BaseURL != "" && c.AIProvider.APIKey != "",
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

func loadYAMLConfig(name string, config *Config) {
	path, ok := findUp(name)
	if !ok {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = yaml.Unmarshal(data, config)
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
