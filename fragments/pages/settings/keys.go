// Package fragments holds the content of the settings pages.
package fragments

import "github.com/Elagoht/collage/pkg/collage"

// NewRecoveryKey is the recovery key just issued, shown once.
var NewRecoveryKey = collage.NewKey[string]("settings-new-recovery-key")
