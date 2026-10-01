package controllers

import (
	"os"
	"time"

	"eventx/cache"
	"eventx/clients"
	"eventx/models"
	"eventx/services"

	"github.com/beego/beego/v2/server/web"
)

var ProviderFactory = func() services.Provider {
	gk := os.Getenv("GOOGLE_PLACES_API_KEY")
	tk := os.Getenv("TICKETMASTER_API_KEY")
	return &compositeProvider{
		google:       &clients.GoogleClient{APIKey: gk},
		ticketmaster: &clients.TicketmasterClient{APIKey: tk},
	}
}

func getProvider() services.Provider {
	return ProviderFactory()
}

func init() {
	ttl := 300
	if ttlStr, _ := web.AppConfig.String("CACHE_TTL"); ttlStr != "" {
		if v, err := time.ParseDuration(ttlStr + "s"); err == nil {
			cache.Init(v)
			return
		}
	}
	cache.Init(time.Duration(ttl) * time.Second)
}

type compositeProvider struct {
	google       *clients.GoogleClient
	ticketmaster *clients.TicketmasterClient
}

func (c *compositeProvider) Autocomplete(input, sessionToken string) ([]models.Suggestion, error) {
	return c.google.Autocomplete(input, sessionToken)
}
func (c *compositeProvider) Resolve(placeID, sessionToken string) (*models.Place, error) {
	return c.google.Resolve(placeID, sessionToken)
}
func (c *compositeProvider) ListEvents(city, countryCode, category string) ([]models.Event, error) {
	return c.ticketmaster.ListEvents(city, countryCode, category)
}
func (c *compositeProvider) GetEvent(eventID string) (*models.Event, error) {
	return c.ticketmaster.GetEvent(eventID)
}