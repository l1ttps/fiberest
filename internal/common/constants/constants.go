package constants

import "time"

const (
	// SessionDuration is the expiration time for user sessions (7 days)
	SessionDuration = 7 * 24 * time.Hour
)
