# Event Explorer — Beego Full-Stack

Users select a city via Google Places
autocomplete, browse Music and Sports events from Ticketmaster, view event
details, and continue safely to the ticket provider.

## Requirements
- Go 1.21+
- Beego v2

## Run (mock mode — no keys required)
```bash
go mod tidy
go run main.go
# open http://localhost:8080
```

## Run (live mode)
Set environment variables in `.env` file:
```
APP_MODE = live
GOOGLE_PLACES_API_KEY = <server-side key>
TICKETMASTER_API_KEY  = <server-side key>
```

## Routes

### Frontend (HTML)
| Method | Path | Purpose |
|---|---|---|
| GET | `/` | Home + city autocomplete |
| GET | `/events?city=&countryCode=` | Music + Sports listing |
| GET | `/events/:eventId` | Event details |
| GET | `/redirect/:eventId` | HTTP 302 to approved ticket URL |
| GET | `/demo/tickets/:eventId` | Mock-only safe local destination |

### JSON APIs
| Method | Path | Purpose |
|---|---|---|
| GET | `/api/locations/autocomplete?input=&sessionToken=` | City suggestions |
| GET | `/api/locations/:placeId?sessionToken=` | Resolve city + countryCode |
| GET | `/healthz` | Runtime health |

## Go Concepts Demonstrated
- **MVC + templates**: routers / controllers / models / services / views.
- **Goroutines + channels**: Music and Sports fetched concurrently in
  `services/events.go`; results/errors merged via channels.
- **Mutex-protected cache**: `cache/cache.go` — 5-min TTL, keyed by
  `city|country|category|scenario`, headers `X-Music-Cache` / `X-Sports-Cache`.
- **Safe redirects**: `ValidateTicketURL` enforces HTTPS + approved hostname.

## Tested Scenarios
- Full flow: Tor → Toronto → first Music event → View tickets
- Direct link: `/events/mock-toronto-music-1`
- Cache reuse: same listing twice → `X-*-Cache: HIT`
- Empty: search Dhaka → empty messages, HTTP 200
- Partial failure: one section fails, the other still renders
- Missing ticket: 5th Music event has no link
- Unsafe ticket: 6th Music event blocked (non-approved host)
- Missing event: `/events/does-not-exist` → 404 error page

## Unit Tests
```bash
go test ./tests/...
```
Covers cache hit/expiry, service success/failure, and ticket-link validation.
All tests run without live API keys.