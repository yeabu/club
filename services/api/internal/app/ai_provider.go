package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var errAIProviderConfigRequired = errors.New("ai provider is not configured")

func (app *App) handleAITasks(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []AITaskRecord{}, "counts": map[string]int{"tasks": 0}})
		return
	}
	items, err := app.store.AITasks(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("taskType"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": map[string]int{"tasks": len(items)}})
}

func (app *App) handleAITask(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	task, err := app.store.AITask(r.Context(), r.PathValue("taskID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (app *App) handleDispatchAITask(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	task, err := app.store.DispatchAITask(r.Context(), r.PathValue("taskID"), app.config.AIProvider)
	if errors.Is(err, errAIProviderConfigRequired) {
		writeJSON(w, http.StatusConflict, task)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (app *App) handleAITaskCallback(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	if app.config.AIProvider.CallbackSecret != "" && r.Header.Get("X-AI-Callback-Secret") != app.config.AIProvider.CallbackSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid callback secret"})
		return
	}
	var req map[string]any
	if decodeJSON(r, &req) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	status := stringValue(req["status"])
	if status == "" {
		status = "succeeded"
	}
	result := normalizeAIResult(req["result"])
	errorMessage := stringValue(req["errorMessage"])
	if errorMessage == "" {
		errorMessage = stringValue(req["error"])
	}
	task, err := app.store.ApplyAITaskCallback(r.Context(), r.PathValue("taskID"), status, result, errorMessage)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Store) AITasks(ctx context.Context, status, taskType string) ([]AITaskRecord, error) {
	query := `SELECT id,task_type,status,COALESCE(provider,''),request_json,result_json,COALESCE(source_object_key,''),COALESCE(source_url,''),COALESCE(owner_type,''),COALESCE(owner_id,''),COALESCE(created_by,''),COALESCE(error_message,''),created_at FROM ai_tasks`
	conditions := []string{}
	args := []any{}
	if strings.TrimSpace(status) != "" {
		normalizedStatus, err := normalizeAITaskStatus(strings.TrimSpace(status))
		if err != nil {
			return nil, err
		}
		if normalizedStatus == "succeeded" {
			conditions = append(conditions, "status IN (?,?)")
			args = append(args, "succeeded", "completed")
		} else {
			conditions = append(conditions, "status=?")
			args = append(args, normalizedStatus)
		}
	}
	if strings.TrimSpace(taskType) != "" {
		conditions = append(conditions, "task_type=?")
		args = append(args, strings.TrimSpace(taskType))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY updated_at DESC, created_at DESC LIMIT 100"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AITaskRecord{}
	for rows.Next() {
		item, err := scanAITask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) AITask(ctx context.Context, id string) (AITaskRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,task_type,status,COALESCE(provider,''),request_json,result_json,COALESCE(source_object_key,''),COALESCE(source_url,''),COALESCE(owner_type,''),COALESCE(owner_id,''),COALESCE(created_by,''),COALESCE(error_message,''),created_at FROM ai_tasks WHERE id=?`, id)
	return scanAITask(row)
}

func (s *Store) DispatchAITask(ctx context.Context, id string, config AIProviderConfig) (AITaskRecord, error) {
	task, err := s.AITask(ctx, id)
	if err != nil {
		return AITaskRecord{}, err
	}
	provider := aiProviderName(config)
	if strings.TrimSpace(config.BaseURL) == "" || strings.TrimSpace(config.APIKey) == "" {
		updated, updateErr := s.updateAITask(ctx, id, "config_required", provider, nil, "AI provider baseUrl and apiKey are required")
		if updateErr != nil {
			return updated, updateErr
		}
		return updated, errAIProviderConfigRequired
	}

	payload := map[string]any{
		"taskId":          task.ID,
		"taskType":        task.TaskType,
		"request":         task.Request,
		"sourceObjectKey": task.SourceObjectKey,
		"sourceUrl":       task.SourceURL,
		"ownerType":       task.OwnerType,
		"ownerId":         task.OwnerID,
		"createdBy":       task.CreatedBy,
		"callbackPath":    fmt.Sprintf("/api/ai/tasks/%s/callback", task.ID),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AITaskRecord{}, err
	}

	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, strings.TrimSpace(config.BaseURL), bytes.NewReader(body))
	if err != nil {
		return s.updateAITask(ctx, id, "failed", provider, nil, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	if config.CallbackSecret != "" {
		req.Header.Set("X-AI-Callback-Secret", config.CallbackSecret)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return s.updateAITask(ctx, id, "failed", provider, nil, err.Error())
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	result := map[string]any{
		"httpStatus": response.StatusCode,
		"body":       parseAIProviderBody(responseBody),
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return s.updateAITask(ctx, id, "failed", provider, result, fmt.Sprintf("provider returned HTTP %d", response.StatusCode))
	}
	return s.updateAITask(ctx, id, "processing", provider, result, "")
}

func (s *Store) ApplyAITaskCallback(ctx context.Context, id, status string, result map[string]any, errorMessage string) (AITaskRecord, error) {
	normalizedStatus, err := normalizeAITaskStatus(status)
	if err != nil {
		return AITaskRecord{}, err
	}
	task, err := s.AITask(ctx, id)
	if err != nil {
		return AITaskRecord{}, err
	}
	if result == nil {
		result = map[string]any{}
	}
	return s.updateAITask(ctx, id, normalizedStatus, task.Provider, result, errorMessage)
}

func (s *Store) updateAITask(ctx context.Context, id, status, provider string, result map[string]any, errorMessage string) (AITaskRecord, error) {
	if provider == "" {
		provider = "generic-http"
	}
	if result == nil {
		_, err := s.db.ExecContext(ctx, `UPDATE ai_tasks SET status=?, provider=?, error_message=? WHERE id=?`, status, provider, errorMessage, id)
		if err != nil {
			return AITaskRecord{}, err
		}
		return s.AITask(ctx, id)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return AITaskRecord{}, err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE ai_tasks SET status=?, provider=?, result_json=?, error_message=? WHERE id=?`, status, provider, string(resultJSON), errorMessage, id)
	if err != nil {
		return AITaskRecord{}, err
	}
	return s.AITask(ctx, id)
}

type aiTaskScanner interface {
	Scan(dest ...any) error
}

func scanAITask(scanner aiTaskScanner) (AITaskRecord, error) {
	var item AITaskRecord
	var requestRaw []byte
	var resultRaw []byte
	var created time.Time
	if err := scanner.Scan(&item.ID, &item.TaskType, &item.Status, &item.Provider, &requestRaw, &resultRaw, &item.SourceObjectKey, &item.SourceURL, &item.OwnerType, &item.OwnerID, &item.CreatedBy, &item.ErrorMessage, &created); err != nil {
		return item, err
	}
	item.Request = parseJSONMap(requestRaw)
	item.Result = parseJSONMap(resultRaw)
	item.CreatedAt = created.Format(time.RFC3339)
	item.Message = aiTaskMessage(item)
	return item, nil
}

func parseJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err == nil && result != nil {
		return result
	}
	return map[string]any{"raw": string(raw)}
}

func normalizeAIResult(value any) map[string]any {
	switch v := value.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return v
	default:
		return map[string]any{"value": v}
	}
}

func parseAIProviderBody(raw []byte) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return string(raw)
}

func aiProviderName(config AIProviderConfig) string {
	if strings.TrimSpace(config.Name) == "" {
		return "generic-http"
	}
	return strings.TrimSpace(config.Name)
}

func normalizeAITaskStatus(status string) (string, error) {
	switch strings.TrimSpace(status) {
	case "pending", "processing", "succeeded", "failed", "config_required":
		return strings.TrimSpace(status), nil
	case "completed":
		return "succeeded", nil
	default:
		return "", fmt.Errorf("unsupported ai task status: %s", status)
	}
}

func aiTaskMessage(item AITaskRecord) string {
	if item.ErrorMessage != "" {
		return item.ErrorMessage
	}
	switch item.Status {
	case "pending":
		return "第三方 AI 任务已创建，等待人工或调度器派发"
	case "config_required":
		return "第三方 AI Provider 未完成配置"
	case "processing":
		return "第三方 AI Provider 已接收任务，等待回调结果"
	case "succeeded", "completed":
		return "第三方 AI 任务已完成"
	case "failed":
		return "第三方 AI 任务执行失败"
	default:
		return ""
	}
}
