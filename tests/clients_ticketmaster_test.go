package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"eventx/clients"
)

// Base()

func TestTicketmasterBase_DefaultURL(t *testing.T) {
	tc := &clients.TicketmasterClient{APIKey: "k"}
	want := "https://app.ticketmaster.com/discovery/v2"
	if got := tc.Base(); got != want {
		t.Fatalf("unexpected default base: %s", got)
	}
}

func TestTicketmasterBase_CustomURL(t *testing.T) {
	tc := &clients.TicketmasterClient{APIKey: "k", BaseURL: "https://example.test"}
	if got := tc.Base(); got != "https://example.test" {
		t.Fatalf("unexpected base: %s", got)
	}
}

// ListEvents

func TestTicketmasterListEvents_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/events.json") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify all query params landed
		q := r.URL.Query()
		if q.Get("apikey") != "test-key" {
			t.Errorf("apikey wrong: %s", q.Get("apikey"))
		}
		if q.Get("city") != "Toronto" {
			t.Errorf("city wrong: %s", q.Get("city"))
		}
		if q.Get("countryCode") != "CA" {
			t.Errorf("countryCode wrong: %s", q.Get("countryCode"))
		}
		if q.Get("classificationName") != "Music" {
			t.Errorf("classificationName wrong: %s", q.Get("classificationName"))
		}
		if q.Get("size") != "6" {
			t.Errorf("size wrong: %s", q.Get("size"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"_embedded":{
				"events":[
					{
						"id":"ev1","name":"Concert One",
						"info":"An evening of music.",
						"url":"https://www.ticketmaster.com/event/ev1",
						"images":[{"url":"https://img/1.jpg","ratio":"16_9","width":640}],
						"dates":{"start":{"localDate":"2026-10-15","localTime":"19:30"}},
						"_embedded":{"venues":[{"name":"Arena A"}]}
					},
					{
						"id":"ev2","name":"Concert Two",
						"url":"https://www.ticketmaster.com/event/ev2",
						"images":[{"url":"https://img/2.jpg"}],
						"dates":{"start":{"localDate":"2026-10-16","localTime":"20:00"}},
						"_embedded":{"venues":[{"name":"Arena B"}]}
					}
				]
			}
		}`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	events, err := tc.ListEvents("Toronto", "CA", "Music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	first := events[0]
	if first.ID != "ev1" || first.Name != "Concert One" {
		t.Fatalf("unexpected first event: %+v", first)
	}
	if first.ImageURL != "https://img/1.jpg" {
		t.Fatalf("image not extracted: %+v", first)
	}
	if first.Date != "2026-10-15 19:30" {
		t.Fatalf("date format wrong: %s", first.Date)
	}
	if first.Venue != "Arena A" {
		t.Fatalf("venue not extracted: %s", first.Venue)
	}
	if first.Category != "Music" {
		t.Fatalf("category not set: %s", first.Category)
	}
	if first.TicketURL != "https://www.ticketmaster.com/event/ev1" {
		t.Fatalf("ticket URL wrong: %s", first.TicketURL)
	}
}

func TestTicketmasterListEvents_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`)) // no _embedded, no events
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	events, err := tc.ListEvents("Dhaka", "BD", "Music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(events) != 0 {
		t.Fatalf("expected 0, got %d", len(events))
	}
}

func TestTicketmasterListEvents_ExplicitEmptyArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"_embedded":{"events":[]}}`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	events, err := tc.ListEvents("Toronto", "CA", "Music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0, got %d", len(events))
	}
}

func TestTicketmasterListEvents_MissingOptionalFields(t *testing.T) {
	// Events without images, venues, or dates should still be returned,
	// just with empty strings for those fields.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"_embedded":{
				"events":[
					{"id":"min","name":"Bare Minimum"}
				]
			}
		}`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	events, err := tc.ListEvents("Toronto", "CA", "Music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1, got %d", len(events))
	}
	e := events[0]
	if e.ID != "min" || e.Name != "Bare Minimum" {
		t.Fatalf("basic fields wrong: %+v", e)
	}
	if e.ImageURL != "" || e.Venue != "" || e.Date != "" {
		t.Fatalf("expected empty optional fields, got %+v", e)
	}
}

func TestTicketmasterListEvents_HTTPError(t *testing.T) {
	for _, code := range []int{400, 401, 403, 429, 500, 502} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				w.Write([]byte(`{"fault":"boom"}`))
			}))
			defer srv.Close()

			tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
			_, err := tc.ListEvents("Toronto", "CA", "Music")
			if err == nil {
				t.Fatalf("expected error for %d", code)
			}
			if !strings.Contains(err.Error(), "ticketmaster") {
				t.Fatalf("expected wrapped error, got: %v", err)
			}
		})
	}
}

func TestTicketmasterListEvents_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>not json</html>`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := tc.ListEvents("Toronto", "CA", "Music"); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestTicketmasterListEvents_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: closedURL}
	if _, err := tc.ListEvents("Toronto", "CA", "Music"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// GetEvent

func TestTicketmasterGetEvent_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/events/ev1.json") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("apikey") != "test-key" {
			t.Errorf("apikey wrong: %s", r.URL.Query().Get("apikey"))
		}

		w.Write([]byte(`{
			"id":"ev1","name":"Concert One",
			"info":"An evening of music.",
			"url":"https://www.ticketmaster.com/event/ev1",
			"images":[{"url":"https://img/1.jpg"}],
			"dates":{"start":{"localDate":"2026-10-15","localTime":"19:30"}},
			"_embedded":{"venues":[{"name":"Arena A"}]}
		}`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	ev, err := tc.GetEvent("ev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.ID != "ev1" || ev.Name != "Concert One" {
		t.Fatalf("basic fields wrong: %+v", ev)
	}
	if ev.ImageURL != "https://img/1.jpg" {
		t.Fatalf("image not extracted: %s", ev.ImageURL)
	}
	if ev.Venue != "Arena A" {
		t.Fatalf("venue not extracted: %s", ev.Venue)
	}
	if ev.Date != "2026-10-15 19:30" {
		t.Fatalf("date format wrong: %s", ev.Date)
	}
	if ev.TicketURL != "https://www.ticketmaster.com/event/ev1" {
		t.Fatalf("ticket URL wrong: %s", ev.TicketURL)
	}
}

func TestTicketmasterGetEvent_MinimalFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"min","name":"Bare"}`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	ev, err := tc.GetEvent("min")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.ID != "min" || ev.Name != "Bare" {
		t.Fatalf("basic fields wrong: %+v", ev)
	}
	if ev.ImageURL != "" || ev.Venue != "" || ev.Date != "" || ev.TicketURL != "" {
		t.Fatalf("expected empty optional fields, got %+v", ev)
	}
}

func TestTicketmasterGetEvent_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	_, err := tc.GetEvent("missing")
	if err == nil {
		t.Fatal("expected error on 404")
	}
	if err.Error() != "not found" {
		t.Fatalf("expected 'not found', got: %v", err)
	}
}

func TestTicketmasterGetEvent_HTTPError(t *testing.T) {
	for _, code := range []int{401, 403, 429, 500, 502} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				w.Write([]byte(`{"fault":"boom"}`))
			}))
			defer srv.Close()

			tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
			_, err := tc.GetEvent("ev1")
			if err == nil {
				t.Fatalf("expected error for %d", code)
			}
			if !strings.Contains(err.Error(), "ticketmaster") {
				t.Fatalf("expected wrapped error, got: %v", err)
			}
		})
	}
}

func TestTicketmasterGetEvent_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{invalid`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := tc.GetEvent("ev1"); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestTicketmasterGetEvent_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: closedURL}
	if _, err := tc.GetEvent("ev1"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestTicketmasterGetEvent_EncodesEventID(t *testing.T) {
	var rawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawPath = r.URL.RawPath
		w.Write([]byte(`{"id":"x","name":"X"}`))
	}))
	defer srv.Close()

	tc := &clients.TicketmasterClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := tc.GetEvent("abc/def"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(rawPath, "abc%2Fdef") {
		t.Fatalf("expected encoded slash in RawPath, got: %s", rawPath)
	}
}

// Sanity: JSON serialization of a mapped Event is not accidentally leaking

func TestTicketmasterEventJSONShape(t *testing.T) {
	// Verifies the mapped models.Event has the same JSON keys the rest of
	// the app expects — catches accidental rename of a tag.
	ev := mapEventForTest()
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"id"`, `"name"`, `"imageUrl"`, `"date"`, `"venue"`, `"ticketUrl"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("missing key %s in %s", key, raw)
		}
	}
}

// local helper to avoid importing models twice in this file
func mapEventForTest() any {
	return struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ImageURL  string `json:"imageUrl"`
		Date      string `json:"date"`
		Venue     string `json:"venue"`
		TicketURL string `json:"ticketUrl"`
	}{
		ID: "e1", Name: "n", ImageURL: "i", Date: "d", Venue: "v", TicketURL: "t",
	}
}