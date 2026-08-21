package httpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

// DefaultResolver 是 DNS 查询使用的 Go resolver。
var DefaultResolver = &net.Resolver{}

func init() {
	DefaultResolver.PreferGo = true
}

// Resolver 提供带定时刷新的进程内 DNS 缓存。
type Resolver struct {
	// cache 按 host 保存最近一次成功解析的地址列表。
	cache sync.Map
	// ResolverTimeout 是自动刷新单个 host 的最大解析时长。
	ResolverTimeout time.Duration
	// lock 串行化同一时刻的缓存未命中解析。
	lock sync.Mutex
}

// NewResolver 创建 DNS 缓存；refreshRate 大于零时启动定时刷新。
func NewResolver(cacheTimes, refreshRate time.Duration) *Resolver {
	resolver := &Resolver{
		ResolverTimeout: cacheTimes,
	}
	if refreshRate > 0 {
		go resolver.autoRefresh(refreshRate)
	}
	return resolver
}

// Get 返回 host 的缓存地址，未命中时同步解析并写入缓存。
func (r *Resolver) Get(ctx context.Context, host string) ([]string, error) {
	value, loaded := r.cache.Load(host)
	if loaded {
		return value.([]string), nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	value, loaded = r.cache.Load(host)
	if loaded {
		return value.([]string), nil
	}
	return r.Lookup(ctx, host)
}

// Refresh 在 ResolverTimeout 限制内刷新当前缓存的全部 host。
func (r *Resolver) Refresh() {
	addresses := make([]string, 0)
	r.cache.Range(func(key, value interface{}) bool {
		addresses = append(addresses, key.(string))
		return true
	})
	for _, host := range addresses {
		ctx, cancel := context.WithTimeout(context.Background(), r.ResolverTimeout)
		_, _ = r.Lookup(ctx, host)
		cancel()
	}
}

// Lookup 解析 host，并只在成功获得地址时更新缓存。
func (r *Resolver) Lookup(ctx context.Context, host string) ([]string, error) {
	//log.Debug("查询dns", zap.String("host", host))
	//ips, err := net.DefaultResolver.LookupIPAddr(ctx, host) // 调用默认的resolver
	ips, err := DefaultResolver.LookupHost(ctx, host) // 调用默认的resolver
	if err != nil {
		log.Error("错误", zap.Error(err))
		return nil, err
	}
	if len(ips) == 0 {
		log.Error("没有获取到任何对应的ip", zap.String("host", host))
		return nil, nil
	}
	for i, ip := range ips {
		if strings.Contains(ip, ":") {
			ips[i] = fmt.Sprintf("[%s]", ip)
		}
	}
	r.cache.Store(host, ips)
	return ips, nil
}

// autoRefresh 按固定间隔刷新已经缓存的 host。
func (r *Resolver) autoRefresh(rate time.Duration) {
	for {
		time.Sleep(rate)
		r.Refresh()
	}
}
