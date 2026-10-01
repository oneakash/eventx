package tests

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"eventx/models"

	"github.com/beego/beego/v2/server/web"
)

// withApprovedHosts sets the config value for the duration of the test.
func withApprovedHosts(t *testing.T, hosts string) {
	t.Helper()
	web.AppConfig.Set("APPROVED_TICKET_HOSTS", hosts)
	t.Cleanup(func() {
		web.AppConfig.Set("APPROVED_TICKET_HOSTS", "")
	})
}

func TestRedirect_ValidURLIssues302(t *testing.T) {
	withApprovedHosts(t, "www.ticketmaster.com,ticketmaster.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				TicketURL: "https://www.ticketmaster.com/event/" + id,
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 302 {
		t.Fatalf("expected 302, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc != "https://www.ticketmaster.com/event/ev1" {
		t.Fatalf("expected exact Ticketmaster URL, got %s", loc)
	}
}

func TestRedirect_EventNotFound(t *testing.T) {
	withApprovedHosts(t, "www.ticketmaster.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return nil, errors.New("not found")
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/missing")

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestRedirect_UnapprovedHost(t *testing.T) {
	withApprovedHosts(t, "www.ticketmaster.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				TicketURL: "https://evil.example.com/x",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 400 {
		t.Fatalf("expected 400 for unapproved host, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "" {
		t.Fatal("expected no Location header on rejection")
	}
}

func TestRedirect_HTTPURLRejected(t *testing.T) {
	withApprovedHosts(t, "www.ticketmaster.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				TicketURL: "http://www.ticketmaster.com/x",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 400 {
		t.Fatalf("expected 400 for http URL, got %d", rec.Code)
	}
}

func TestRedirect_EmptyTicketURL(t *testing.T) {
	withApprovedHosts(t, "www.ticketmaster.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{ID: id, TicketURL: ""}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 400 {
		t.Fatalf("expected 400 for empty URL, got %d", rec.Code)
	}
}

func TestRedirect_SubdomainAllowed(t *testing.T) {
	withApprovedHosts(t, "ticketmaster.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				TicketURL: "https://events.ticketmaster.com/event/ev1",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 302 {
		t.Fatalf("expected 302 for subdomain, got %d", rec.Code)
	}
}

func TestRedirect_HostNotInAllowList(t *testing.T) {
	// Allow-list has entries, but none match the URL's host.
	withApprovedHosts(t, "someone-else.example.com")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				TicketURL: "https://www.ticketmaster.com/event/ev1",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 400 {
		t.Fatalf("expected 400 when host not in allow-list, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "" {
		t.Fatal("expected no Location header on rejection")
	}
}

func TestRedirect_EmptyEventID_ReturnsNonRedirect(t *testing.T) {
	withApprovedHosts(t, "www.ticketmaster.com")
	withProvider(t, &ctrlFake{})

	// No event ID in the URL. Beego's router will not match /redirect/,
	// so the response will be a 404 (or similar), not a 302.
	rec := doRequest(t, http.MethodGet, "/redirect/")

	if rec.Code == 302 {
		t.Fatal("expected non-302 for missing event ID")
	}
}


// Sanity: the Location header must never contain a trailing/leading space.
func TestRedirect_NoWhitespaceInLocation(t *testing.T) {
	withApprovedHosts(t, "  www.ticketmaster.com  , ticketmaster.com ")
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				TicketURL: "https://www.ticketmaster.com/event/ev1",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/redirect/ev1")

	if rec.Code != 302 {
		t.Fatalf("expected 302 after trimming whitespace, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if strings.TrimSpace(loc) != loc {
		t.Fatalf("Location header has whitespace: %q", loc)
	}
}