package services

import (
	"fmt"
	"strings"
	"sync"
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

func ValidateTicketURL(rawURL string,approvedHosts []string)(string, error) {
	if rawURL==""{
		return "",fmt.Errorf("ticket link unavailable")
	}
	if !strings.HasPrefix(rawURL,"https://") {
		return "",fmt.Errorf("ticket destination blocked")
	}
	rest:=strings.TrimPrefix(rawURL, "https://")
	host:=rest
	if i := strings.IndexAny(rest, "/?"); i >= 0 {
		host =rest[:i]
	}
	for _, h:=range approvedHosts {
		if strings.EqualFold(host, h) {
			return rawURL,nil
		}
	}
	return "",fmt.Errorf("ticket destination blocked")
}