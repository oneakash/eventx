package controllers
import (
	"strings"
	"eventx/models"
	"eventx/services"
)
type EventsController struct {
	BaseController
}
func (c *EventsController) List() {
	city:=c.GetString("city")
	countryCode:=c.GetString("countryCode")
	scenario:=c.GetString("scenario")
	if city=="" || countryCode=="" {
		c.Data["Error"]="Please select a city to see events."
		c.TplName="listing.tpl"
		return
	}
	svc:=services.NewEventService(getProvider())
	music,sports:=svc.GetListing(city,countryCode,scenario)

	// Both failed -> 502
	if music.Err!="" && sports.Err!="" {
		c.Ctx.Output.SetStatus(502)
		c.Data["Error"]= "Event providers are currently unavailable."
		c.TplName = "listing.tpl"
		return
	}
	c.Data["City"] = city
	c.Data["CountryCode"] = countryCode
	c.Data["Music"] = music
	c.Data["Sports"] = sports
	c.Ctx.Output.Header("X-Music-Cache", music.Cache)
	c.Ctx.Output.Header("X-Sports-Cache", sports.Cache)
	c.TplName = "listing.tpl"
}

func (c *EventsController) Details() {
	eventID:=c.Ctx.Input.Param(":eventId")
	if eventID==""{
		c.Ctx.Output.SetStatus(404)
		c.Data["Error"]="Event not found."
		c.TplName="error.tpl"
		return
	}
	p:=getProvider()
	ev,err:=p.GetEvent(eventID)
	if err!=nil {
		c.Ctx.Output.SetStatus(404)
		c.Data["Error"]="Event not found."
		c.TplName="error.tpl"
		return
	}

	// Determine if this is a mock ID
	isMock:=strings.HasPrefix(eventID, "mock-")
	c.Data["Event"]= ev
	c.Data["IsMock"] =isMock
	c.Data["ShowTickets"] =ev.TicketURL != "" || isMock
	c.TplName ="details.tpl"
}

var _ = models.Event{}