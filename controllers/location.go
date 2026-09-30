package controllers
import (
	"regexp"
	"eventx/models"
)
var sessionTokenRe =regexp.MustCompile(`^[A-Za-z0-9_-]{1,36}$`)
type LocationController struct {
	BaseController
}
func (c *LocationController)Autocomplete(){
	input :=c.GetString("input")
	token :=c.GetString("sessionToken")
	if len([]byte(input))>200{
		c.Ctx.Output.SetStatus(400)
		c.Data["json"]=models.ErrorResponse{Error:"Input too long."}
		c.ServeJSON()
		return
	}
	if token!="" && !sessionTokenRe.MatchString(token) {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"]= models.ErrorResponse{Error:"Invalid session token."}
		c.ServeJSON()
		return
	}
	p:=getProvider()
	suggestions,err:=p.Autocomplete(input,token)
	if err!=nil{
		c.Ctx.Output.SetStatus(502)
		c.Data["json"] = models.ErrorResponse{Error: "Location provider failed."}
		c.ServeJSON()
		return
	}
	if suggestions ==nil {
		suggestions =[]models.Suggestion{}
	}
	c.Data["json"]=models.AutocompleteResponse{Suggestions:suggestions}
	c.ServeJSON()
}

func (c *LocationController)Resolve(){
	placeID:=c.Ctx.Input.Param(":placeId")
	token:=c.GetString("sessionToken")
	if placeID=="" || token =="" {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = models.ErrorResponse{Error: "Missing place ID or session token."}
		c.ServeJSON()
		return
	}
	if !sessionTokenRe.MatchString(token) {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] =models.ErrorResponse{Error:"Invalid session token."}
		c.ServeJSON()
		return
	}
	p :=getProvider()
	place,err:= p.Resolve(placeID,token)
	if err !=nil{
		c.Ctx.Output.SetStatus(502)
		c.Data["json"]=models.ErrorResponse{Error: "Location lookup failed."}
		c.ServeJSON()
		return
	}
	c.Data["json"]=place
	c.ServeJSON()
}