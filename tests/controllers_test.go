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
	_ "eventx/routers"

	"github.com/beego/beego/v2/server/web"
)

// Beego test harness — set up views/static paths once

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

// base.go — Health endpoint

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

// home.go — Home page

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
		t.Fatalf("expected HTML doctype in body, got: %s", body[:min(200, len(body))])
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

// Sanity: unknown routes return 404

func TestUnknownRoute_Returns404(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/does-not-exist")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// small helper for older Go versions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}