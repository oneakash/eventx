package cache
import (
	"sync"
	"time"
	"eventx/models"
)
type entry struct{
	events []models.Event
	expiresAt time.Time
}

type Cache struct{
	mu   sync.Mutex
	items  map[string]entry
	ttl  time.Duration
}

var Default *Cache

func Init(ttl time.Duration){
	Default=&Cache{
		items:make(map[string]entry),
		ttl: ttl,
	}
}

// key is city|country|category|scenario
func (c *Cache) Get(key string) ([]models.Event, bool){
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok:=c.items[key]
	if !ok{
		return nil,false
	}
	if time.Now().After(e.expiresAt) {
		delete(c.items,key)
		return nil,false
	}
	return e.events,true
}

func (c *Cache)Set(key string,events []models.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key]=entry{
		events:events,
		expiresAt:time.Now().Add(c.ttl),
	}
}
func (c *Cache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items= make(map[string]entry)
}