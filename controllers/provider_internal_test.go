package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eventx/clients"
)


func TestGetProvider_UsesEnvVars(t *testing.T) {
	t.Setenv("GOOGLE_PLACES_API_KEY", "g-key-123")
	t.Setenv("TICKETMASTER_API_KEY", "t-key-456")

	p := getProvider()

	cp, ok := p.(*compositeProvider)
	if !ok {
		t.Fatalf("expected *compositeProvider, got %T", p)
	}
	if cp.google.APIKey != "g-key-123" {
		t.Fatalf("wrong google key: %s", cp.google.APIKey)
	}
	if cp.ticketmaster.APIKey != "t-key-456" {
		t.Fatalf("wrong ticketmaster key: %s", cp.ticketmaster.APIKey)
	}
}

func TestGetProvider_EmptyEnvVars(t *testing.T) {
	t.Setenv("GOOGLE_PLACES_API_KEY", "")
	t.Setenv("TICKETMASTER_API_KEY", "")

	p := getProvider()
	cp, ok := p.(*compositeProvider)
	if !ok {
		t.Fatalf("expected *compositeProvider, got %T", p)
	}
	if cp.google.APIKey != "" || cp.ticketmaster.APIKey != "" {
		t.Fatal("expected empty keys")
	}
}


func TestCompositeProvider_AutocompleteDelegatesToGoogle(t *testing.T) {
	var hit bool
	googleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{"suggestions":[{"placePrediction":{"placeId":"p1","text":{"text":"Toronto"}}}]}`))
	}))
	defer googleSrv.Close()

	cp := &compositeProvider{
		google:       &clients.GoogleClient{APIKey: "k", BaseURL: googleSrv.URL},
		ticketmaster: &clients.TicketmasterClient{APIKey: "k"},
	}

	out, err := cp.Autocomplete("Tor", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Fatal("google server was not hit")
	}
	if len(out) != 1 || out[0].PlaceID != "p1" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestCompositeProvider_ResolveDelegatesToGoogle(t *testing.T) {
	var hit bool
	googleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"Toronto","shortText":"Toronto","types":["locality"]},
				{"longText":"Canada","shortText":"CA","types":["country"]}
			]
		}`))
	}))
	defer googleSrv.Close()

	cp := &compositeProvider{
		google:       &clients.GoogleClient{APIKey: "k", BaseURL: googleSrv.URL},
		ticketmaster: &clients.TicketmasterClient{APIKey: "k"},
	}

	place, err := cp.Resolve("p1", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Fatal("google server was not hit")
	}
	if place.City != "Toronto" || place.CountryCode != "CA" {
		t.Fatalf("unexpected place: %+v", place)
	}
}

func TestCompositeProvider_ListEventsDelegatesToTicketmaster(t *testing.T) {
	var hit bool
	tmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{
			"_embedded":{
				"events":[
					{"id":"ev1","name":"Concert","url":"https://www.ticketmaster.com/ev1"}
				]
			}
		}`))
	}))
	defer tmSrv.Close()

	cp := &compositeProvider{
		google:       &clients.GoogleClient{APIKey: "k"},
		ticketmaster: &clients.TicketmasterClient{APIKey: "k", BaseURL: tmSrv.URL},
	}

	events, err := cp.ListEvents("Toronto", "CA", "Music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Fatal("ticketmaster server was not hit")
	}
	if len(events) != 1 || events[0].ID != "ev1" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestCompositeProvider_GetEventDelegatesToTicketmaster(t *testing.T) {
	var hit bool
	tmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{"id":"ev1","name":"Concert","url":"https://www.ticketmaster.com/ev1"}`))
	}))
	defer tmSrv.Close()

	cp := &compositeProvider{
		google:       &clients.GoogleClient{APIKey: "k"},
		ticketmaster: &clients.TicketmasterClient{APIKey: "k", BaseURL: tmSrv.URL},
	}

	ev, err := cp.GetEvent("ev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Fatal("ticketmaster server was not hit")
	}
	if ev.ID != "ev1" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestCompositeProvider_ErrorPropagation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	cp := &compositeProvider{
		google:       &clients.GoogleClient{APIKey: "k", BaseURL: srv.URL},
		ticketmaster: &clients.TicketmasterClient{APIKey: "k", BaseURL: srv.URL},
	}

	if _, err := cp.Autocomplete("Tor", "tok"); err == nil {
		t.Error("Autocomplete should propagate error")
	}
	if _, err := cp.Resolve("p1", "tok"); err == nil {
		t.Error("Resolve should propagate error")
	}
	if _, err := cp.ListEvents("T", "CA", "Music"); err == nil {
		t.Error("ListEvents should propagate error")
	}
	if _, err := cp.GetEvent("ev1"); err == nil {
		t.Error("GetEvent should propagate error")
	}
}

func TestCompositeProvider_SanityJSONShape(t *testing.T) {
	payload := map[string]any{
		"suggestions": []any{
			map[string]any{"placePrediction": map[string]any{
				"placeId": "p1",
				"text":    map[string]any{"text": "Toronto"},
			}},
		},
	}
	raw, _ := json.Marshal(payload)
	if len(raw) == 0 {
		t.Fatal("marshal produced no output")
	}
}