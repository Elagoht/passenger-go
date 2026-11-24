package layouts

import "github.com/Elagoht/collage/pkg/collage"

// Auth is the centered-card shell of the setup, login and recovery screens.
// It has no guard: each page states its own.
func Auth() *collage.Fragment {
	return collage.NewFragment("auth-layout", "layouts/auth.html").
		WithTitle("Passenger").
		Build()
}
