package app

import (
	"bytes"
	"context"
	"database/sql"
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
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
	config := app.activeAIProviderConfig(r.Context())
	task, err := app.store.DispatchAITask(r.Context(), r.PathValue("taskID"), config)
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
	config := app.activeAIProviderConfig(r.Context())
	if config.CallbackSecret != "" && r.Header.Get("X-AI-Callback-Secret") != config.CallbackSecret {
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

func (app *App) handleAIProviders(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	current, err := app.store.ActiveAIProvider(r.Context())
	if err != nil {
		if err == sql.ErrNoRows {
			current = aiProviderSettingFromConfig(app.config.AIProvider)
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ai provider query failed"})
			return
		}
	}
	saved, err := app.store.AIProviderSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ai provider query failed"})
		return
	}
	writeJSON(w, http.StatusOK, AIProviderSettingsResponse{
		Current:  scrubAIProviderSetting(current),
		Channels: mergeAIProviderChannels(saved, current),
	})
}

func (app *App) handleSaveAIProvider(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req AIProviderSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	setting, err := app.store.SaveAIProviderSetting(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	app.config.AIProvider = aiProviderConfigFromSetting(setting)
	writeJSON(w, http.StatusOK, scrubAIProviderSetting(setting))
}

func (app *App) handleDeleteAIProvider(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	providerName := strings.TrimSpace(r.PathValue("providerName"))
	if providerName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider name is required"})
		return
	}
	deleted, err := app.store.DeleteAIProviderSetting(r.Context(), providerName)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ai provider setting not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ai provider delete failed"})
		return
	}
	if deleted.Active {
		app.config.AIProvider = AIProviderConfig{Name: "generic-http", TimeoutSeconds: 30}
		if current, err := app.store.ActiveAIProvider(r.Context()); err == nil {
			app.config.AIProvider = aiProviderConfigFromSetting(current)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "deletedName": deleted.Name})
}

func (app *App) activeAIProviderConfig(ctx context.Context) AIProviderConfig {
	if app.store != nil {
		if setting, err := app.store.ActiveAIProvider(ctx); err == nil {
			return aiProviderConfigFromSetting(setting)
		}
	}
	return app.config.AIProvider
}

func aiProviderConfigFromSetting(setting AIProviderSetting) AIProviderConfig {
	config := AIProviderConfig{
		Name:           setting.Name,
		BaseURL:        setting.BaseURL,
		APIKey:         setting.APIKey,
		Model:          setting.Model,
		TimeoutSeconds: setting.TimeoutSeconds,
		CallbackSecret: setting.CallbackSecret,
	}
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = 30
	}
	normalizeAIProviderConfig(&config)
	return config
}

func aiProviderSettingFromConfig(config AIProviderConfig) AIProviderSetting {
	normalizeAIProviderConfig(&config)
	item := AIProviderSetting{
		ID:             "env_" + aiProviderName(config),
		Name:           aiProviderName(config),
		DisplayName:    aiProviderDisplayName(aiProviderName(config)),
		BaseURL:        config.BaseURL,
		Model:          config.Model,
		APIKey:         config.APIKey,
		TimeoutSeconds: config.TimeoutSeconds,
		CallbackSecret: config.CallbackSecret,
		Active:         true,
	}
	if item.TimeoutSeconds <= 0 {
		item.TimeoutSeconds = 30
	}
	item.APIKeyProvided = strings.TrimSpace(item.APIKey) != ""
	item.CallbackSecretProvided = strings.TrimSpace(item.CallbackSecret) != ""
	item.Configured = item.BaseURL != "" && item.Model != "" && item.APIKeyProvided
	return item
}

func scrubAIProviderSetting(item AIProviderSetting) AIProviderSetting {
	item.APIKeyProvided = strings.TrimSpace(item.APIKey) != "" || item.APIKeyProvided
	item.CallbackSecretProvided = strings.TrimSpace(item.CallbackSecret) != "" || item.CallbackSecretProvided
	item.Configured = item.BaseURL != "" && item.Model != "" && item.APIKeyProvided
	item.APIKey = ""
	item.CallbackSecret = ""
	return item
}

func mergeAIProviderChannels(saved []AIProviderSetting, current AIProviderSetting) []AIProviderSetting {
	channelByName := map[string]AIProviderSetting{}
	for _, item := range defaultAIProviderChannels() {
		channelByName[item.Name] = item
	}
	for _, item := range saved {
		item = scrubAIProviderSetting(item)
		channelByName[item.Name] = item
	}
	current = scrubAIProviderSetting(current)
	if current.Name != "" {
		current.Active = true
		channelByName[current.Name] = current
	}
	order := []string{"deepseek", "openai", "qwen", "anthropic", "generic-http"}
	channels := make([]AIProviderSetting, 0, len(channelByName))
	for _, name := range order {
		if item, ok := channelByName[name]; ok {
			if item.DisplayName == "" {
				item.DisplayName = aiProviderDisplayName(item.Name)
			}
			channels = append(channels, item)
			delete(channelByName, name)
		}
	}
	for _, item := range channelByName {
		if item.DisplayName == "" {
			item.DisplayName = aiProviderDisplayName(item.Name)
		}
		channels = append(channels, item)
	}
	return channels
}

func defaultAIProviderChannels() []AIProviderSetting {
	return []AIProviderSetting{
		{ID: "channel_deepseek", Name: "deepseek", DisplayName: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat", TimeoutSeconds: 30},
		{ID: "channel_openai", Name: "openai", DisplayName: "OpenAI / ChatGPT", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini", TimeoutSeconds: 30},
		{ID: "channel_qwen", Name: "qwen", DisplayName: "通义千问", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Model: "qwen-plus", TimeoutSeconds: 30},
		{ID: "channel_anthropic", Name: "anthropic", DisplayName: "Anthropic Claude", BaseURL: "https://api.anthropic.com/v1", Model: "claude-3-5-sonnet-latest", TimeoutSeconds: 30},
		{ID: "channel_generic", Name: "generic-http", DisplayName: "OpenAI 兼容接口", BaseURL: "", Model: "", TimeoutSeconds: 30},
	}
}

func aiProviderDisplayName(name string) string {
	switch strings.TrimSpace(name) {
	case "deepseek":
		return "DeepSeek"
	case "openai":
		return "OpenAI / ChatGPT"
	case "qwen":
		return "通义千问"
	case "anthropic":
		return "Anthropic Claude"
	default:
		if strings.TrimSpace(name) == "" {
			return "OpenAI 兼容接口"
		}
		return name
	}
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
		return s.updateAITask(ctx, id, "failed", provider, standardAIResult(task, "failed", provider, result, fmt.Sprintf("provider returned HTTP %d", response.StatusCode)), fmt.Sprintf("provider returned HTTP %d", response.StatusCode))
	}
	return s.updateAITask(ctx, id, "processing", provider, standardAIResult(task, "processing", provider, result, ""), "")
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
	return s.updateAITask(ctx, id, normalizedStatus, task.Provider, standardAIResult(task, normalizedStatus, task.Provider, result, errorMessage), errorMessage)
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

func standardAIResult(task AITaskRecord, status string, provider string, raw map[string]any, errorMessage string) map[string]any {
	if raw == nil {
		raw = map[string]any{}
	}
	outputs := map[string]any{}
	metrics := map[string]any{}
	if value, ok := raw["outputs"].(map[string]any); ok {
		outputs = value
	} else if value, ok := raw["result"].(map[string]any); ok {
		outputs = value
	} else if value, ok := raw["body"].(map[string]any); ok {
		outputs = value
	} else if len(raw) > 0 {
		outputs = raw
	}
	if value, ok := raw["metrics"].(map[string]any); ok {
		metrics = value
	}
	modelVersion := stringValue(raw["modelVersion"])
	if modelVersion == "" {
		modelVersion = stringValue(raw["model"])
	}
	if provider == "" {
		provider = task.Provider
	}
	return map[string]any{
		"schemaVersion": "ai-result.v1",
		"task": map[string]any{
			"id":        task.ID,
			"taskType":  task.TaskType,
			"ownerType": task.OwnerType,
			"ownerId":   task.OwnerID,
		},
		"provider":     provider,
		"status":       status,
		"modelVersion": modelVersion,
		"outputs":      outputs,
		"metrics":      metrics,
		"errorMessage": errorMessage,
		"raw":          raw,
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

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error any `json:"error,omitempty"`
}

type anthropicChatRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	System      string              `json:"system"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
}

type anthropicChatResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error any `json:"error,omitempty"`
}

type flexibleTemplateAISuggestionResponse struct {
	PaperName          string                       `json:"paperName"`
	QuestionCount      int                          `json:"questionCount"`
	TotalScore         int                          `json:"totalScore"`
	SuggestedQuestions []flexibleAIQuestionTemplate `json:"suggestedQuestions"`
	ReviewRequired     bool                         `json:"reviewRequired"`
	Source             string                       `json:"source"`
}

type flexibleAIQuestionTemplate struct {
	ID             any              `json:"id"`
	No             any              `json:"no"`
	Type           string           `json:"type"`
	Score          float64          `json:"score"`
	StandardAnswer any              `json:"standardAnswer"`
	ScoringRules   any              `json:"scoringRules"`
	Knowledge      any              `json:"knowledge"`
	Region         flexibleAIRegion `json:"region"`
}

type flexibleAIRegion struct {
	Page   any `json:"page"`
	X      any `json:"x"`
	Y      any `json:"y"`
	Width  any `json:"width"`
	Height any `json:"height"`
}

func (app *App) generateTemplateAISuggestionWithProvider(ctx context.Context, template PaperTemplate, req TemplateAISuggestionRequest) (TemplateAISuggestionResponse, error) {
	config := app.activeAIProviderConfig(ctx)
	provider := aiProviderName(config)
	if strings.TrimSpace(config.BaseURL) == "" || strings.TrimSpace(config.APIKey) == "" || strings.TrimSpace(config.Model) == "" {
		return TemplateAISuggestionResponse{}, errAIProviderConfigRequired
	}
	prompt, err := templateAISuggestionPrompt(template, req)
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	if aiProviderUsesAnthropic(config) {
		return app.generateTemplateAISuggestionWithAnthropic(ctx, template, req, prompt, config)
	}
	endpoint := aiChatCompletionsURL(config.BaseURL)
	payload := openAIChatRequest{
		Model: strings.TrimSpace(config.Model),
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "你是阅卷平台的答题卡结构化生成助手。只输出 JSON，不输出解释、Markdown 或代码块。",
			},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TemplateAISuggestionResponse{}, fmt.Errorf("provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var chatResp openAIChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	if len(chatResp.Choices) == 0 || strings.TrimSpace(chatResp.Choices[0].Message.Content) == "" {
		return TemplateAISuggestionResponse{}, fmt.Errorf("provider returned empty choices")
	}
	content := extractJSONDocument(chatResp.Choices[0].Message.Content)
	suggestion, err := decodeTemplateAISuggestionContent(content)
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	if len(suggestion.SuggestedQuestions) == 0 {
		return TemplateAISuggestionResponse{}, fmt.Errorf("provider returned no suggested questions")
	}
	return normalizeTemplateAISuggestionResponse(suggestion, template, req, provider), nil
}

func (app *App) generateTemplateAISuggestionWithAnthropic(ctx context.Context, template PaperTemplate, req TemplateAISuggestionRequest, prompt string, config AIProviderConfig) (TemplateAISuggestionResponse, error) {
	provider := aiProviderName(config)
	payload := anthropicChatRequest{
		Model:       strings.TrimSpace(config.Model),
		MaxTokens:   4096,
		System:      "你是阅卷平台的答题卡结构化生成助手。只输出 JSON，不输出解释、Markdown 或代码块。",
		Messages:    []openAIChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodPost, aiAnthropicMessagesURL(config.BaseURL), bytes.NewReader(body))
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)
	httpReq.Header.Set("X-API-Key", config.APIKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TemplateAISuggestionResponse{}, fmt.Errorf("provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var chatResp anthropicChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	content := ""
	for _, item := range chatResp.Content {
		if strings.TrimSpace(item.Text) != "" {
			content = item.Text
			break
		}
	}
	if strings.TrimSpace(content) == "" {
		return TemplateAISuggestionResponse{}, fmt.Errorf("provider returned empty content")
	}
	suggestion, err := decodeTemplateAISuggestionContent(extractJSONDocument(content))
	if err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	if len(suggestion.SuggestedQuestions) == 0 {
		return TemplateAISuggestionResponse{}, fmt.Errorf("provider returned no suggested questions")
	}
	return normalizeTemplateAISuggestionResponse(suggestion, template, req, provider), nil
}

func decodeTemplateAISuggestionContent(content string) (TemplateAISuggestionResponse, error) {
	var strict TemplateAISuggestionResponse
	if err := json.Unmarshal([]byte(content), &strict); err == nil {
		return strict, nil
	}
	var flexible flexibleTemplateAISuggestionResponse
	if err := json.Unmarshal([]byte(content), &flexible); err != nil {
		return TemplateAISuggestionResponse{}, err
	}
	resp := TemplateAISuggestionResponse{
		PaperName:      flexible.PaperName,
		QuestionCount:  flexible.QuestionCount,
		TotalScore:     flexible.TotalScore,
		ReviewRequired: flexible.ReviewRequired,
		Source:         flexible.Source,
	}
	for _, question := range flexible.SuggestedQuestions {
		resp.SuggestedQuestions = append(resp.SuggestedQuestions, QuestionTemplate{
			ID:             aiAnyString(question.ID),
			No:             aiAnyString(question.No),
			Type:           question.Type,
			Score:          question.Score,
			StandardAnswer: aiAnyString(question.StandardAnswer),
			ScoringRules:   aiAnyStringSlice(question.ScoringRules),
			Knowledge:      aiAnyStringSlice(question.Knowledge),
			Region: Region{
				Page:   aiAnyInt(question.Region.Page),
				X:      aiAnyInt(question.Region.X),
				Y:      aiAnyInt(question.Region.Y),
				Width:  aiAnyInt(question.Region.Width),
				Height: aiAnyInt(question.Region.Height),
			},
		})
	}
	return resp, nil
}

func aiAnyString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func aiAnyStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		items := []string{}
		for _, item := range v {
			if text := aiAnyString(item); text != "" {
				items = append(items, text)
			}
		}
		return items
	case string:
		if strings.TrimSpace(v) == "" {
			return []string{}
		}
		return []string{strings.TrimSpace(v)}
	default:
		return []string{}
	}
}

func aiAnyInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &parsed); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func aiChatCompletionsURL(baseURL string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(endpoint)
	switch {
	case strings.HasSuffix(lower, "/chat/completions"):
		return endpoint
	case strings.HasSuffix(lower, "/v1"):
		return endpoint + "/chat/completions"
	case strings.Contains(lower, "api.deepseek.com"):
		return endpoint + "/chat/completions"
	case strings.Contains(lower, "api.openai.com") || strings.Contains(lower, "dashscope.aliyuncs.com"):
		return endpoint + "/v1/chat/completions"
	default:
		return endpoint
	}
}

func aiAnthropicMessagesURL(baseURL string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(endpoint)
	switch {
	case strings.HasSuffix(lower, "/messages"):
		return endpoint
	case strings.HasSuffix(lower, "/v1"):
		return endpoint + "/messages"
	default:
		return endpoint + "/v1/messages"
	}
}

func aiProviderUsesAnthropic(config AIProviderConfig) bool {
	name := strings.ToLower(strings.TrimSpace(config.Name))
	baseURL := strings.ToLower(strings.TrimSpace(config.BaseURL))
	return strings.Contains(name, "anthropic") || strings.Contains(baseURL, "anthropic")
}

func templateAISuggestionPrompt(template PaperTemplate, req TemplateAISuggestionRequest) (string, error) {
	if req.PaperName == "" {
		req.PaperName = template.Name
	}
	if req.Subject == "" {
		req.Subject = template.Subject
	}
	if req.Grade == "" {
		req.Grade = template.Grade
	}
	if req.TargetScore == 0 {
		req.TargetScore = template.TotalScore
	}
	existing, err := json.Marshal(template.Questions)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`你是小学数学答题卡版式设计和题区结构化专家。请根据试卷生成可打印、可扫描识别的答题卡题区 JSON。

试卷信息：
- 试卷名称：%s
- 学科：%s
- 年级：%s
- 满分：%d
- 试卷来源文件 URL 或对象键：%s
- 已有模板题区参考：%s

目标版式参考：
- A4 纵向答题卡，页眉保留标题、完成时间/满分、学校、班级、姓名、座号、考场、缺考标记。
- 第 1 页顶部集中放选择题涂点区：例如 1-7 题每题一格，选项 A/B/C/D 横向排列。
- 非选择题按原试卷题号连续排列，不要漏题、不要改变题号。
- 二年级数学答题卡要尽量像正式纸质答题卡：填空题用括号/短横线/小方格，计算题用大方框或方格纸区域，作图题用网格/作图框，数轴题用横向作图区，应用题用横线作答区，统计题保留条形统计图/分类框/补充作答区。
- 不要把所有非选择题做成同一种大空白框；每个题区高度要匹配该题作答量。

题型判定规则：
- 单项选择题使用 single_choice；多项选择题使用 multiple_choice；判断题使用 judge。
- 只需填写数字、单位、序号、括号的题使用 fill_blank。
- 口算、竖式、列式计算、验算使用 calculation。
- 看图列式、应用题、说明理由、提出问题并解答使用 subjective。
- 作文类才使用 essay；数学卷通常不要使用 essay。

布局规则：
1. region 使用答题卡画布坐标，画布宽 760、高 900；page 从 1 开始。
2. 页眉和考生信息区域占用 y < 145，题目区域不要放到 y < 150。
3. 每页左右边距 45-70；题区宽度通常 620-680。
4. 选择题区域高度 70-120；填空题 45-90；计算题 100-170；作图/数轴/统计题 150-260；应用题 170-320。
5. 题区之间至少留 12 像素间距，不要重叠；跨页时从下一页 y=150 继续。
6. 如果原卷有 1-7 选择题、8-22 非选择题，应返回 22 个题区，而不是只返回几个大题区。
7. 分值未知时按题目作答量合理分配，并让 totalScore 等于满分；不要让某一道题异常占用 50 分以上，除非试卷明确如此。

返回 JSON 要求：
1. 只输出 JSON，不要输出解释、Markdown 或代码块。
2. 返回对象字段必须包含 paperName、questionCount、totalScore、suggestedQuestions、reviewRequired、source。
3. suggestedQuestions 是题目数组，每题字段必须包含 id、no、type、score、standardAnswer、scoringRules、knowledge、region。
4. id 和 no 必须是字符串；score 必须是数字；scoringRules 和 knowledge 必须是字符串数组。
5. source 固定填写 third-party-ai。
6. 如果无法直接读取来源文件，请依据试卷名称、学科、年级、满分、已有题区参考和上述版式规则生成“待教师复核”的完整答题卡结构，不要凭空编写过多标准答案。`, req.PaperName, req.Subject, req.Grade, req.TargetScore, req.SourceFileURL, string(existing)), nil
}

func normalizeTemplateAISuggestionResponse(resp TemplateAISuggestionResponse, template PaperTemplate, req TemplateAISuggestionRequest, provider string) TemplateAISuggestionResponse {
	if resp.PaperName == "" {
		resp.PaperName = req.PaperName
	}
	if resp.PaperName == "" {
		resp.PaperName = template.Name
	}
	for index := range resp.SuggestedQuestions {
		q := &resp.SuggestedQuestions[index]
		q.ID = templateRecordID("ai_q", q.ID)
		if q.No == "" {
			q.No = fmt.Sprintf("%d", index+1)
		}
		q.Type = normalizeQuestionType(q.Type)
		if q.Score <= 0 {
			q.Score = defaultQuestionScore(q.Type)
		}
		if q.ScoringRules == nil {
			q.ScoringRules = []string{}
		}
		if q.Knowledge == nil {
			q.Knowledge = []string{}
		}
		if q.Region.Page <= 0 || q.Region.Width <= 0 || q.Region.Height <= 0 {
			q.Region = defaultQuestionRegion(index, q.Type)
		}
	}
	resp.QuestionCount = len(resp.SuggestedQuestions)
	if resp.TotalScore <= 0 {
		total := 0
		for _, question := range resp.SuggestedQuestions {
			total += int(question.Score + 0.5)
		}
		resp.TotalScore = total
	}
	resp.ReviewRequired = true
	resp.Source = "third-party-ai"
	resp.Provider = provider
	return resp
}

func normalizeQuestionType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(normalized, "multiple") || strings.Contains(normalized, "多选"):
		return "multiple_choice"
	case strings.Contains(normalized, "single") || strings.Contains(normalized, "choice") || strings.Contains(normalized, "选择"):
		return "single_choice"
	case strings.Contains(normalized, "judge") || strings.Contains(normalized, "判断"):
		return "judge"
	case strings.Contains(normalized, "fill") || strings.Contains(normalized, "blank") || strings.Contains(normalized, "填空"):
		return "fill_blank"
	case strings.Contains(normalized, "calc") || strings.Contains(normalized, "计算"):
		return "calculation"
	case strings.Contains(normalized, "essay") || strings.Contains(normalized, "作文"):
		return "essay"
	case normalized == "single_choice" || normalized == "multiple_choice" || normalized == "fill_blank" || normalized == "calculation" || normalized == "subjective":
		return normalized
	default:
		return "subjective"
	}
}

func defaultQuestionScore(questionType string) float64 {
	switch questionType {
	case "single_choice", "judge":
		return 2
	case "multiple_choice", "fill_blank":
		return 4
	case "essay":
		return 30
	default:
		return 8
	}
}

func defaultQuestionRegion(index int, questionType string) Region {
	height := 96
	if questionType == "subjective" || questionType == "calculation" || questionType == "essay" {
		height = 150
	}
	offset := index * 120
	page := 1 + offset/700
	return Region{Page: page, X: 90, Y: 170 + (offset % 700), Width: 580, Height: height}
}

func extractJSONDocument(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end >= start {
		return trimmed[start : end+1]
	}
	return trimmed
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
