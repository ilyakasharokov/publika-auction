package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

const sessionCookie = "auction_session"

func signValue(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func SetSession(w http.ResponseWriter, secret string) {
	value := "authenticated"
	sig := signValue(secret, value)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value + "." + sig,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    sessionCookie,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}

func CheckSession(r *http.Request, secret string) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	idx := strings.LastIndex(cookie.Value, ".")
	if idx < 0 {
		return false
	}
	value := cookie.Value[:idx]
	sig := cookie.Value[idx+1:]
	expected := signValue(secret, value)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func RequireSession(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !CheckSession(r, secret) {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
