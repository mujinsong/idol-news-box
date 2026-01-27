package proxy

import (
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// Pool 代理池
type Pool struct {
	proxies   []string
	failed    map[string]int // 记录失败次数
	mu        sync.RWMutex
	maxFailed int // 最大失败次数，超过则移除
}

// NewPool 创建代理池
func NewPool(proxies []string) *Pool {
	return &Pool{
		proxies:   proxies,
		failed:    make(map[string]int),
		maxFailed: 3,
	}
}

// Get 随机获取一个代理
func (p *Pool) Get() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.proxies) == 0 {
		return ""
	}

	rand.Seed(time.Now().UnixNano())
	return p.proxies[rand.Intn(len(p.proxies))]
}

// MarkFailed 标记代理失败
func (p *Pool) MarkFailed(proxy string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.failed[proxy]++
	if p.failed[proxy] >= p.maxFailed {
		p.remove(proxy)
		logger.Warn.Printf("代理 %s 失败次数过多，已移除", proxy)
	}
}

// MarkSuccess 标记代理成功（重置失败计数）
func (p *Pool) MarkSuccess(proxy string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.failed, proxy)
}

// remove 移除代理
func (p *Pool) remove(proxy string) {
	for i, pr := range p.proxies {
		if pr == proxy {
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			delete(p.failed, proxy)
			break
		}
	}
}

// Size 返回代理池大小
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

// IsEmpty 检查代理池是否为空
func (p *Pool) IsEmpty() bool {
	return p.Size() == 0
}

// GetTransport 获取带代理的 http.Transport
func (p *Pool) GetTransport() *http.Transport {
	proxy := p.Get()
	if proxy == "" {
		return nil
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		logger.Error.Printf("解析代理URL失败: %s, %v", proxy, err)
		return nil
	}

	return &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
}
