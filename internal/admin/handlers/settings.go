package handlers

import (
	"net/http"

	"publika-auction/internal/tg"
)

type SettingsHandler struct {
	botManager *tg.Manager
}

func NewSettingsHandler(m *tg.Manager) *SettingsHandler {
	return &SettingsHandler{botManager: m}
}

func (h *SettingsHandler) Page(w http.ResponseWriter, r *http.Request) {
	render(w, r, "settings.html", map[string]interface{}{
		"Status": h.botManager.GetStatus(),
	})
}

func (h *SettingsHandler) Connect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	token := r.Form.Get("token")
	endpoint := r.Form.Get("endpoint")
	if token == "" {
		render(w, r, "settings.html", map[string]interface{}{
			"Status": h.botManager.GetStatus(),
			"Error":  "Token cannot be empty",
		})
		return
	}
	if err := h.botManager.Connect(token, endpoint); err != nil {
		render(w, r, "settings.html", map[string]interface{}{
			"Status": h.botManager.GetStatus(),
			"Error":  "Connection failed: " + err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/admin/settings", http.StatusFound)
}

func (h *SettingsHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	h.botManager.Disconnect()
	http.Redirect(w, r, "/admin/settings", http.StatusFound)
}
