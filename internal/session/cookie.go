package session

import (
	"net/http"
	"time"
)

const (
	SessionDuration = 24 * time.Hour
)

const (
	CookieName     = "TOKEN"
	cookieMaxAge   = int(SessionDuration / time.Second)
	cookiePath     = "/"
	cookieSecure   = true
	cookieHTTPOnly = true
	cookieSameSite = http.SameSiteStrictMode
)

func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sessionID,
		Path:     cookiePath,
		MaxAge:   cookieMaxAge,
		Secure:   cookieSecure,
		HttpOnly: cookieHTTPOnly,
		SameSite: cookieSameSite,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     cookiePath,
		MaxAge:   -1, // Immediately expire
		Secure:   cookieSecure,
		HttpOnly: cookieHTTPOnly,
		SameSite: cookieSameSite,
	})
}
