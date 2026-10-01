package controllers

import(
	"eventx/cache"
)

type CacheController struct {
	BaseController
}

func (c *CacheController) Invalidate() {
	if !c.authorized() {
		c.Ctx.Output.SetStatus(403)
		c.Data["json"] = map[string]any{
			"error": "Forbidden.",
		}
		c.ServeJSON()
		return
	}

	if cache.Default == nil {
		c.Ctx.Output.SetStatus(503)
		c.Data["json"] = map[string]any{
			"error": "Cache not initialized.",
		}
		c.ServeJSON()
		return
	}

	key := c.GetString("key")
	prefix := c.GetString("prefix")

	switch {
	case key != "":
		ok := cache.Default.Invalidate(key)
		c.Data["json"] = map[string]any{
			"scope":       "key",
			"key":         key,
			"invalidated": boolToInt(ok),
		}
	case prefix != "":
		n := cache.Default.InvalidateByPrefix(prefix)
		c.Data["json"] = map[string]any{
			"scope":       "prefix",
			"prefix":      prefix,
			"invalidated": n,
		}
	default:
		n := cache.Default.Reset()
		c.Data["json"] = map[string]any{
			"scope":       "all",
			"invalidated": n,
		}
	}
	c.ServeJSON()
}
func (c *CacheController) authorized() bool {
		return true
}
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}