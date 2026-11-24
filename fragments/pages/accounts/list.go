package fragments

import (
	"context"
	"strings"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/strength"
	"passenger-go/internal/web"
	"passenger-go/utils"
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
	return collage.NewInlineFragment("accounts-list", listBlock).
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
					Initials:   utils.Initials(a.Platform),
					Strength:   "strength-" + strength.Label(strength.Score(a.Passphrase)),
					Link:       utils.ExternalLink(a.URL),
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

const listBlock collage.InlineHTML = `{{if .Empty}}
<section class="empty">
  <h1>Your vault is empty</h1>
  <p>Bring your passwords over from a browser export, or add your first account by hand.</p>
  <div class="actions">
    <a class="button button-primary" href="{{pageURL "account-new"}}">Add an account</a>
    <a class="button" href="/import">Import a CSV</a>
  </div>
  <p class="muted">No accounts yet.</p>
</section>
{{else}}
<form class="finder" method="get" action="{{pageURL "accounts"}}" role="search">
  <label class="visually-hidden" for="finder-input">Search accounts</label>
  <input id="finder-input" type="search" name="q" value="{{.Query}}" placeholder="Search by site, username or address" data-filter="accounts-grid" data-q="{{.Query}}" autocomplete="off" spellcheck="false" data-autofocus>
  <kbd class="finder-hint" aria-hidden="true">/</kbd>
</form>

<p class="keys-hint" aria-hidden="true" hidden data-keys-hint>
  <kbd>↑</kbd><kbd>↓</kbd> choose, <kbd>Enter</kbd> copies the password, <kbd>Shift</kbd>+<kbd>Enter</kbd> the username
</p>

{{if .NoMatch}}<p class="empty-note">No account matches “{{.Query}}”. <a href="{{pageURL "accounts"}}">Show all</a></p>{{end}}

<ol id="accounts-grid" class="tickets" aria-label="Accounts">
  {{range .Cards}}
  <li class="ticket" data-card data-search="{{.Search}}" tabindex="-1">
    <a class="ticket-main" href="{{pageURL "account" "id" .ID}}">
      <span class="initials" aria-hidden="true">{{.Initials}}</span>
      <span class="ticket-text">
        <span class="platform">{{.Platform}}</span>
        <span class="identifier">{{.Identifier}}</span>
      </span>
      <span class="meter {{.Strength}}" title="Password strength"><i></i><i></i><i></i></span>
    </a>
    <div class="ticket-stub">
      <button type="button" class="button button-primary" data-copy-passphrase data-url="{{actionURL "account-passphrase" "id" .ID}}" data-id="{{.ID}}" data-name="{{.Platform}}" hidden>Copy password</button>
      <button type="button" class="button" data-copy-text="{{.Identifier}}" data-id="{{.ID}}" data-name="{{.Platform}}" hidden>Copy username</button>
      {{with .Link}}<a class="button button-quiet" href="{{.}}" target="_blank" rel="noopener noreferrer">Open site</a>{{end}}
    </div>
  </li>
  {{end}}
</ol>
{{end}}`
