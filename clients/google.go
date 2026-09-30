package clients
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"eventx/models"
)
type GoogleClient struct {
	APIKey string
}
type googleAutoReq struct{
	Input string `json:"input"`
	IncludedPrimaryTypes []string `json:"includedPrimaryTypes"`
	SessionToken string `json:"sessionToken,omitempty"`
	LanguageCode string `json:"languageCode"`
}
type googleAutoResp struct{
	Suggestions []struct{
		PlacePrediction struct{
			PlaceID string `json:"placeId"`
			Text  struct{
				Text string `json:"text"`
			} `json:"text"`
		} `json:"placePrediction"`
	} `json:"suggestions"`
}
func (g *GoogleClient) Autocomplete(input,sessionToken string) ([]models.Suggestion, error) {
	body,_:= json.Marshal(googleAutoReq{
		Input: input,
		IncludedPrimaryTypes:[]string{"(cities)"},
		SessionToken:  sessionToken,
		LanguageCode: "en",
	})
	req, _:=http.NewRequest("POST",
		"https://places.googleapis.com/v1/places:autocomplete",
		bytes.NewReader(body))
	req.Header.Set("Content-Type","application/json")
	req.Header.Set("X-Goog-Api-Key",g.APIKey)
	resp, err:=http.DefaultClient.Do(req)
	if err!=nil{
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode!=200{
		b,_:=io.ReadAll(resp.Body)
		return nil,fmt.Errorf("google autocomplete %d: %s",resp.StatusCode,string(b))
	}
	var gr googleAutoResp
	if err :=json.NewDecoder(resp.Body).Decode(&gr);err!=nil{
		return nil,err
	}
	var out []models.Suggestion
	for _,s:=range gr.Suggestions{
		if s.PlacePrediction.PlaceID == "" {
			continue
		}
		out=append(out, models.Suggestion{
			PlaceID:s.PlacePrediction.PlaceID,
			Text:s.PlacePrediction.Text.Text,
		})
		if len(out)>=5 {
			break
		}
	}
	if out==nil{
		out =[]models.Suggestion{}
	}
	return out,nil
}

type googlePlaceResp struct {
	AddressComponents []struct {
		LongText string `json:"longText"`
		ShortText string `json:"shortText"`
		Types []string `json:"types"`
	} `json:"addressComponents"`
}

func (g *GoogleClient)Resolve(placeID, sessionToken string)(*models.Place, error){
	u:=fmt.Sprintf("https://places.googleapis.com/v1/places/%s?sessionToken=%s&languageCode=en",
		url.PathEscape(placeID), url.QueryEscape(sessionToken))
	req,_:=http.NewRequest("GET",u,nil)
	req.Header.Set("X-Goog-Api-Key",g.APIKey)
	req.Header.Set("X-Goog-FieldMask","addressComponents")

	resp,err:= http.DefaultClient.Do(req)
	if err!=nil{
		return nil,err
	}
	defer resp.Body.Close()
	if resp.StatusCode!=200 {
		b,_:=io.ReadAll(resp.Body)
		return nil,fmt.Errorf("google place %d: %s",resp.StatusCode,string(b))
	}
	var gr googlePlaceResp
	if err:=json.NewDecoder(resp.Body).Decode(&gr);err!=nil{
		return nil,err
	}
	p:=&models.Place{PlaceID:placeID}
	for _, c :=range gr.AddressComponents {
		for _, t := range c.Types {
			switch t {
			case "locality":
				p.City = c.LongText
			case "postal_town":
				if p.City == "" {
					p.City = c.LongText
				}
			case "country":
				p.CountryCode = c.ShortText
			}
		}
	}
	if p.City=="" || p.CountryCode==""{
		return nil,fmt.Errorf("incomplete place data")
	}
	p.Label=p.City+", "+p.CountryCode
	return p,nil
}