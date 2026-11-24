package transfer

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

func TestPlatformFromURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://accounts.google.com/x":  "Google",
		"https://github.com":             "Github",
		"http://localhost:8080/login":    "Unknown",
		"not a url":                      "Unknown",
		"":                               "Unknown",
		"https://www.example.co.uk:443/": "Example",
	} {
		if got := PlatformFromURL(in); got != want {
			t.Errorf("PlatformFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestImportFirefoxCRLFQuotedWithBOM(t *testing.T) {
	src := "\xEF\xBB\xBF" +
		`"url","username","password","httpRealm","formActionOrigin","guid","timeCreated","timeLastUsed","timePasswordChanged"` + "\r\n" +
		`"https://accounts.google.com","alice","p,w""1","","https://accounts.google.com","{g}","1","2","3"` + "\r\n" +
		`"https://github.com/login","bob","pw2","","","{h}","1","2","3"` + "\r\n"
	format, recs, err := Import(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []vault.Record{
		{Platform: "Google", Identifier: "alice", Passphrase: `p,w"1`, URL: "https://accounts.google.com"},
		{Platform: "Github", Identifier: "bob", Passphrase: "pw2", URL: "https://github.com/login"},
	}
	if format != "Firefox" || !reflect.DeepEqual(recs, want) {
		t.Fatalf("Import = %q, %+v; want Firefox, %+v", format, recs, want)
	}
}

func TestImportChromiumLF(t *testing.T) {
	src := "name,url,username,password,note\n" +
		"GitHub,https://github.com,alice,pw1,work\n" +
		"short,row\n" +
		"GitLab,https://gitlab.com,bob,pw2,\n"
	format, recs, err := Import(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []vault.Record{
		{Platform: "GitHub", Identifier: "alice", Passphrase: "pw1", URL: "https://github.com", Notes: "work"},
		{Platform: "GitLab", Identifier: "bob", Passphrase: "pw2", URL: "https://gitlab.com"},
	}
	if format != "Chromium" || !reflect.DeepEqual(recs, want) {
		t.Fatalf("Import = %q, %+v; want Chromium, %+v", format, recs, want)
	}
}

func TestImportPassengerAwkwardNotes(t *testing.T) {
	src := "platform,identifier,passphrase,url,notes\r\n" +
		"GitHub,alice,pw,github.com,\"one, two \"\"three\"\"\nsecond line\"\r\n"
	format, recs, err := Import(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []vault.Record{{Platform: "GitHub", Identifier: "alice", Passphrase: "pw", URL: "github.com",
		Notes: "one, two \"three\"\nsecond line"}}
	if format != "Passenger" || !reflect.DeepEqual(recs, want) {
		t.Fatalf("Import = %q, %+v; want Passenger, %+v", format, recs, want)
	}
}

func TestImportUnknownFormat(t *testing.T) {
	for _, src := range []string{"", "a,b,c\n1,2,3\n", "name,url,username,password\n"} {
		if _, _, err := Import(strings.NewReader(src)); !errors.Is(err, ErrUnknownFormat) {
			t.Errorf("Import(%q) err = %v, want ErrUnknownFormat", src, err)
		}
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	now := time.Now()
	accounts := []store.Account{
		{ID: "1", Record: vault.Record{Platform: "A", Identifier: "x", Passphrase: "p,1", URL: "a.com", Notes: "line1\nline2 \"q\""}, CreatedAt: now},
		{ID: "2", Record: vault.Record{Platform: "B", Identifier: "y\r\nz", Passphrase: "é€ü", URL: "", Notes: ""}},
	}
	var buf bytes.Buffer
	if err := Export(&buf, accounts); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "platform,identifier,passphrase,url,notes\n") {
		t.Fatalf("export header: %q", buf.String())
	}
	format, recs, err := Import(&buf)
	if err != nil {
		t.Fatal(err)
	}
	var want []vault.Record
	for _, a := range accounts {
		want = append(want, a.Record)
	}
	// encoding/csv normalises a bare \r\n inside a quoted field to \n.
	want[1].Identifier = "y\nz"
	if format != "Passenger" || !reflect.DeepEqual(recs, want) {
		t.Fatalf("round trip = %q, %+v; want %+v", format, recs, want)
	}
}
