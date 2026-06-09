package handlers

import (
	"net/http"

	"publika-auction/internal/admin/middleware"
)

type AuthHandler struct {
	adminUser string
	adminPass string
	secret    string
}

func NewAuthHandler(user, pass, secret string) *AuthHandler {
	return &AuthHandler{adminUser: user, adminPass: pass, secret: secret}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if middleware.CheckSession(r, h.secret) {
		http.Redirect(w, r, "/admin/auctions", http.StatusFound)
		return
	}
	render(w, r, "login.html", map[string]interface{}{"Error": ""})
}

func (h *AuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.Form.Get("user") == h.adminUser && r.Form.Get("password") == h.adminPass {
		middleware.SetSession(w, h.secret)
		http.Redirect(w, r, "/admin/auctions", http.StatusFound)
		return
	}
	render(w, r, "login.html", map[string]interface{}{"Error": "Invalid username or password"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}
