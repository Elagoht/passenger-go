package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/settings"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/settings"
	"passenger-go/internal/web"
)

// Recovery is /settings/recovery, the recovery key regeneration.
func Recovery(d *web.Deps) *collage.Page {
	return collage.NewPage("settings-recovery").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Recovery()).
		WithPath("en", "/settings/recovery").
		WithActionFor(actions.Recovery(d)).
		Dynamic().
		Build()
}
