package controllers

import (
	"strings"

	"eventx/services"

	"github.com/beego/beego/v2/server/web"
)

type RedirectController struct {
	BaseController
}
func (c *RedirectController) Redirect() {
	eventID := c.Ctx.Input.Param(":eventId")
	if eventID == "" {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Missing event ID."}
		c.ServeJSON()
		return
	}

	p := getProvider()
	ev, err := p.GetEvent(eventID)
	if err != nil {
		c.Ctx.Output.SetStatus(404)
		c.Data["json"] = map[string]string{"error": "Event not found."}
		c.ServeJSON()
		return
	}

	hostsStr, _ := web.AppConfig.String("APPROVED_TICKET_HOSTS")
	var approved []string
	for _, h := range strings.Split(hostsStr, ",") {
		h = strings.TrimSpace(h)
		if h != "" {
			approved = append(approved, h)
		}
	}

	target, err := services.ValidateTicketURL(ev.TicketURL, approved)
	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Ticket destination is not allowed."}
		c.ServeJSON()
		return
	}

	c.Ctx.Redirect(302, target)
}