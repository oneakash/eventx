package tests

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"eventx/cache"
	"eventx/models"
)

func TestEventsList_MissingCityShowsMessage(t *testing.T) {
	withProvider(t, &ctrlFake{})

	rec := doRequest(t, http.MethodGet, "/events")

	// Renders the listing template with an error message, HTTP 200.
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Please select a city") {
		t.Fatalf("expected 'Please select a city' message, got: %s", rec.Body.String())
	}
}

func TestEventsList_MissingCountryCodeShowsMessage(t *testing.T) {
	withProvider(t, &ctrlFake{})

	rec := doRequest(t, http.MethodGet, "/events?city=Toronto")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Please select a city") {
		t.Fatalf("expected error message, got: %s", rec.Body.String())
	}
}

func TestEventsList_BothCategoriesSucceed(t *testing.T) {
	cache.Init(time.Minute)
	withProvider(t, &ctrlFake{
		listEventsFn: func(city, cc, cat string) ([]models.Event, error) {
			return []models.Event{{ID: cat + "-1", Name: cat + " Event"}}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/events?city=Toronto&countryCode=CA")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Toronto") {
		t.Fatalf("expected city in output")
	}
	// Cache headers
	if rec.Header().Get("X-Music-Cache") == "" {
		t.Fatalf("missing X-Music-Cache header")
	}
	if rec.Header().Get("X-Sports-Cache") == "" {
		t.Fatalf("missing X-Sports-Cache header")
	}
}

func TestEventsList_CacheMissThenHit(t *testing.T) {
	cache.Init(time.Minute)
	withProvider(t, &ctrlFake{
		listEventsFn: func(city, cc, cat string) ([]models.Event, error) {
			return []models.Event{{ID: cat + "-1"}}, nil
		},
	})

	rec1 := doRequest(t, http.MethodGet, "/events?city=Toronto&countryCode=CA")
	if got := rec1.Header().Get("X-Music-Cache"); got != "MISS" {
		t.Fatalf("expected MISS first, got %s", got)
	}

	rec2 := doRequest(t, http.MethodGet, "/events?city=Toronto&countryCode=CA")
	if got := rec2.Header().Get("X-Music-Cache"); got != "HIT" {
		t.Fatalf("expected HIT second, got %s", got)
	}
}

func TestEventsList_OneCategoryFails(t *testing.T) {
	cache.Init(time.Minute)
	withProvider(t, &ctrlFake{
		listEventsFn: func(city, cc, cat string) ([]models.Event, error) {
			if cat == "Sports" {
				return nil, errors.New("sports down")
			}
			return []models.Event{{ID: "music-1", Name: "Music"}}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/events?city=Toronto&countryCode=CA")

	if rec.Code != 200 {
		t.Fatalf("expected 200 with partial failure, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Sports-Cache"); got != "ERROR" {
		t.Fatalf("expected sports ERROR, got %s", got)
	}
	if got := rec.Header().Get("X-Music-Cache"); got != "MISS" {
		t.Fatalf("expected music MISS, got %s", got)
	}
}

func TestEventsList_BothCategoriesFail(t *testing.T) {
	cache.Init(time.Minute)
	withProvider(t, &ctrlFake{
		listEventsFn: func(city, cc, cat string) ([]models.Event, error) {
			return nil, errors.New("both down")
		},
	})

	rec := doRequest(t, http.MethodGet, "/events?city=Toronto&countryCode=CA")

	if rec.Code != 502 {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unavailable") {
		t.Fatalf("expected error message, got: %s", rec.Body.String())
	}
}

func TestEventsList_EmptyResult(t *testing.T) {
	cache.Init(time.Minute)
	withProvider(t, &ctrlFake{
		listEventsFn: func(city, cc, cat string) ([]models.Event, error) {
			return []models.Event{}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/events?city=Dhaka&countryCode=BD")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No Music events") {
		t.Fatalf("expected empty Music message, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "No Sports events") {
		t.Fatalf("expected empty Sports message, got: %s", rec.Body.String())
	}
}

func TestEventsDetails_Success(t *testing.T) {
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				Name:      "Concert One",
				ImageURL:  "https://img/1.jpg",
				Date:      "2026-10-15 19:30",
				Venue:     "Arena A",
				TicketURL: "https://www.ticketmaster.com/event/abc",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/events/abc")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Concert One") {
		t.Fatalf("expected event name in body")
	}
	if !strings.Contains(body, "/redirect/abc") {
		t.Fatalf("expected redirect link with event ID")
	}
	// The raw ticket URL must NOT be in the HTML
	if strings.Contains(body, "https://www.ticketmaster.com/event/abc") {
		t.Fatalf("raw ticket URL leaked into HTML")
	}
}

func TestEventsDetails_EventNotFound(t *testing.T) {
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return nil, errors.New("not found")
		},
	})

	rec := doRequest(t, http.MethodGet, "/events/missing")

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not found") && !strings.Contains(rec.Body.String(), "Event not found") {
		t.Fatalf("expected error message, got: %s", rec.Body.String())
	}
}

func TestEventsDetails_NoTicketURL(t *testing.T) {
	withProvider(t, &ctrlFake{
		getEventFn: func(id string) (*models.Event, error) {
			return &models.Event{
				ID:        id,
				Name:      "No Tickets Event",
				TicketURL: "", // empty
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/events/no-tickets")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unavailable") {
		t.Fatalf("expected 'unavailable' message for missing ticket URL")
	}
}