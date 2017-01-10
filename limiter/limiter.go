package limiter

import (
	"cache"
	"sync"
	"time"
)

const (
	defExpireSecond = 300
)

type secondRate struct {
	sec      int64
	rate     int //用于计算的
	baserate int //原始的rate
}

type limiter struct {
	sync.RWMutex
	m map[string]secondRate
}

func (l *limiter) add(k string, rate int) {
	l.Lock()
	if _, ok := l.m[k]; !ok {
		sr := secondRate{
			sec:      time.Now().Unix(),
			rate:     rate,
			baserate: rate,
		}
		l.m[k] = sr
	}
	l.Unlock()
}

//存在的才更新，并且只更新baserate，在下一秒的时候才会自动生效
func (l *limiter) upd(k string, rate int) {
	l.Lock()
	if v, ok := l.m[k]; ok {
		v.baserate = rate
		l.m[k] = v
	}
	l.Unlock()
}

func (l *limiter) get(k string) (secondRate, bool) {
	l.RLock()
	defer l.RUnlock()

	v, ok := l.m[k]
	return v, ok
}

func (l *limiter) del(k string) {
	l.Lock()
	delete(l.m, k)
	l.Unlock()
}

func (l *limiter) exist(k string) bool {
	l.RLock()
	defer l.RUnlock()
	_, ok := l.m[k]
	return ok
}

type RateLimiter struct {
	rls   *limiter // 每秒内允许请求的最大次数
	cache *cache.Cache
}

func newlimiter() *limiter {
	return &limiter{
		m: make(map[string]secondRate),
	}
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		rls:   newlimiter(),
		cache: cache.NewCache(defExpireSecond),
	}
}

//add k element just when the k is not existed
func (l *RateLimiter) AddElement(k string, rate int) {
	l.rls.add(k, rate)
}

//update rate
func (l *RateLimiter) UpdElement(k string, rate int) {
	l.rls.upd(k, rate)
}

//check k element exist
func (l *RateLimiter) ExistElement(k string) bool {
	return l.rls.exist(k)
}

//delete
func (l *RateLimiter) DelElemnt(k string) {
	l.rls.del(k)
	l.cache.Del(k)
}

func (l *RateLimiter) Limit(k string) bool {
	ok := l.cache.Exist(k)
	sr, e := l.rls.get(k)
	if !e { //不存在限速
		if ok { //已经删除了限速
			l.cache.Del(k)
		}
		return true
	}
	if !ok { //首次设置
		l.cache.Set(k, sr)
		return true
	}

	cb := func(exist bool, newValue interface{}, oldValue interface{}) (interface{}, bool) {
		nowSec := time.Now().Unix()
		sr := oldValue.(secondRate)

		flag := true
		if nowSec == sr.sec { //在同一秒内
			if sr.rate > 0 {
				sr.rate -= 1
			} else {
				sr.rate = 0
				flag = false
			}
		} else {
			sr.sec = nowSec
			if exist {
				raw := newValue.(secondRate)
				sr.rate = raw.baserate
			} else {
				sr.rate = sr.baserate
			}
		}

		return sr, flag
	}

	return l.cache.UpdateAtomic(k, e, sr, cb)
}

func (l *RateLimiter) Close() {
	l.cache.Close()
}
