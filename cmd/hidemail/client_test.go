package main

import (
	"regexp"
	"testing"
)

func TestParseValidate(t *testing.T) {
	body := []byte(`{
		"dsInfo": {"dsid": 123456789},
		"webservices": {"premiummailsettings": {"url": "https://p42-maildomainws.icloud.com:443", "status": "active"}}
	}`)
	dsid, base, err := parseValidate(body)
	if err != nil {
		t.Fatal(err)
	}
	if dsid != "123456789" {
		t.Fatalf("dsid = %q", dsid)
	}
	if base != "https://p42-maildomainws.icloud.com:443" {
		t.Fatalf("base = %q", base)
	}

	// Missing service root falls back to the hardcoded shard.
	_, base, err = parseValidate([]byte(`{"dsInfo":{"dsid":1},"webservices":{}}`))
	if err != nil || base != fallbackURL {
		t.Fatalf("fallback base = %q err = %v", base, err)
	}
}

func TestApiError(t *testing.T) {
	if err := apiError([]byte(`{"success":true,"result":{}}`)); err != nil {
		t.Fatalf("success should be nil, got %v", err)
	}
	err := apiError([]byte(`{"success":false,"reason":"nope"}`))
	if err == nil || err.Error() != "iCloud error: nope" {
		t.Fatalf("got %v", err)
	}
}

func TestCookieRoundtrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const want = "X-APPLE-WEBAUTH-TOKEN=abc; foo=bar"
	if err := saveCookies("  " + want + "\n"); err != nil {
		t.Fatal(err)
	}
	got, err := loadCookies()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestUUIDV4(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if u := uuidV4(); !re.MatchString(u) {
		t.Fatalf("not a v4 uuid: %s", u)
	}
}

func TestLooksLikeCookies(t *testing.T) {
	if looksLikeCookies("too short") {
		t.Fatal("should reject junk")
	}
	if !looksLikeCookies("X-APPLE-WEBAUTH-TOKEN=v=1:t=xyz; X-APPLE-WEBAUTH-USER=v=1:s=0:d=1") {
		t.Fatal("should accept a real-looking header")
	}
}
