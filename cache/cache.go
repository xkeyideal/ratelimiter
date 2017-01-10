package cache

import (
	"container/list"
	"sync"
	"time"
)

type Item struct {
	Key   string
	Value interface{}
	t     time.Time
}

type Cache struct {
	lock       *sync.Mutex
	l          *list.List
	items      map[string]*list.Element
	expireTime time.Duration //element expire time
}

const (
	CHECKINTERVAL = 120
)

var nowFunc = time.Now

func NewCache(expire uint32) *Cache {
	c := &Cache{
		lock:       &sync.Mutex{},
		l:          list.New(),
		items:      make(map[string]*list.Element),
		expireTime: time.Duration(expire) * time.Second,
	}

	go c.clear()

	return c
}

func (c *Cache) clear() {
	for {
		c.removeExpired()
		time.Sleep(CHECKINTERVAL * time.Second)
	}
}

//front to back -> older time element to newer time element
func (c *Cache) removeExpired() {
	c.lock.Lock()
	for c.l.Len() != 0 {
		e := c.l.Front() //取最老的时间
		if e == nil {
			break
		}
		item := e.Value.(*Item)
		if item.t.Add(c.expireTime).After(nowFunc()) {
			break
		}
		k := item.Key
		c.l.Remove(e)
		delete(c.items, k)
	}
	c.lock.Unlock()
}

/**
  最新的数据一律放到链表的尾部，因此链表从首至尾时间是从小到大的
*/

func (c *Cache) Set(k string, v interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.items[k] = c.l.PushBack(&Item{Key: k, Value: v, t: nowFunc()})
}

func (c *Cache) Exist(k string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.items[k]
	return ok
}

func (c *Cache) Get(k string) (interface{}, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[k]; ok {
		v.Value.(*Item).t = nowFunc()
		c.l.MoveToBack(v)
		return v.Value.(*Item).Value, true
	}

	return nil, false
}

func (c *Cache) Del(k string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[k]; ok {
		c.l.Remove(v)
		delete(c.items, k)
	}
}

type UpdateAtomicCb func(exist bool, newValue interface{}, oldValue interface{}) (interface{}, bool)

//change value atomic
func (c *Cache) UpdateAtomic(k string, e bool, value interface{}, cb UpdateAtomicCb) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[k]; ok {
		newValue, ok := cb(e, value, v.Value.(*Item).Value)
		c.l.Remove(v)
		c.items[k] = c.l.PushBack(&Item{Key: k, Value: newValue, t: nowFunc()})
		return ok
	}
	return true
}

func (c *Cache) Update(k string, vv interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[k]; ok {
		c.l.Remove(v)
		c.items[k] = c.l.PushBack(&Item{Key: k, Value: vv, t: nowFunc()})
	}
}

func (c *Cache) Close() {
	c.lock.Lock()
	c.l.Init()
	c.items = make(map[string]*list.Element, 0)
	c.lock.Unlock()
}
