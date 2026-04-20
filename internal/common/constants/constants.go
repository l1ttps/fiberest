package constants

import "time"

const (
	// AccessTokenDuration is the expiration time for access tokens
	AccessTokenDuration = 15 * time.Minute
	// RefreshTokenDuration is the expiration time for refresh tokens
	RefreshTokenDuration = 7 * 24 * time.Hour
)
