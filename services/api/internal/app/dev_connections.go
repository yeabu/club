package app

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type DevConnectionReport struct {
	MySQL   ConnectionCheck `json:"mysql"`
	Redis   ConnectionCheck `json:"redis"`
	Storage ConnectionCheck `json:"storage"`
}

type ConnectionCheck struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	OK      bool   `json:"ok"`
	Latency string `json:"latency"`
	Error   string `json:"error,omitempty"`
}

func CheckDevConnections(config Config) DevConnectionReport {
	mysqlTarget := net.JoinHostPort(config.MySQL.Host, config.MySQL.Port)
	storageName, storageTarget := storageTarget(config)

	return DevConnectionReport{
		MySQL:   checkTCP("mysql", mysqlTarget, 2*time.Second),
		Redis:   checkRedis(config.Redis, 2*time.Second),
		Storage: checkTCP(storageName, storageTarget, 2*time.Second),
	}
}

func checkTCP(name string, target string, timeout time.Duration) ConnectionCheck {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, timeout)
	latency := time.Since(start)
	if err != nil {
		return ConnectionCheck{Name: name, Target: target, OK: false, Latency: latency.String(), Error: err.Error()}
	}
	_ = conn.Close()
	return ConnectionCheck{Name: name, Target: target, OK: true, Latency: latency.String()}
}

func checkRedis(config RedisConfig, timeout time.Duration) ConnectionCheck {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", config.Addr, timeout)
	latency := time.Since(start)
	if err != nil {
		return ConnectionCheck{Name: "redis", Target: config.Addr, OK: false, Latency: latency.String(), Error: err.Error()}
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return ConnectionCheck{Name: "redis", Target: config.Addr, OK: false, Latency: time.Since(start).String(), Error: err.Error()}
	}

	reader := bufio.NewReader(conn)
	if config.Password != "" {
		if _, err := fmt.Fprintf(conn, "*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(config.Password), config.Password); err != nil {
			return redisError(config.Addr, start, err)
		}
		if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "+OK") {
			return ConnectionCheck{Name: "redis", Target: config.Addr, OK: false, Latency: time.Since(start).String(), Error: "redis auth failed"}
		}
	}

	if config.DB > 0 {
		db := strconv.Itoa(config.DB)
		if _, err := fmt.Fprintf(conn, "*2\r\n$6\r\nSELECT\r\n$%d\r\n%s\r\n", len(db), db); err != nil {
			return redisError(config.Addr, start, err)
		}
		if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "+OK") {
			return ConnectionCheck{Name: "redis", Target: config.Addr, OK: false, Latency: time.Since(start).String(), Error: "redis select db failed"}
		}
	}

	if _, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		return redisError(config.Addr, start, err)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return redisError(config.Addr, start, err)
	}
	if !strings.HasPrefix(line, "+PONG") {
		return ConnectionCheck{Name: "redis", Target: config.Addr, OK: false, Latency: time.Since(start).String(), Error: "unexpected redis response"}
	}

	return ConnectionCheck{Name: "redis", Target: config.Addr, OK: true, Latency: time.Since(start).String()}
}

func redisError(target string, start time.Time, err error) ConnectionCheck {
	return ConnectionCheck{Name: "redis", Target: target, OK: false, Latency: time.Since(start).String(), Error: err.Error()}
}

func storageTarget(config Config) (string, string) {
	if config.StorageDriver == "obs" {
		return "obs", endpointAddress(config.OBS.Endpoint, "443")
	}
	return "minio", endpointAddress(config.MinIO.Endpoint, "9000")
}

func endpointAddress(endpoint string, defaultPort string) string {
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimRight(endpoint, "/")
	if endpoint == "" {
		return net.JoinHostPort("127.0.0.1", defaultPort)
	}
	if _, _, err := net.SplitHostPort(endpoint); err == nil {
		return endpoint
	}
	return net.JoinHostPort(endpoint, defaultPort)
}
