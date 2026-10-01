package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"eventx/models"
)
type TicketmasterClient struct{
	APIKey string
	BaseURL string 
}
func (t *TicketmasterClient) Base() string { // ← NEW method
	if t.BaseURL != "" {
		return t.BaseURL
	}
	return "https://app.ticketmaster.com/discovery/v2"
}
type tmListResp struct{
	Embedded struct{
		Events []tmEvent `json:"events"`
	} `json:"_embedded"`
}

type tmEvent struct{
	ID string `json:"id"`
	Name string `json:"name"`
	Info string `json:"info"`
	URL  string `json:"url"`
	Images []struct{
		URL string `json:"url"`
		Ratio string `json:"ratio"`
		Width int    `json:"width"`
	} `json:"images"`
	Dates struct{
		Start struct{
			LocalDate string `json:"localDate"`
			LocalTime string `json:"localTime"`
		} `json:"start"`
	} `json:"dates"`
	Embedded struct{
		Venues []struct{
			Name string `json:"name"`
		} `json:"venues"`
	} `json:"_embedded"`
}

func (t *TicketmasterClient)ListEvents(city, countryCode, category string)([]models.Event,error){
	u:=fmt.Sprintf("%s/events.json?apikey=%s&city=%s&countryCode=%s&classificationName=%s&size=6",
		t.Base(),
		url.QueryEscape(t.APIKey),
		url.QueryEscape(city),
		url.QueryEscape(countryCode),
		url.QueryEscape(category))
	resp,err :=http.Get(u)
	if err!= nil{
		return nil,err
	}
	defer resp.Body.Close()
	if resp.StatusCode!=200 {
		b, _:= io.ReadAll(resp.Body)
		return nil,fmt.Errorf("ticketmaster %d: %s", resp.StatusCode, string(b))
	}
	var tr tmListResp
	if err:= json.NewDecoder(resp.Body).Decode(&tr);err != nil {
		return nil,err
	}
	var out []models.Event
	for _,e:=range tr.Embedded.Events{
		ev:=models.Event{
			ID: e.ID,
			Name:e.Name,
			Info:e.Info,
			TicketURL:e.URL,
			Category:category,
		}
		if len(e.Images)>0{
			ev.ImageURL=e.Images[0].URL
		}
		if e.Dates.Start.LocalDate!=""{
			ev.Date = e.Dates.Start.LocalDate + " " + e.Dates.Start.LocalTime
		}
		if len(e.Embedded.Venues)>0{
			ev.Venue=e.Embedded.Venues[0].Name
		}
		out=append(out,ev)
	}
	if out==nil{
		out=[]models.Event{}
	}
	return out,nil
}
func (t *TicketmasterClient)GetEvent(eventID string) (*models.Event, error){
	u:=fmt.Sprintf("%s/events/%s.json?apikey=%s",
		t.Base(), url.PathEscape(eventID), url.QueryEscape(t.APIKey))
	resp,err:=http.Get(u)
	if err !=nil{
		return nil,err
	}
	defer resp.Body.Close()
	if resp.StatusCode ==404{
		return nil,fmt.Errorf("not found")
	}
	if resp.StatusCode!=200{
		b, _:=io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ticketmaster %d: %s", resp.StatusCode, string(b))
	}
	var e tmEvent
	if err:= json.NewDecoder(resp.Body).Decode(&e);err!= nil{
		return nil,err
	}
	ev:=models.Event{
		ID: e.ID,
		Name: e.Name,
		Info:e.Info,
		TicketURL: e.URL,
	}
	if len(e.Images)>0{
		ev.ImageURL=e.Images[0].URL
	}
	if e.Dates.Start.LocalDate !=""{
		ev.Date=e.Dates.Start.LocalDate + " " + e.Dates.Start.LocalTime
	}
	if len(e.Embedded.Venues) > 0{
		ev.Venue =e.Embedded.Venues[0].Name
	}
	return &ev,nil
}