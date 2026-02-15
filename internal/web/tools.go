package web

import (
	"encoding/json"
	"net/http"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/tools"
)

type ToolsHandler struct {
	config *config.Manager
	tools  *tools.Manager
}

func NewToolsHandler(cfg *config.Manager, toolMgr *tools.Manager) *ToolsHandler {
	return &ToolsHandler{
		config: cfg,
		tools:  toolMgr,
	}
}

func (h *ToolsHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools := h.tools.GetAll()
	cfg := h.config.Get()

	type ToolInfo struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Enabled     bool                   `json:"enabled"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	result := make([]ToolInfo, 0)
	for _, tool := range tools {
		info := ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
			Enabled:     true,
		}

		if enabled, ok := cfg.Tools.EnabledTools[tool.Name()]; ok {
			info.Enabled = enabled
		}

		result = append(result, info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ToolsHandler) ToggleTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
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
	cfg.Tools.EnabledTools[req.Name] = req.Enabled

	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"name":    req.Name,
		"enabled": req.Enabled,
	})
}

func (h *ToolsHandler) ListCustomAPIs(w http.ResponseWriter, r *http.Request) {
	cfg := h.config.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg.Tools.CustomAPIs)
}

func (h *ToolsHandler) AddCustomAPI(w http.ResponseWriter, r *http.Request) {
	var api config.CustomAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	cfg.Tools.CustomAPIs = append(cfg.Tools.CustomAPIs, api)
	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api)
}

func (h *ToolsHandler) UpdateCustomAPI(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	var api config.CustomAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	for i, a := range cfg.Tools.CustomAPIs {
		if a.Name == name {
			cfg.Tools.CustomAPIs[i] = api
			h.config.Update(cfg)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(api)
			return
		}
	}

	http.Error(w, "API not found", http.StatusNotFound)
}

func (h *ToolsHandler) DeleteCustomAPI(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	for i, a := range cfg.Tools.CustomAPIs {
		if a.Name == name {
			cfg.Tools.CustomAPIs = append(cfg.Tools.CustomAPIs[:i], cfg.Tools.CustomAPIs[i+1:]...)
			h.config.Update(cfg)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
	}

	http.Error(w, "API not found", http.StatusNotFound)
}

func (h *ToolsHandler) ListLLMPresets(w http.ResponseWriter, r *http.Request) {
	cfg := h.config.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg.LLMPresets)
}

func (h *ToolsHandler) GetLanguage(w http.ResponseWriter, r *http.Request) {
	cfg := h.config.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg.Language)
}

func (h *ToolsHandler) SetLanguage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Language string `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.config.Get()
	cfg.Language.Current = req.Language
	h.config.Update(cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
