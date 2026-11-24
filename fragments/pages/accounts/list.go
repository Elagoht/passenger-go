package fragments

import (
	"context"
	"strings"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/strength"
	"passenger-go/internal/web"
)

type cardView struct {
	ID         string
	Platform   string
	Identifier string
	Initials   string
	Strength   string // strength-weak | strength-fair | strength-strong
	Link       string
	Search     string // lower-cased text the live filter matches
}

type listView struct {
	Query   string
	Empty   bool // no accounts at all
	Cards   []cardView
	NoMatch bool
}

// List is the account list; ?q= keeps the accounts whose platform, identifier,
// URL or notes contain it, ignoring case.
func List(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("accounts-list", "pages/accounts/list.html").
		WithTitle("Accounts - Passenger").
		Required().
		WithData(collage.Load(func(ctx context.Context, rc *collage.RenderContext) (listView, error) {
			s, err := web.RequireSessionFrom(rc)
			if err != nil {
				return listView{}, err
			}
			defer s.DEK.Wipe()
			all, err := d.Store.ListAccounts(ctx, s.DEK)
			if err != nil {
				return listView{}, err
			}
			q := strings.TrimSpace(rc.Request.URL.Query().Get("q"))
			v := listView{Query: q, Empty: len(all) == 0, Cards: []cardView{}}
			for _, a := range all {
				text := searchText(a)
				if !strings.Contains(text, strings.ToLower(q)) {
					continue
				}
				v.Cards = append(v.Cards, cardView{
					ID:         a.ID,
					Platform:   a.Platform,
					Identifier: a.Identifier,
					Initials:   initials(a.Platform),
					Strength:   "strength-" + strength.Label(strength.Score(a.Passphrase)),
					Link:       externalLink(a.URL),
					Search:     clientText(a),
				})
			}
			v.NoMatch = !v.Empty && len(v.Cards) == 0
			return v, nil
		})).
		Build()
}

// clientText is what the in-page filter matches: notes can hold secrets, so
// they stay out of the HTML (the server-side ?q= still searches them).
func clientText(a store.Account) string {
	return strings.ToLower(strings.Join([]string{a.Platform, a.Identifier, a.URL}, "\n"))
}

func searchText(a store.Account) string {
	return strings.ToLower(strings.Join([]string{a.Platform, a.Identifier, a.URL, a.Notes}, "\n"))
}
