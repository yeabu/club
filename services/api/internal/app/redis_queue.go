package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const scanTaskStream = "club:scan:tasks"

func enqueueScanTask(ctx context.Context, config RedisConfig, payload ScanQueuePayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	conn, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", config.Addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", err
	}
	reader := bufio.NewReader(conn)
	if config.Password != "" {
		if _, err := redisCommand(conn, "AUTH", config.Password); err != nil {
			return "", err
		}
		if _, err := readRedisSimple(reader); err != nil {
			return "", err
		}
	}
	if config.DB > 0 {
		if _, err := redisCommand(conn, "SELECT", strconv.Itoa(config.DB)); err != nil {
			return "", err
		}
		if _, err := readRedisSimple(reader); err != nil {
			return "", err
		}
	}
	if _, err := redisCommand(conn,
		"XADD", scanTaskStream, "*",
		"taskId", payload.TaskID,
		"templateId", payload.TemplateID,
		"templateVersion", strconv.Itoa(payload.TemplateVersion),
		"fileKeys", strings.Join(payload.FileKeys, ","),
		"payload", string(raw),
	); err != nil {
		return "", err
	}
	return readRedisBulkOrSimple(reader)
}

func redisCommand(conn net.Conn, args ...string) (int, error) {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		builder.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}
	return fmt.Fprint(conn, builder.String())
}

func readRedisSimple(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "-") {
		return "", fmt.Errorf("redis error: %s", strings.TrimPrefix(line, "-"))
	}
	if strings.HasPrefix(line, "+") {
		return strings.TrimPrefix(line, "+"), nil
	}
	return line, nil
}

func readRedisBulkOrSimple(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "-") {
		return "", fmt.Errorf("redis error: %s", strings.TrimPrefix(line, "-"))
	}
	if strings.HasPrefix(line, "+") || strings.HasPrefix(line, ":") {
		return line[1:], nil
	}
	if !strings.HasPrefix(line, "$") {
		return line, nil
	}
	length, err := strconv.Atoi(strings.TrimPrefix(line, "$"))
	if err != nil {
		return "", err
	}
	if length < 0 {
		return "", nil
	}
	buffer := make([]byte, length+2)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return "", err
	}
	return string(buffer[:length]), nil
}

func scanQueuePayload(task ScanJob, template PaperTemplate) ScanQueuePayload {
	fileKeys := make([]string, 0, len(task.Files))
	for _, file := range task.Files {
		fileKeys = append(fileKeys, file.Key)
	}
	templateVersion := task.TemplateVersion
	if templateVersion == 0 {
		templateVersion = template.Version
	}
	return ScanQueuePayload{
		TaskID:          task.ID,
		Title:           task.Title,
		ClassName:       task.ClassName,
		TemplateID:      task.TemplateID,
		TemplateVersion: templateVersion,
		Pages:           task.Pages,
		FileKeys:        fileKeys,
		CreatedAt:       time.Now().Format(time.RFC3339),
	}
}
