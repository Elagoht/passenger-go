package utils

import (
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type expiryResponse struct {
	ExpiresAt   int64 `json:"expiresAt"`
	RemainingMs int64 `json:"remainingMs"`
}

// SessionExpiry answers the expiry banner's endpoints with when the session
// ends, as unix milliseconds and as the time left by the server's clock, or
// 401 with zeros when there is none. These URLs answer scripts, so they say so
// themselves instead of redirecting to login.
func SessionExpiry(d *web.Deps, rc *collage.RenderContext) (*collage.ActionResult, error) {
	s := web.SessionFrom(rc)
	if s == nil {
		return collage.JSON(http.StatusUnauthorized, expiryResponse{})
	}
	return collage.JSON(http.StatusOK, expiryResponse{
		ExpiresAt:   d.Sessions.ExpiresAt(s).UnixMilli(),
		RemainingMs: d.Sessions.Remaining(s).Milliseconds(),
	})
}
