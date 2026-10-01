package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"eventx/clients"
)

// Autocomplete

func TestGoogleAutocomplete_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/places:autocomplete") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Goog-Api-Key") != "test-key" {
			t.Errorf("missing or wrong API key header")
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Errorf("expected JSON content-type, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("could not decode request body: %v", err)
		}
		if body["input"] != "Tor" {
			t.Errorf("expected input=Tor, got %v", body["input"])
		}
		if body["languageCode"] != "en" {
			t.Errorf("expected languageCode=en, got %v", body["languageCode"])
		}
		if body["sessionToken"] != "tok-1" {
			t.Errorf("expected sessionToken=tok-1, got %v", body["sessionToken"])
		}
		primary, ok := body["includedPrimaryTypes"].([]any)
		if !ok || len(primary) != 1 || primary[0] != "(cities)" {
			t.Errorf("expected includedPrimaryTypes=[(cities)], got %v", body["includedPrimaryTypes"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"suggestions":[
				{"placePrediction":{"placeId":"p1","text":{"text":"Toronto, Canada"}}},
				{"placePrediction":{"placeId":"p2","text":{"text":"Toronto, UK"}}}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	got, err := g.Autocomplete("Tor", "tok-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(got))
	}
	if got[0].PlaceID != "p1" || got[0].Text != "Toronto, Canada" {
		t.Fatalf("unexpected first suggestion: %+v", got[0])
	}
}

func TestGoogleAutocomplete_EmptySuggestions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"suggestions":[]}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	got, err := g.Autocomplete("Xyz", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestGoogleAutocomplete_SkipsEntriesWithoutPlaceID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"suggestions":[
				{"placePrediction":{"placeId":"","text":{"text":"skip me"}}},
				{"placePrediction":{"placeId":"p1","text":{"text":"keep me"}}},
				{"placePrediction":{"placeId":"","text":{"text":"skip me too"}}}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	got, err := g.Autocomplete("Tor", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].PlaceID != "p1" {
		t.Fatalf("expected only p1, got %+v", got)
	}
}

func TestGoogleAutocomplete_CapsAtFive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"suggestions":[
				{"placePrediction":{"placeId":"p1","text":{"text":"1"}}},
				{"placePrediction":{"placeId":"p2","text":{"text":"2"}}},
				{"placePrediction":{"placeId":"p3","text":{"text":"3"}}},
				{"placePrediction":{"placeId":"p4","text":{"text":"4"}}},
				{"placePrediction":{"placeId":"p5","text":{"text":"5"}}},
				{"placePrediction":{"placeId":"p6","text":{"text":"6"}}},
				{"placePrediction":{"placeId":"p7","text":{"text":"7"}}},
				{"placePrediction":{"placeId":"p8","text":{"text":"8"}}},
				{"placePrediction":{"placeId":"p9","text":{"text":"9"}}},
				{"placePrediction":{"placeId":"p10","text":{"text":"10"}}}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	got, err := g.Autocomplete("Tor", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5, got %d", len(got))
	}
	if got[4].PlaceID != "p5" {
		t.Fatalf("expected p5 last, got %s", got[4].PlaceID)
	}
}

func TestGoogleAutocomplete_HTTPError(t *testing.T) {
	for _, code := range []int{400, 403, 429, 500, 502} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				w.Write([]byte(`{"error":"boom"}`))
			}))
			defer srv.Close()

			g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
			_, err := g.Autocomplete("Tor", "tok")
			if err == nil {
				t.Fatalf("expected error for %d", code)
			}
			if !strings.Contains(err.Error(), "google autocomplete") {
				t.Fatalf("expected wrapped error, got: %v", err)
			}
		})
	}
}

func TestGoogleAutocomplete_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	_, err := g.Autocomplete("Tor", "tok")
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestGoogleAutocomplete_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close() // now nothing listens; requests fail at the transport layer

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: closedURL}
	_, err := g.Autocomplete("Tor", "tok")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// Resolve

func TestGoogleResolve_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/v1/places/mock-toronto") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("sessionToken") != "tok-1" {
			t.Errorf("missing sessionToken in query")
		}
		if r.URL.Query().Get("languageCode") != "en" {
			t.Errorf("missing languageCode in query")
		}
		if r.Header.Get("X-Goog-Api-Key") != "test-key" {
			t.Errorf("missing API key header")
		}
		if r.Header.Get("X-Goog-FieldMask") != "addressComponents" {
			t.Errorf("missing or wrong field mask")
		}

		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"Toronto","shortText":"Toronto","types":["locality"]},
				{"longText":"Canada","shortText":"CA","types":["country"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	p, err := g.Resolve("mock-toronto", "tok-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PlaceID != "mock-toronto" {
		t.Fatalf("wrong place ID: %s", p.PlaceID)
	}
	if p.City != "Toronto" || p.CountryCode != "CA" {
		t.Fatalf("unexpected place: %+v", p)
	}
	if p.Label != "Toronto, CA" {
		t.Fatalf("unexpected label: %s", p.Label)
	}
}

func TestGoogleResolve_PostalTownFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"Smallville","shortText":"Smallville","types":["postal_town"]},
				{"longText":"USA","shortText":"US","types":["country"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	p, err := g.Resolve("p1", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.City != "Smallville" || p.CountryCode != "US" {
		t.Fatalf("unexpected place: %+v", p)
	}
}

func TestGoogleResolve_LocalityWinsOverPostalTown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"PostalTown","shortText":"PT","types":["postal_town"]},
				{"longText":"RealCity","shortText":"RC","types":["locality"]},
				{"longText":"UK","shortText":"GB","types":["country"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	p, err := g.Resolve("p1", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.City != "RealCity" {
		t.Fatalf("expected locality to win, got %s", p.City)
	}
}

func TestGoogleResolve_MissingCity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"Canada","shortText":"CA","types":["country"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := g.Resolve("p1", "tok"); err == nil {
		t.Fatal("expected error for missing city")
	}
}

func TestGoogleResolve_MissingCountry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"Toronto","shortText":"Toronto","types":["locality"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := g.Resolve("p1", "tok"); err == nil {
		t.Fatal("expected error for missing country")
	}
}

func TestGoogleResolve_EmptyComponents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"addressComponents":[]}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := g.Resolve("p1", "tok"); err == nil {
		t.Fatal("expected error on empty components")
	}
}

func TestGoogleResolve_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	_, err := g.Resolve("p1", "tok")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "google place") {
		t.Fatalf("expected wrapped error, got: %v", err)
	}
}

func TestGoogleResolve_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not valid}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := g.Resolve("p1", "tok"); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestGoogleResolve_EncodesPlaceID(t *testing.T) {
	var rawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawPath = r.URL.RawPath
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"X","shortText":"X","types":["locality"]},
				{"longText":"Y","shortText":"YY","types":["country"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := g.Resolve("abc/def", "tok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(rawPath, "abc%2Fdef") {
		t.Fatalf("expected encoded slash in RawPath, got: %s", rawPath)
	}
}

func TestGoogleResolve_RealisticPlaceIDUnchanged(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Write([]byte(`{
			"addressComponents":[
				{"longText":"Toronto","shortText":"Toronto","types":["locality"]},
				{"longText":"Canada","shortText":"CA","types":["country"]}
			]
		}`))
	}))
	defer srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: srv.URL}
	if _, err := g.Resolve("ChIJN1t_tDeuEmsRUsoyG83frY4", "tok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(capturedPath, "/ChIJN1t_tDeuEmsRUsoyG83frY4") {
		t.Fatalf("place ID path wrong: %s", capturedPath)
	}
}

func TestGoogleResolve_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	g := &clients.GoogleClient{APIKey: "test-key", BaseURL: closedURL}
	_, err := g.Resolve("p1", "tok")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// Base() — covers the default-URL branch

func TestGoogleBase_DefaultURL(t *testing.T) {
	g := &clients.GoogleClient{APIKey: "k"}
	if got := g.Base(); got != "https://places.googleapis.com" {
		t.Fatalf("unexpected default base: %s", got)
	}
}

func TestGoogleBase_CustomURL(t *testing.T) {
	g := &clients.GoogleClient{APIKey: "k", BaseURL: "https://example.test"}
	if got := g.Base(); got != "https://example.test" {
		t.Fatalf("unexpected base: %s", got)
	}
}