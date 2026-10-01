package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"eventx/models"
)

func TestLocationAutocomplete_Success(t *testing.T) {
	withProvider(t, &ctrlFake{
		autocompleteFn: func(input, token string) ([]models.Suggestion, error) {
			return []models.Suggestion{{PlaceID: "p1", Text: "Toronto, Canada"}}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input=Tor&sessionToken=demo-1")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var body models.AutocompleteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Suggestions) != 1 || body.Suggestions[0].PlaceID != "p1" {
		t.Fatalf("unexpected suggestions: %+v", body.Suggestions)
	}
}

func TestLocationAutocomplete_InputTooLong(t *testing.T) {
	withProvider(t, &ctrlFake{})

	longInput := strings.Repeat("a", 201)
	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input="+longInput)

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Input too long") {
		t.Fatalf("expected 'Input too long' error, got: %s", rec.Body.String())
	}
}

func TestLocationAutocomplete_InvalidSessionToken(t *testing.T) {
	withProvider(t, &ctrlFake{})

	// Contains characters not allowed by the regex (dot)
	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input=Tor&sessionToken=bad.token")

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid session token") {
		t.Fatalf("expected invalid token error, got: %s", rec.Body.String())
	}
}

func TestLocationAutocomplete_TokenTooLong(t *testing.T) {
	withProvider(t, &ctrlFake{})

	// Regex caps at 36 chars
	longToken := strings.Repeat("a", 37)
	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input=Tor&sessionToken="+longToken)

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLocationAutocomplete_EmptyTokenAllowed(t *testing.T) {
	withProvider(t, &ctrlFake{
		autocompleteFn: func(input, token string) ([]models.Suggestion, error) {
			return []models.Suggestion{}, nil
		},
	})

	// No sessionToken param — should be allowed (validation is skipped for empty)
	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input=Tor")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLocationAutocomplete_ProviderError(t *testing.T) {
	withProvider(t, &ctrlFake{
		autocompleteFn: func(input, token string) ([]models.Suggestion, error) {
			return nil, errors.New("google down")
		},
	})

	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input=Tor&sessionToken=demo-1")

	if rec.Code != 502 {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Location provider failed") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestLocationAutocomplete_NilSuggestionsBecomesEmptyArray(t *testing.T) {
	withProvider(t, &ctrlFake{
		autocompleteFn: func(input, token string) ([]models.Suggestion, error) {
			return nil, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/api/locations/autocomplete?input=Tor&sessionToken=demo-1")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Beego pretty-prints JSON in dev/test; strip whitespace before matching.
	compact := strings.ReplaceAll(rec.Body.String(), " ", "")
	compact = strings.ReplaceAll(compact, "\n", "")
	compact = strings.ReplaceAll(compact, "\t", "")

	if !strings.Contains(compact, `"suggestions":[]`) {
		t.Fatalf("expected empty array, got: %s", rec.Body.String())
	}
}

func TestLocationResolve_Success(t *testing.T) {
	withProvider(t, &ctrlFake{
		resolveFn: func(placeID, token string) (*models.Place, error) {
			return &models.Place{
				PlaceID:     placeID,
				City:        "Toronto",
				CountryCode: "CA",
				Label:       "Toronto, Canada",
			}, nil
		},
	})

	rec := doRequest(t, http.MethodGet, "/api/locations/mock-toronto?sessionToken=demo-1")

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var place models.Place
	if err := json.Unmarshal(rec.Body.Bytes(), &place); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if place.City != "Toronto" || place.CountryCode != "CA" {
		t.Fatalf("unexpected place: %+v", place)
	}
}

func TestLocationResolve_MissingToken(t *testing.T) {
	withProvider(t, &ctrlFake{})

	rec := doRequest(t, http.MethodGet, "/api/locations/mock-toronto")

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Missing place ID or session token") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestLocationResolve_InvalidToken(t *testing.T) {
	withProvider(t, &ctrlFake{})

	rec := doRequest(t, http.MethodGet, "/api/locations/mock-toronto?sessionToken=bad.token")

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLocationResolve_ProviderError(t *testing.T) {
	withProvider(t, &ctrlFake{
		resolveFn: func(placeID, token string) (*models.Place, error) {
			return nil, errors.New("google down")
		},
	})

	rec := doRequest(t, http.MethodGet, "/api/locations/mock-toronto?sessionToken=demo-1")

	if rec.Code != 502 {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Location lookup failed") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}