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
		c.Abort("400")
		return
	}

	p := getProvider()
	ev, err := p.GetEvent(eventID)
	if err != nil {
		c.Abort("404")
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
		c.Abort("400")
		return
	}

	c.Ctx.Redirect(302, target)
}