package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"eventx/controllers"
	"eventx/models"
	"eventx/services"

	_ "eventx/routers"

	"github.com/beego/beego/v2/server/web"
)

var beegoSetupOnce sync.Once

func setupBeego() {
	beegoSetupOnce.Do(func() {
		_, file, _, _ := runtime.Caller(0)
		root := filepath.Dir(filepath.Dir(file))
		web.TestBeegoInit(root)
	})
}

func doRequest(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	setupBeego()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(rec, req)
	return rec
}

type ctrlFake struct {
	autocompleteFn func(input, token string) ([]models.Suggestion, error)
	resolveFn      func(placeID, token string) (*models.Place, error)
	listEventsFn   func(city, cc, cat string) ([]models.Event, error)
	getEventFn     func(eventID string) (*models.Event, error)
}

func (f *ctrlFake) Autocomplete(input, token string) ([]models.Suggestion, error) {
	if f.autocompleteFn == nil {
		return []models.Suggestion{}, nil
	}
	return f.autocompleteFn(input, token)
}

func (f *ctrlFake) Resolve(placeID, token string) (*models.Place, error) {
	if f.resolveFn == nil {
		return &models.Place{}, nil
	}
	return f.resolveFn(placeID, token)
}

func (f *ctrlFake) ListEvents(city, cc, cat string) ([]models.Event, error) {
	if f.listEventsFn == nil {
		return []models.Event{}, nil
	}
	return f.listEventsFn(city, cc, cat)
}

func (f *ctrlFake) GetEvent(eventID string) (*models.Event, error) {
	if f.getEventFn == nil {
		return &models.Event{}, nil
	}
	return f.getEventFn(eventID)
}

// withProvider swaps the controller provider for the duration of one test.
func withProvider(t *testing.T, p services.Provider) {
	t.Helper()
	old := controllers.ProviderFactory
	controllers.ProviderFactory = func() services.Provider { return p }
	t.Cleanup(func() { controllers.ProviderFactory = old })
}

func TestHealth_ReturnsOK(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHealth_ReturnsJSON(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/healthz")

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON content type, got: %s", ct)
	}
}

func TestHealth_BodyShape(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/healthz")

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v (body: %s)", err, rec.Body.String())
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
	if body["runtime"] != "Beego" {
		t.Fatalf("expected runtime=Beego, got %q", body["runtime"])
	}
}

// ---------------------------------------------------------------------------
// home.go — Home page
// ---------------------------------------------------------------------------

func TestHome_ReturnsOK(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHome_RendersHTML(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/")

	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("expected HTML doctype in body")
	}
}

func TestHome_ContainsHeaderBrand(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/")

	if !strings.Contains(rec.Body.String(), "Event Explorer") {
		t.Fatal("expected brand text in home page")
	}
}

func TestHome_ContainsSearchInput(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/")

	body := rec.Body.String()
	if !strings.Contains(body, `id="cityInput"`) {
		t.Fatal("expected cityInput element in home page")
	}
	if !strings.Contains(body, `id="searchBtn"`) {
		t.Fatal("expected searchBtn element in home page")
	}
}

func TestHome_IncludesAutocompleteScript(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/")

	if !strings.Contains(rec.Body.String(), "/static/js/autocomplete.js") {
		t.Fatal("expected autocomplete.js script tag")
	}
}

// ---------------------------------------------------------------------------
// Unknown routes
// ---------------------------------------------------------------------------

func TestUnknownRoute_Returns404(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/does-not-exist")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}