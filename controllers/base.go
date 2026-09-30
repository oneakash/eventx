package controllers
import (
	"github.com/beego/beego/v2/server/web"
)

type BaseController struct {
	web.Controller
}
func (c *BaseController) Health() {
	c.Data["json"] = map[string]string{
		"status":  "ok",
		"runtime": "Beego",
	}
	c.ServeJSON()
}