package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"eventx/models"
)

// Suggestion

func TestSuggestionJSON(t *testing.T) {
	s := models.Suggestion{PlaceID: "mock-toronto", Text: "Toronto, Canada"}

	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	want := `{"placeId":"mock-toronto","text":"Toronto, Canada"}`
	if string(raw) != want {
		t.Fatalf("wrong JSON\n got: %s\nwant: %s", raw, want)
	}

	var back models.Suggestion
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back != s {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

func TestAutocompleteResponseEmpty(t *testing.T) {
	// The API guide says an empty search returns an empty array, not null.
	// This test locks in that contract.
	resp := models.AutocompleteResponse{Suggestions: []models.Suggestion{}}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(raw) != `{"suggestions":[]}` {
		t.Fatalf("expected empty array, got: %s", raw)
	}
}

func TestAutocompleteResponseWithSuggestions(t *testing.T) {
	resp := models.AutocompleteResponse{
		Suggestions: []models.Suggestion{
			{PlaceID: "p1", Text: "Toronto, Canada"},
			{PlaceID: "p2", Text: "Toronto, UK"},
		},
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var back models.AutocompleteResponse
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(back.Suggestions) != 2 {
		t.Fatalf("expected 2, got %d", len(back.Suggestions))
	}
	if back.Suggestions[0].PlaceID != "p1" {
		t.Fatalf("unexpected first suggestion: %+v", back.Suggestions[0])
	}
}

// Place

func TestPlaceJSON(t *testing.T) {
	p := models.Place{
		PlaceID:     "mock-toronto",
		City:        "Toronto",
		CountryCode: "CA",
		Label:       "Toronto, Canada",
	}

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// All four keys must be present, with the exact casing from the guide.
	for _, key := range []string{`"placeId"`, `"city"`, `"countryCode"`, `"label"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("missing key %s in %s", key, raw)
		}
	}

	var back models.Place
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back != p {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

// Event

func TestEventJSONKeys(t *testing.T) {
	e := models.Event{
		ID:        "ev1",
		Name:      "Concert",
		ImageURL:  "https://img/1.jpg",
		Date:      "2026-10-15 19:30",
		Venue:     "Arena A",
		Info:      "An evening of music.",
		TicketURL: "https://www.ticketmaster.com/event/ev1",
		Category:  "Music",
	}

	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	expectedKeys := []string{
		`"id"`, `"name"`, `"imageUrl"`, `"date"`,
		`"venue"`, `"info"`, `"ticketUrl"`, `"category"`,
	}
	for _, key := range expectedKeys {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("missing key %s in %s", key, raw)
		}
	}

	var back models.Event
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back != e {
		t.Fatalf("round-trip mismatch\n got: %+v\nwant: %+v", back, e)
	}
}

func TestEventZeroValues(t *testing.T) {
	// An event with no fields still marshals cleanly.
	var e models.Event
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.HasPrefix(string(raw), "{") {
		t.Fatalf("expected object, got: %s", raw)
	}
}

// EventSection

func TestEventSectionOmitEmptyErr(t *testing.T) {
	// When Err is empty, it must not appear in the JSON — the listing
	// template and any API consumer rely on this.
	sec := models.EventSection{
		Category: "Music",
		Events:   []models.Event{},
		Cache:    "HIT",
	}

	raw, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if strings.Contains(string(raw), `"err"`) {
		t.Fatalf("expected err to be omitted, got: %s", raw)
	}
	if !strings.Contains(string(raw), `"cache":"HIT"`) {
		t.Fatalf("expected cache key, got: %s", raw)
	}
}

func TestEventSectionIncludesErrWhenSet(t *testing.T) {
	sec := models.EventSection{
		Category: "Sports",
		Err:      "sports failed",
		Cache:    "ERROR",
	}

	raw, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(raw), `"err":"sports failed"`) {
		t.Fatalf("expected err key, got: %s", raw)
	}
}

func TestEventSectionRoundTrip(t *testing.T) {
	sec := models.EventSection{
		Category: "Music",
		Events: []models.Event{
			{ID: "e1", Name: "One"},
			{ID: "e2", Name: "Two"},
		},
		Cache: "MISS",
	}

	raw, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var back models.EventSection
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back.Category != "Music" || back.Cache != "MISS" {
		t.Fatalf("scalar fields lost: %+v", back)
	}
	if len(back.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(back.Events))
	}
}

// ListingData — internal struct, no JSON tags

func TestListingDataFields(t *testing.T) {
	// ListingData is used internally by the controller, not serialized.
	// Just verify the fields exist and can be assigned.
	ld := models.ListingData{
		City:        "Toronto",
		CountryCode: "CA",
		Music:       models.EventSection{Category: "Music"},
		Sports:      models.EventSection{Category: "Sports"},
	}
	if ld.City != "Toronto" || ld.CountryCode != "CA" {
		t.Fatalf("unexpected: %+v", ld)
	}
	if ld.Music.Category != "Music" || ld.Sports.Category != "Sports" {
		t.Fatalf("section categories wrong: %+v", ld)
	}
}

// ErrorResponse

func TestErrorResponseJSON(t *testing.T) {
	er := models.ErrorResponse{Error: "Select a valid city suggestion."}

	raw, err := json.Marshal(er)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	want := `{"error":"Select a valid city suggestion."}`
	if string(raw) != want {
		t.Fatalf("wrong JSON\n got: %s\nwant: %s", raw, want)
	}

	var back models.ErrorResponse
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back != er {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}