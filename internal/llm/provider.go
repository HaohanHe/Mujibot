package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
	"github.com/HaohanHe/mujibot/internal/session"
)

// Provider LLM提供商接口
type Provider interface {
	Chat(messages []session.Message, tools []Tool) (*Response, error)
	ChatStream(messages []session.Message, tools []Tool, callback func(chunk string)) (*Response, error)
	GetModel() string
}

// Tool 工具定义
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function 函数定义
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Response LLM响应
type Response struct {
	Content   string
	ToolCalls []session.ToolCall
	Usage     Usage
}

// Usage 使用量
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// OpenAIProvider OpenAI提供商
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	model      string
	timeout    time.Duration
	maxRetries int
	client     *http.Client
	log        *logger.Logger
}

// NewOpenAIProvider 创建OpenAI提供商
func NewOpenAIProvider(apiKey, baseURL, model string, timeout, maxRetries int, log *logger.Logger) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &OpenAIProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      model,
		timeout:    time.Duration(timeout) * time.Second,
		maxRetries: maxRetries,
		client:     &http.Client{Timeout: time.Duration(timeout) * time.Second},
		log:        log,
	}
}

// Chat 发送聊天请求
func (p *OpenAIProvider) Chat(messages []session.Message, tools []Tool) (*Response, error) {
	reqBody := p.buildRequest(messages, tools, false)
	return p.doRequest(reqBody)
}

// ChatStream 发送流式聊天请求
func (p *OpenAIProvider) ChatStream(messages []session.Message, tools []Tool, callback func(chunk string)) (*Response, error) {
	reqBody := p.buildRequest(messages, tools, true)
	return p.doStreamRequest(reqBody, callback)
}

// GetModel 获取模型名称
func (p *OpenAIProvider) GetModel() string {
	return p.model
}

// buildRequest 构建请求体
func (p *OpenAIProvider) buildRequest(messages []session.Message, tools []Tool, stream bool) map[string]interface{} {
	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": p.convertMessages(messages),
		"stream":   stream,
	}

	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	return reqBody
}

// convertMessages 转换消息格式
func (p *OpenAIProvider) convertMessages(messages []session.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			m["tool_calls"] = msg.ToolCalls
		}
		result[i] = m
	}
	return result
}

// doRequest 发送请求
func (p *OpenAIProvider) doRequest(reqBody map[string]interface{}) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		resp, err := p.sendRequest(reqBody)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		p.log.Warn("llm request failed, retrying", "attempt", attempt+1, "error", err)
	}

	return nil, fmt.Errorf("llm request failed after %d retries: %w", p.maxRetries+1, lastErr)
}

// sendRequest 发送单次请求
func (p *OpenAIProvider) sendRequest(reqBody map[string]interface{}) (*Response, error) {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm api error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string             `json:"content"`
				ToolCalls []session.ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from llm")
	}

	return &Response{
		Content:   result.Choices[0].Message.Content,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// doStreamRequest 发送流式请求
func (p *OpenAIProvider) doStreamRequest(reqBody map[string]interface{}, callback func(chunk string)) (*Response, error) {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm api error: %s - %s", resp.Status, string(body))
	}

	var fullContent strings.Builder
	var toolCalls []session.ToolCall

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string             `json:"content"`
					ToolCalls []session.ToolCall `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			if content := chunk.Choices[0].Delta.Content; content != "" {
				fullContent.WriteString(content)
				if callback != nil {
					callback(content)
				}
			}
			if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, chunk.Choices[0].Delta.ToolCalls...)
			}
		}
	}

	return &Response{
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
	}, nil
}

// AnthropicProvider Anthropic Claude提供商
type AnthropicProvider struct {
	apiKey     string
	model      string
	timeout    time.Duration
	maxRetries int
	client     *http.Client
	log        *logger.Logger
}

// NewAnthropicProvider 创建Anthropic提供商
func NewAnthropicProvider(apiKey, model string, timeout, maxRetries int, log *logger.Logger) *AnthropicProvider {
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	return &AnthropicProvider{
		apiKey:     apiKey,
		model:      model,
		timeout:    time.Duration(timeout) * time.Second,
		maxRetries: maxRetries,
		client:     &http.Client{Timeout: time.Duration(timeout) * time.Second},
		log:        log,
	}
}

// Chat 发送聊天请求
func (p *AnthropicProvider) Chat(messages []session.Message, tools []Tool) (*Response, error) {
	reqBody := p.buildRequest(messages, tools, false)
	return p.doRequest(reqBody)
}

// ChatStream 发送流式聊天请求
func (p *AnthropicProvider) ChatStream(messages []session.Message, tools []Tool, callback func(chunk string)) (*Response, error) {
	reqBody := p.buildRequest(messages, tools, true)
	return p.doStreamRequest(reqBody, callback)
}

// GetModel 获取模型名称
func (p *AnthropicProvider) GetModel() string {
	return p.model
}

// buildRequest 构建请求体
func (p *AnthropicProvider) buildRequest(messages []session.Message, tools []Tool, stream bool) map[string]interface{} {
	systemMsg, userMsgs := p.separateMessages(messages)

	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": userMsgs,
		"stream":   stream,
		"max_tokens": 4096,
	}

	if systemMsg != "" {
		reqBody["system"] = systemMsg
	}

	if len(tools) > 0 {
		reqBody["tools"] = p.convertTools(tools)
	}

	return reqBody
}

// separateMessages 分离系统消息和用户消息
func (p *AnthropicProvider) separateMessages(messages []session.Message) (string, []map[string]interface{}) {
	var systemMsg string
	var userMsgs []map[string]interface{}

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		} else {
			userMsgs = append(userMsgs, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	return systemMsg, userMsgs
}

// convertTools 转换工具格式
func (p *AnthropicProvider) convertTools(tools []Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		}
	}
	return result
}

// doRequest 发送请求
func (p *AnthropicProvider) doRequest(reqBody map[string]interface{}) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		resp, err := p.sendRequest(reqBody)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		p.log.Warn("anthropic request failed, retrying", "attempt", attempt+1, "error", err)
	}

	return nil, fmt.Errorf("anthropic request failed after %d retries: %w", p.maxRetries+1, lastErr)
}

// sendRequest 发送单次请求
func (p *AnthropicProvider) sendRequest(reqBody map[string]interface{}) (*Response, error) {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic api error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Name  string `json:"name"`
			Input map[string]interface{} `json:"input"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var content string
	var toolCalls []session.ToolCall

	for _, c := range result.Content {
		if c.Type == "text" {
			content += c.Text
		} else if c.Type == "tool_use" {
			inputData, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, session.ToolCall{
				ID:   c.Name,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      c.Name,
					Arguments: string(inputData),
				},
			})
		}
	}

	return &Response{
		Content:   content,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

// doStreamRequest 发送流式请求
func (p *AnthropicProvider) doStreamRequest(reqBody map[string]interface{}, callback func(chunk string)) (*Response, error) {
	// 简化实现，非流式
	return p.doRequest(reqBody)
}

// OllamaProvider Ollama本地提供商
type OllamaProvider struct {
	baseURL    string
	model      string
	timeout    time.Duration
	maxRetries int
	client     *http.Client
	log        *logger.Logger
}

// NewOllamaProvider 创建Ollama提供商
func NewOllamaProvider(baseURL, model string, timeout, maxRetries int, log *logger.Logger) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      model,
		timeout:    time.Duration(timeout) * time.Second,
		maxRetries: maxRetries,
		client:     &http.Client{Timeout: time.Duration(timeout) * time.Second},
		log:        log,
	}
}

// Chat 发送聊天请求
func (p *OllamaProvider) Chat(messages []session.Message, tools []Tool) (*Response, error) {
	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": p.convertMessages(messages),
		"stream":   false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Response{
		Content: result.Message.Content,
	}, nil
}

// ChatStream 发送流式聊天请求
func (p *OllamaProvider) ChatStream(messages []session.Message, tools []Tool, callback func(chunk string)) (*Response, error) {
	// 简化实现，非流式
	return p.Chat(messages, tools)
}

// GetModel 获取模型名称
func (p *OllamaProvider) GetModel() string {
	return p.model
}

// convertMessages 转换消息格式
func (p *OllamaProvider) convertMessages(messages []session.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return result
}

// NewProvider 创建LLM提供商
func NewProvider(provider, apiKey, baseURL, model string, timeout, maxRetries int, log *logger.Logger) (Provider, error) {
	switch provider {
	case "openai":
		return NewOpenAIProvider(apiKey, baseURL, model, timeout, maxRetries, log), nil
	case "anthropic":
		return NewAnthropicProvider(apiKey, model, timeout, maxRetries, log), nil
	case "ollama":
		return NewOllamaProvider(baseURL, model, timeout, maxRetries, log), nil
	default:
		// 兼容OpenAI的API
		return NewOpenAIProvider(apiKey, baseURL, model, timeout, maxRetries, log), nil
	}
}
