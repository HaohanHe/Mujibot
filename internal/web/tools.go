package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/logger"
	"github.com/HaohanHe/mujibot/internal/tools"
)

// ToolsHandler 工具处理器
type ToolsHandler struct {
	config     *config.Manager
	toolMgr    *tools.Manager
	log        *logger.Logger
}

// NewToolsHandler 创建工具处理器
func NewToolsHandler(cfg *config.Manager, toolMgr *tools.Manager, log *logger.Logger) *ToolsHandler {
	return &ToolsHandler{
		config:     cfg,
		toolMgr:    toolMgr,
		log:        log,
	}
}

// ListTools 获取所有工具列表
func (h *ToolsHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.config.Get()

	// 获取内置工具列表
	builtinTools := h.toolMgr.GetAllTools()
	
	// 构建工具信息
	toolsInfo := make([]map[string]interface{}, 0, len(builtinTools))
	for name, tool := range builtinTools {
		enabled := cfg.Tools.EnabledTools[name]
		if !enabled {
			enabled = false
		}
		
		toolsInfo = append(toolsInfo, map[string]interface{}{
			"name":     name,
			"description": tool.Description,
			"enabled":  enabled,
			"type":     "builtin",
		})
	}

	// 添加自定义API
	for _, api := range cfg.Tools.CustomAPIs {
		toolsInfo = append(toolsInfo, map[string]interface{}{
			"name":        api.Name,
			"description": api.Description,
			"enabled":     api.Enabled,
			"type":        "custom",
			"url":         api.URL,
			"method":      api.Method,
			"timeout":     api.Timeout,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": toolsInfo,
	})
}

// ToggleTool 切换工具开关
func (h *ToolsHandler) ToggleTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Tool    string `json:"tool"`
		Enabled bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	if cfg.Tools.EnabledTools == nil {
		cfg.Tools.EnabledTools = make(map[string]bool)
	}
	cfg.Tools.EnabledTools[req.Tool] = req.Enabled

	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Tool %s %s", req.Tool, boolToStatus(req.Enabled)),
	})
}

// ListCustomAPIs 获取自定义API列表
func (h *ToolsHandler) ListCustomAPIs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.config.Get()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg.Tools.CustomAPIs)
}

// AddCustomAPI 添加自定义API
func (h *ToolsHandler) AddCustomAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var api config.CustomAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if api.Name == "" || api.URL == "" || api.Method == "" {
		http.Error(w, "Name, URL and Method are required", http.StatusBadRequest)
		return
	}

	// 设置默认值
	if api.Timeout == 0 {
		api.Timeout = 10
	}

	cfg := h.config.Get()
	cfg.Tools.CustomAPIs = append(cfg.Tools.CustomAPIs, api)
	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Custom API added successfully",
	})
}

// UpdateCustomAPI 更新自定义API
func (h *ToolsHandler) UpdateCustomAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL中获取API ID
	path := strings.TrimPrefix(r.URL.Path, "/api/tools/custom/")
	apiName := strings.TrimSpace(path)

	if apiName == "" {
		http.Error(w, "API name is required", http.StatusBadRequest)
		return
	}

	var api config.CustomAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	updated := false

	for i, existingAPI := range cfg.Tools.CustomAPIs {
		if existingAPI.Name == apiName {
			cfg.Tools.CustomAPIs[i] = api
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Custom API not found", http.StatusNotFound)
		return
	}

	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Custom API updated successfully",
	})
}

// DeleteCustomAPI 删除自定义API
func (h *ToolsHandler) DeleteCustomAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL中获取API ID
	path := strings.TrimPrefix(r.URL.Path, "/api/tools/custom/")
	apiName := strings.TrimSpace(path)

	if apiName == "" {
		http.Error(w, "API name is required", http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	updated := false
	newAPIs := make([]config.CustomAPIConfig, 0)

	for _, existingAPI := range cfg.Tools.CustomAPIs {
		if existingAPI.Name != apiName {
			newAPIs = append(newAPIs, existingAPI)
		} else {
			updated = true
		}
	}

	if !updated {
		http.Error(w, "Custom API not found", http.StatusNotFound)
		return
	}

	cfg.Tools.CustomAPIs = newAPIs
	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Custom API deleted successfully",
	})
}

// TestCustomAPI 测试自定义API
func (h *ToolsHandler) TestCustomAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL中获取API ID
	path := strings.TrimPrefix(r.URL.Path, "/api/tools/custom/")
	pathParts := strings.Split(path, "/test")
	if len(pathParts) != 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	apiName := strings.TrimSpace(pathParts[0])
	if apiName == "" {
		http.Error(w, "API name is required", http.StatusBadRequest)
		return
	}

	// 解析测试参数
	var testParams map[string]string
	if err := json.NewDecoder(r.Body).Decode(&testParams); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	var targetAPI *config.CustomAPIConfig

	for i := range cfg.Tools.CustomAPIs {
		if cfg.Tools.CustomAPIs[i].Name == apiName {
			targetAPI = &cfg.Tools.CustomAPIs[i]
			break
		}
	}

	if targetAPI == nil {
		http.Error(w, "Custom API not found", http.StatusNotFound)
		return
	}

	// 测试API调用
	response, err := h.toolMgr.TestCustomAPI(*targetAPI, testParams)
	if err != nil {
		http.Error(w, fmt.Sprintf("API test failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"message":  "API test successful",
		"response": response,
	})
}

// ListLLMPresets 获取LLM预设列表
func (h *ToolsHandler) ListLLMPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.config.Get()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg.LLMPresets)
}

// GetLanguage 获取语言设置
func (h *ToolsHandler) GetLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.config.Get()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"default":   cfg.Language.Default,
		"current":   cfg.Language.Current,
		"supported": cfg.Language.Supported,
	})
}

// SetLanguage 设置语言
func (h *ToolsHandler) SetLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Language string `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	
	// 验证语言是否支持
	supported := false
	for _, lang := range cfg.Language.Supported {
		if lang == req.Language {
			supported = true
			break
		}
	}

	if !supported {
		http.Error(w, "Unsupported language", http.StatusBadRequest)
		return
	}

	cfg.Language.Current = req.Language
	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Language updated successfully",
	})
}

// boolToStatus 将布尔值转换为状态字符串
func boolToStatus(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}
