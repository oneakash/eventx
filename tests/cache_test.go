package tests

import (
	"sync"
	"testing"
	"time"

	"eventx/cache"
	"eventx/models"
)

func eventsFor(id string) []models.Event {
	return []models.Event{{ID: id, Name: "Event " + id}}
}

// Init

func TestInitCreatesEmptyCache(t *testing.T) {
	cache.Init(time.Minute)
	if cache.Default == nil {
		t.Fatal("Default cache is nil after Init")
	}
	if n := cache.Default.Reset(); n != 0 {
		t.Fatalf("expected empty cache, got %d entries", n)
	}
}

// Get / Set

func TestGetOnEmptyCacheMisses(t *testing.T) {
	cache.Init(time.Minute)
	if _, ok := cache.Default.Get("Toronto|CA|Music|"); ok {
		t.Fatal("expected miss on empty cache")
	}
}

func TestSetThenGetHits(t *testing.T) {
	cache.Init(time.Minute)
	key := "Toronto|CA|Music|"
	want := eventsFor("e1")

	cache.Default.Set(key, want)

	got, ok := cache.Default.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != 1 || got[0].ID != "e1" {
		t.Fatalf("cached value mismatch: %+v", got)
	}
}

func TestSetOverwritesExistingEntry(t *testing.T) {
	cache.Init(time.Minute)
	key := "Toronto|CA|Music|"

	cache.Default.Set(key, eventsFor("first"))
	cache.Default.Set(key, eventsFor("second"))

	got, ok := cache.Default.Get(key)
	if !ok {
		t.Fatal("expected hit")
	}
	if got[0].ID != "second" {
		t.Fatalf("expected overwrite, got %s", got[0].ID)
	}
}

func TestSetStoresEmptySlice(t *testing.T) {
	// Empty event lists are still "successful" per the API guide —
	// they must be cacheable (HTTP 200 with an empty section).
	cache.Init(time.Minute)
	key := "Dhaka|BD|Music|"
	cache.Default.Set(key, []models.Event{})

	got, ok := cache.Default.Get(key)
	if !ok {
		t.Fatal("expected hit for empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d", len(got))
	}
}

func TestCacheHitAndExpiry(t *testing.T) {
	cache.Init(50 * time.Millisecond)
	key := "Toronto|CA|Music|"

	cache.Default.Set(key, eventsFor("e1"))

	if _, ok := cache.Default.Get(key); !ok {
		t.Fatal("expected hit before expiry")
	}

	time.Sleep(80 * time.Millisecond)

	if _, ok := cache.Default.Get(key); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestExpiredEntryIsDeletedOnAccess(t *testing.T) {
	// Get must remove the expired key so it doesn't linger in the map.
	cache.Init(30 * time.Millisecond)
	key := "Toronto|CA|Music|"
	cache.Default.Set(key, eventsFor("e1"))

	time.Sleep(50 * time.Millisecond)
	cache.Default.Get(key) // triggers delete

	// Reset reports remaining entries; the expired one must be gone.
	if n := cache.Default.Reset(); n != 0 {
		t.Fatalf("expected expired entry to be gone, still %d present", n)
	}
}

func TestDifferentKeysAreIsolated(t *testing.T) {
	cache.Init(time.Minute)
	cache.Default.Set("Toronto|CA|Music|", eventsFor("t-music"))
	cache.Default.Set("Toronto|CA|Sports|", eventsFor("t-sports"))
	cache.Default.Set("London|GB|Music|", eventsFor("l-music"))

	m, _ := cache.Default.Get("Toronto|CA|Music|")
	s, _ := cache.Default.Get("Toronto|CA|Sports|")
	l, _ := cache.Default.Get("London|GB|Music|")

	if m[0].ID != "t-music" || s[0].ID != "t-sports" || l[0].ID != "l-music" {
		t.Fatalf("keys not isolated: %s %s %s", m[0].ID, s[0].ID, l[0].ID)
	}
}

// Invalidate (single key)

func TestInvalidateRemovesSingleKey(t *testing.T) {
	cache.Init(time.Minute)
	key := "Toronto|CA|Music|"
	cache.Default.Set(key, eventsFor("e1"))

	if !cache.Default.Invalidate(key) {
		t.Fatal("expected Invalidate to report true")
	}
	if _, ok := cache.Default.Get(key); ok {
		t.Fatal("expected miss after invalidate")
	}
}

func TestInvalidateMissingKeyReturnsFalse(t *testing.T) {
	cache.Init(time.Minute)
	if cache.Default.Invalidate("nope") {
		t.Fatal("expected false for missing key")
	}
}

func TestInvalidateDoesNotAffectOtherKeys(t *testing.T) {
	cache.Init(time.Minute)
	cache.Default.Set("Toronto|CA|Music|", eventsFor("m"))
	cache.Default.Set("Toronto|CA|Sports|", eventsFor("s"))

	cache.Default.Invalidate("Toronto|CA|Music|")

	if _, ok := cache.Default.Get("Toronto|CA|Sports|"); !ok {
		t.Fatal("sports should still be cached")
	}
}

// InvalidateByPrefix

func TestInvalidateByPrefixRemovesMatching(t *testing.T) {
	cache.Init(time.Minute)
	cache.Default.Set("Toronto|CA|Music|", eventsFor("1"))
	cache.Default.Set("Toronto|CA|Sports|", eventsFor("2"))
	cache.Default.Set("London|GB|Music|", eventsFor("3"))

	n := cache.Default.InvalidateByPrefix("Toronto|CA|")
	if n != 2 {
		t.Fatalf("expected 2 invalidated, got %d", n)
	}
	if _, ok := cache.Default.Get("London|GB|Music|"); !ok {
		t.Fatal("London entry should survive")
	}
}

func TestInvalidateByPrefixEmptyPrefixRemovesAll(t *testing.T) {
	cache.Init(time.Minute)
	cache.Default.Set("a", eventsFor("1"))
	cache.Default.Set("b", eventsFor("2"))

	n := cache.Default.InvalidateByPrefix("")
	if n != 2 {
		t.Fatalf("expected 2 with empty prefix, got %d", n)
	}
}

func TestInvalidateByPrefixNoMatch(t *testing.T) {
	cache.Init(time.Minute)
	cache.Default.Set("Toronto|CA|Music|", eventsFor("1"))

	if n := cache.Default.InvalidateByPrefix("London|GB|"); n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
	if _, ok := cache.Default.Get("Toronto|CA|Music|"); !ok {
		t.Fatal("Toronto entry should survive")
	}
}

func TestInvalidateByPrefixLongerThanKey(t *testing.T) {
	// Prefix is longer than the key itself; must not panic and must not match.
	cache.Init(time.Minute)
	cache.Default.Set("a", eventsFor("1"))

	if n := cache.Default.InvalidateByPrefix("abc"); n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
	if _, ok := cache.Default.Get("a"); !ok {
		t.Fatal("entry should survive")
	}
}

func TestInvalidateByPrefixExactLength(t *testing.T) {
	// Prefix matches the whole key exactly; must match.
	cache.Init(time.Minute)
	cache.Default.Set("abc", eventsFor("1"))

	if n := cache.Default.InvalidateByPrefix("abc"); n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}
// Reset

func TestResetReturnsCountAndClears(t *testing.T) {
	cache.Init(time.Minute)
	cache.Default.Set("a", eventsFor("1"))
	cache.Default.Set("b", eventsFor("2"))

	if n := cache.Default.Reset(); n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
	if n := cache.Default.Reset(); n != 0 {
		t.Fatalf("expected 0 after first reset, got %d", n)
	}
	if _, ok := cache.Default.Get("a"); ok {
		t.Fatal("expected miss after reset")
	}
}

// Concurrency (run with -race)

func TestConcurrentAccessIsSafe(t *testing.T) {
	cache.Init(time.Minute)

	var wg sync.WaitGroup
	const workers = 20
	const iterations = 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Default.Set("key", eventsFor("e"))
			}
		}(i)
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Default.Get("key")
			}
		}(i)
	}

	for i := 0; i < workers/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Default.Invalidate("key")
			}
		}(i)
	}

	wg.Wait()
}