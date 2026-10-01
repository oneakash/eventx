package routers

import (
	"eventx/controllers"
	"github.com/beego/beego/v2/server/web"
	beego "github.com/beego/beego/v2/server/web"
)

func init() {
    beego.Router("/", &controllers.HomeController{})
	// Frontend SSR routes
	web.Router("/events", &controllers.EventsController{}, "get:List")
	web.Router("/events/:eventId", &controllers.EventsController{}, "get:Details")

	// Ticket redirect (backend action)
	web.Router("/redirect/:eventId", &controllers.RedirectController{}, "get:Redirect")


	// JSON APIs
	web.Router("/api/locations/autocomplete", &controllers.LocationController{}, "get:Autocomplete")
	web.Router("/api/locations/:placeId", &controllers.LocationController{}, "get:Resolve")

	web.Router("/api/cache/invalidate", &controllers.CacheController{}, "post:Invalidate")
	// Health
	web.Router("/healthz", &controllers.BaseController{}, "get:Health")

}
