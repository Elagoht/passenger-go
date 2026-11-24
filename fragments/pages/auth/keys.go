// Package fragments holds the content of the setup, login and recovery pages.
package fragments

import "github.com/Elagoht/collage/pkg/collage"

// NewRecoveryKey is the recovery key /recover just issued, shown once.
var NewRecoveryKey = collage.NewKey[string]("auth-new-recovery-key")
