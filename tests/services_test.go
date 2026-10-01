package tests

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"eventx/cache"
	"eventx/models"
	"eventx/services"
)

// ---------------------------------------------------------------------------
// Fake provider
// ---------------------------------------------------------------------------

type fakeProvider struct {
	failMusic  bool
	failSports bool
	delay      time.Duration
	calls      int64 // atomic counter
}

func (f *fakeProvider) Autocomplete(input, token string) ([]models.Suggestion, error) { return nil, nil }
func (f *fakeProvider) Resolve(id, token string) (*models.Place, error)               { return nil, nil }
func (f *fakeProvider) GetEvent(id string) (*models.Event, error)                     { return nil, nil }

func (f *fakeProvider) ListEvents(city, cc, category string) ([]models.Event, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if category == "Music" && f.failMusic {
		return nil, errors.New("music failed")
	}
	if category == "Sports" && f.failSports {
		return nil, errors.New("sports failed")
	}
	return []models.Event{{ID: category + "-1", Name: category + " Event"}}, nil
}

func (f *fakeProvider) callCount() int64 {
	return atomic.LoadInt64(&f.calls)
}

// ---------------------------------------------------------------------------
// GetListing — happy path
// ---------------------------------------------------------------------------

func TestGetListing_BothSucceed(t *testing.T) {
	cache.Init(time.Minute)
	svc := services.NewEventService(&fakeProvider{})

	music, sports := svc.GetListing("Toronto", "CA", "")

	if music.Err != "" || sports.Err != "" {
		t.Fatalf("unexpected errors: music=%q sports=%q", music.Err, sports.Err)
	}
	if music.Category != "Music" || sports.Category != "Sports" {
		t.Fatalf("wrong categories: %q %q", music.Category, sports.Category)
	}
	if len(music.Events) != 1 || len(sports.Events) != 1 {
		t.Fatalf("wrong event counts: music=%d sports=%d", len(music.Events), len(sports.Events))
	}
	if music.Cache != "MISS" || sports.Cache != "MISS" {
		t.Fatalf("expected MISS first call, got %s / %s", music.Cache, sports.Cache)
	}
}

func TestGetListing_CacheHitOnSecondCall(t *testing.T) {
	cache.Init(time.Minute)
	svc := services.NewEventService(&fakeProvider{})

	svc.GetListing("Toronto", "CA", "")
	music2, sports2 := svc.GetListing("Toronto", "CA", "")

	if music2.Cache != "HIT" {
		t.Fatalf("expected music HIT, got %s", music2.Cache)
	}
	if sports2.Cache != "HIT" {
		t.Fatalf("expected sports HIT, got %s", sports2.Cache)
	}
}

func TestGetListing_ProviderCalledOncePerCategory(t *testing.T) {
	cache.Init(time.Minute)
	fp := &fakeProvider{}
	svc := services.NewEventService(fp)

	svc.GetListing("Toronto", "CA", "") // 2 calls (Music + Sports)
	svc.GetListing("Toronto", "CA", "") // 0 calls (cache hits)

	if got := fp.callCount(); got != 2 {
		t.Fatalf("expected 2 provider calls, got %d", got)
	}
}

func TestGetListing_DifferentKeysAreCachedSeparately(t *testing.T) {
	cache.Init(time.Minute)
	fp := &fakeProvider{}
	svc := services.NewEventService(fp)

	svc.GetListing("Toronto", "CA", "") // MISS
	svc.GetListing("London", "GB", "")  // MISS (different key)
	svc.GetListing("Toronto", "CA", "") // HIT

	if got := fp.callCount(); got != 4 {
		t.Fatalf("expected 4 provider calls, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// GetListing — failure handling
// ---------------------------------------------------------------------------

func TestGetListing_MusicFailsSportsSucceeds(t *testing.T) {
	cache.Init(time.Minute)
	svc := services.NewEventService(&fakeProvider{failMusic: true})

	music, sports := svc.GetListing("Toronto", "CA", "")

	if music.Err == "" {
		t.Fatal("expected music error")
	}
	if music.Cache != "ERROR" {
		t.Fatalf("expected music cache=ERROR, got %s", music.Cache)
	}
	if sports.Err != "" {
		t.Fatalf("sports should succeed, got: %s", sports.Err)
	}
	if len(sports.Events) != 1 {
		t.Fatalf("sports should have events")
	}
}

func TestGetListing_SportsFailsMusicSucceeds(t *testing.T) {
	cache.Init(time.Minute)
	svc := services.NewEventService(&fakeProvider{failSports: true})

	music, sports := svc.GetListing("Toronto", "CA", "")

	if sports.Err == "" {
		t.Fatal("expected sports error")
	}
	if sports.Cache != "ERROR" {
		t.Fatalf("expected sports cache=ERROR, got %s", sports.Cache)
	}
	if music.Err != "" {
		t.Fatalf("music should succeed, got: %s", music.Err)
	}
}

func TestGetListing_BothFail(t *testing.T) {
	cache.Init(time.Minute)
	svc := services.NewEventService(&fakeProvider{failMusic: true, failSports: true})

	music, sports := svc.GetListing("Toronto", "CA", "")

	if music.Err == "" || sports.Err == "" {
		t.Fatal("expected both to fail")
	}
	if music.Cache != "ERROR" || sports.Cache != "ERROR" {
		t.Fatalf("expected ERROR cache for both")
	}
}

func TestGetListing_ErrorsAreNotCached(t *testing.T) {
	cache.Init(time.Minute)
	fp := &fakeProvider{failSports: true}
	svc := services.NewEventService(fp)

	svc.GetListing("Toronto", "CA", "")
	_, sports2 := svc.GetListing("Toronto", "CA", "")

	// Sports should retry the provider, not return a cached error.
	if sports2.Cache != "ERROR" {
		t.Fatalf("expected ERROR again, got %s", sports2.Cache)
	}

	// Call count reasoning:
	//   Round 1: Music succeeds (1 call, cached) + Sports fails (1 call, not cached) = 2
	//   Round 2: Music HIT (0 calls) + Sports retried (1 call) = 1
	//   Total: 3
	if got := fp.callCount(); got != 3 {   // ← was != 4
		t.Fatalf("expected 3 provider calls, got %d", got)
	}
}

func TestGetListing_MusicCachedEvenWhenSportsFails(t *testing.T) {
	// A failure in one section must not prevent caching the other.
	cache.Init(time.Minute)
	svc := services.NewEventService(&fakeProvider{failSports: true})

	svc.GetListing("Toronto", "CA", "")
	music2, _ := svc.GetListing("Toronto", "CA", "")

	if music2.Cache != "HIT" {
		t.Fatalf("music should be cached despite sports failure, got %s", music2.Cache)
	}
}

// ---------------------------------------------------------------------------
// GetListing — concurrency
// ---------------------------------------------------------------------------

func TestGetListing_RunsConcurrently(t *testing.T) {
	cache.Init(time.Minute)
	// Each provider call takes 100ms. If Music and Sports run
	// sequentially, total is 200ms. If concurrent, ~100ms.
	svc := services.NewEventService(&fakeProvider{delay: 100 * time.Millisecond})

	start := time.Now()
	svc.GetListing("Toronto", "CA", "")
	elapsed := time.Since(start)

	if elapsed > 180*time.Millisecond {
		t.Fatalf("expected concurrent execution, took %v", elapsed)
	}
}

// ---------------------------------------------------------------------------
// GetListing — nil cache safety
// ---------------------------------------------------------------------------

func TestGetListing_NilCacheDoesNotPanic(t *testing.T) {
	// Simulate a scenario where cache.Default is nil.
	// The service must still work, just without caching.
	oldDefault := cache.Default
	cache.Default = nil
	defer func() { cache.Default = oldDefault }()

	svc := services.NewEventService(&fakeProvider{})
	music, sports := svc.GetListing("Toronto", "CA", "")

	if music.Err != "" || sports.Err != "" {
		t.Fatalf("should work without cache: %q %q", music.Err, sports.Err)
	}
	// Without a cache, everything is MISS.
	if music.Cache != "MISS" || sports.Cache != "MISS" {
		t.Fatalf("expected MISS without cache, got %s/%s", music.Cache, sports.Cache)
	}
}

// ---------------------------------------------------------------------------
// ValidateTicketURL
// ---------------------------------------------------------------------------

func TestValidateTicketURL_HappyPath(t *testing.T) {
	approved := []string{"www.ticketmaster.com", "ticketmaster.com"}

	got, err := services.ValidateTicketURL("https://www.ticketmaster.com/event/123", approved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://www.ticketmaster.com/event/123" {
		t.Fatalf("URL was modified: %s", got)
	}
}

func TestValidateTicketURL_ReturnsOriginalString(t *testing.T) {
	// The function must return the input unchanged, not a re-serialized form.
	approved := []string{"www.ticketmaster.com"}
	input := "https://www.ticketmaster.com/event/abc?utm=xyz&ref=1"
	got, err := services.ValidateTicketURL(input, approved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Fatalf("expected original string\n got: %s\nwant: %s", got, input)
	}
}

func TestValidateTicketURL_RejectsHTTP(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	_, err := services.ValidateTicketURL("http://www.ticketmaster.com/x", approved)
	if err == nil {
		t.Fatal("expected http to be rejected")
	}
}

func TestValidateTicketURL_RejectsUnapprovedHost(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	_, err := services.ValidateTicketURL("https://evil.example.com/x", approved)
	if err == nil {
		t.Fatal("expected host rejection")
	}
}

func TestValidateTicketURL_RejectsPrefixCollision(t *testing.T) {
	// "evilticketmaster.com" must NOT match "ticketmaster.com".
	approved := []string{"ticketmaster.com"}
	_, err := services.ValidateTicketURL("https://evilticketmaster.com/x", approved)
	if err == nil {
		t.Fatal("expected prefix-collision host to be rejected")
	}
}

func TestValidateTicketURL_AcceptsSubdomain(t *testing.T) {
	// "events.ticketmaster.com" should be accepted when "ticketmaster.com" is approved.
	approved := []string{"ticketmaster.com"}
	got, err := services.ValidateTicketURL("https://events.ticketmaster.com/x", approved)
	if err != nil {
		t.Fatalf("expected subdomain to be accepted, got: %v", err)
	}
	if got == "" {
		t.Fatal("expected URL returned")
	}
}

func TestValidateTicketURL_CaseInsensitiveHost(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	if _, err := services.ValidateTicketURL("https://WWW.TICKETMASTER.COM/x", approved); err != nil {
		t.Fatalf("expected case-insensitive match, got %v", err)
	}
}

func TestValidateTicketURL_CaseInsensitiveApprovedList(t *testing.T) {
	approved := []string{"WWW.TICKETMASTER.COM"}
	if _, err := services.ValidateTicketURL("https://www.ticketmaster.com/x", approved); err != nil {
		t.Fatalf("expected case-insensitive approved list, got %v", err)
	}
}

func TestValidateTicketURL_EmptyString(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	_, err := services.ValidateTicketURL("", approved)
	if err == nil {
		t.Fatal("expected empty string to be rejected")
	}
}

func TestValidateTicketURL_ProtocolRelative(t *testing.T) {
	// "//host/path" has no scheme and must be rejected.
	approved := []string{"www.ticketmaster.com"}
	_, err := services.ValidateTicketURL("//www.ticketmaster.com/x", approved)
	if err == nil {
		t.Fatal("expected protocol-relative to be rejected")
	}
}

func TestValidateTicketURL_JavascriptScheme(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	_, err := services.ValidateTicketURL("javascript:alert(1)", approved)
	if err == nil {
		t.Fatal("expected javascript: to be rejected")
	}
}

func TestValidateTicketURL_FTPScheme(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	_, err := services.ValidateTicketURL("ftp://www.ticketmaster.com/x", approved)
	if err == nil {
		t.Fatal("expected ftp: to be rejected")
	}
}

func TestValidateTicketURL_EmptyApprovedList(t *testing.T) {
	_, err := services.ValidateTicketURL("https://www.ticketmaster.com/x", nil)
	if err == nil {
		t.Fatal("expected rejection with empty approved list")
	}
}

func TestValidateTicketURL_WhitespaceInApprovedList(t *testing.T) {
	// Config values may come from a comma-split with spaces.
	approved := []string{"  www.ticketmaster.com  ", ""}
	if _, err := services.ValidateTicketURL("https://www.ticketmaster.com/x", approved); err != nil {
		t.Fatalf("expected whitespace to be trimmed, got: %v", err)
	}
}

func TestValidateTicketURL_EmptyStringInApprovedList(t *testing.T) {
	// An empty entry must not accidentally match anything.
	approved := []string{""}
	if _, err := services.ValidateTicketURL("https://www.ticketmaster.com/x", approved); err == nil {
		t.Fatal("empty approved entry should not match")
	}
}

func TestValidateTicketURL_HostWithPort(t *testing.T) {
	// Hostname() strips the port; the host must still match.
	approved := []string{"www.ticketmaster.com"}
	if _, err := services.ValidateTicketURL("https://www.ticketmaster.com:8443/x", approved); err != nil {
		t.Fatalf("expected port to be stripped, got: %v", err)
	}
}

func TestValidateTicketURL_PathWithQuery(t *testing.T) {
	approved := []string{"www.ticketmaster.com"}
	got, err := services.ValidateTicketURL("https://www.ticketmaster.com/event/1?a=b&c=d", approved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://www.ticketmaster.com/event/1?a=b&c=d" {
		t.Fatalf("query lost: %s", got)
	}
}

func TestValidateTicketURL_MultipleApprovedHosts(t *testing.T) {
	approved := []string{
		"www.ticketmaster.com",
		"www.ticketmaster.ca",
		"www.ticketmaster.co.uk",
	}
	cases := []string{
		"https://www.ticketmaster.com/e/1",
		"https://www.ticketmaster.ca/e/2",
		"https://www.ticketmaster.co.uk/e/3",
	}
	for _, u := range cases {
		if _, err := services.ValidateTicketURL(u, approved); err != nil {
			t.Errorf("expected %s to be approved, got: %v", u, err)
		}
	}
}
// ValidateTicketURL — uncovered branches

func TestValidateTicketURL_MalformedURL(t *testing.T) {
	// "https://[::1" is an IPv6 host with a missing closing bracket.
	// url.Parse rejects it, exercising the `if err != nil` branch.
	_, err := services.ValidateTicketURL("https://[::1", []string{"ticketmaster.com"})
	if err == nil {
		t.Fatal("expected parse error for malformed URL")
	}
}

func TestValidateTicketURL_MissingHostname(t *testing.T) {
	// "https://" has a scheme but no host. url.Parse accepts it,
	// Hostname() returns "", triggering the missing-hostname branch.
	_, err := services.ValidateTicketURL("https://", []string{"ticketmaster.com"})
	if err == nil {
		t.Fatal("expected missing hostname error")
	}
}

func TestValidateTicketURL_MissingHostnameWithPath(t *testing.T) {
	// Another shape: scheme + path, still no host.
	_, err := services.ValidateTicketURL("https:///some/path", []string{"ticketmaster.com"})
	if err == nil {
		t.Fatal("expected missing hostname error")
	}
}

func TestValidateTicketURL_EmptyEntryThenValidEntry(t *testing.T) {
	// Ensures the loop's `continue` on empty entries is exercised and
	// does not prevent later valid entries from matching.
	approved := []string{"", "  ", "ticketmaster.com"}
	if _, err := services.ValidateTicketURL("https://ticketmaster.com/x", approved); err != nil {
		t.Fatalf("expected match after skipping empty entries, got: %v", err)
	}
}