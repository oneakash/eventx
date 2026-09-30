package controllers
type HomeController struct {
	BaseController
}

func (c *HomeController) Get() {
	c.TplName= "home.tpl"
}