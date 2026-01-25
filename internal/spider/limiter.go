package spider

import (
	"math/rand"
	"time"
)

// Limiter 频率控制器
type Limiter struct {
	minPages   int
	maxPages   int
	minSeconds int
	maxSeconds int
	pageCount  int
}

// NewLimiter 创建频率控制器
func NewLimiter(pages, seconds [2]int) *Limiter {
	return &Limiter{
		minPages:   pages[0],
		maxPages:   pages[1],
		minSeconds: seconds[0],
		maxSeconds: seconds[1],
	}
}

// Wait 根据页数决定是否等待
func (r *Limiter) Wait() {
	r.pageCount++
	threshold := r.minPages + rand.Intn(r.maxPages-r.minPages+1)
	if r.pageCount >= threshold {
		seconds := r.minSeconds + rand.Intn(r.maxSeconds-r.minSeconds+1)
		time.Sleep(time.Duration(seconds) * time.Second)
		r.pageCount = 0
	}
}
