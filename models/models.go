package models

type Suggestion struct {
	PlaceID string `json:"placeId"`
	Text    string `json:"text"`
}

type AutocompleteResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
}

type Place struct {
	PlaceID     string `json:"placeId"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Label       string `json:"label"`
}

type Event struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	ImageURL string   `json:"imageUrl"`
	Date     string   `json:"date"`
	Venue    string   `json:"venue"`
	Info     string   `json:"info"`
	TicketURL string  `json:"ticketUrl"`
	Category string   `json:"category"`
}

type EventSection struct {
	Category string  `json:"category"`
	Events   []Event `json:"events"`
	Err      string  `json:"err,omitempty"`
	Cache    string  `json:"cache"` // HIT, MISS, ERROR
}

type ListingData struct {
	City        string
	CountryCode string
	Music       EventSection
	Sports      EventSection
}

type ErrorResponse struct {
	Error string `json:"error"`
}