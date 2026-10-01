package services

import (
	"fmt"
	"strings"
	"sync"
	"net/url"
	"errors"
	"eventx/cache"
	"eventx/models"
)
type Provider interface {
	Autocomplete(input,sessionToken string)([]models.Suggestion,error)
	Resolve(placeID,sessionToken string)(*models.Place,error)
	ListEvents(city,countryCode,category string)([]models.Event,error)
	GetEvent(eventID string)(*models.Event,error)
}
type EventService struct {
	provider Provider
}
func NewEventService(p Provider) *EventService {
	return &EventService{provider:p}
}
type sectionResult struct {
	section models.EventSection
}
// Concurrently fetch Music and Sports.
func (s *EventService)GetListing(city,countryCode,scenario string)(models.EventSection,models.EventSection) {
	musicCh:=make(chan sectionResult,1)
	sportsCh:=make(chan sectionResult,1)
	fetch:=func(category string,ch chan<-sectionResult) {
		key:=fmt.Sprintf("%s|%s|%s|%s",city,countryCode,category,scenario)
		var cached []models.Event
		var hit bool
		if cache.Default!=nil {
			cached,hit =cache.Default.Get(key)
		}
		if hit{
			ch <- sectionResult{section:models.EventSection{
				Category:category,
				Events: cached,
				Cache:"HIT",
			}}
			return
		}
		events,err:=s.provider.ListEvents(city,countryCode,category)
		if err!=nil {
			ch <- sectionResult{section:models.EventSection{
				Category:category,
				Err: err.Error(),
				Cache:"ERROR",
			}}
			return
		}
		if cache.Default !=nil {
			cache.Default.Set(key,events)
		}
		ch <- sectionResult{section:models.EventSection{
			Category:category,
			Events:events,
			Cache: "MISS",
		}}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func(){defer wg.Done();fetch("Music",musicCh)}()
	go func(){defer wg.Done();fetch("Sports",sportsCh)}()
	music:= <-musicCh
	sports:= <-sportsCh
	wg.Wait()
	return music.section,sports.section
}
func ValidateTicketURL(rawURL string, approvedHosts []string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.New("invalid URL format")
	}

	if parsedURL.Scheme != "https" { // HTTPS only
		return "", errors.New("ticket destination must use https")
	}

	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "" {
		return "", errors.New("missing hostname")
	}

	for _, approved := range approvedHosts {
		approved = strings.ToLower(strings.TrimSpace(approved))
		if approved == "" {
			continue
		}
		if hostname == approved || strings.HasSuffix(hostname, "."+approved) {
			return rawURL, nil 
		}
	}

	return "", errors.New("host not approved")
}