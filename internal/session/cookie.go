package session

import "time"

const (
	// SessionDuration is the server-side session lifetime; shire's browser
	// cookie uses the same duration.
	SessionDuration = 24 * time.Hour

	// CookieName is the cookie shire forwards the session id under. Moria
	// only reads it; shire owns setting and clearing it in the browser.
	CookieName = "TOKEN"
)
