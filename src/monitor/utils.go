package monitor

import (
	"encoding/json"
	"net/http"
)

// ==================== 工具函数 ====================

func respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func respondSuccess(w http.ResponseWriter, message string) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": message})
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}
