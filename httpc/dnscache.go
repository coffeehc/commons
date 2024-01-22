package httpc

import (
	"context"
	"fmt"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
	"net"
	"strings"
	"sync"
	"time"
)

var DefaultResolver = &net.Resolver{}

func init() {
	DefaultResolver.PreferGo = true
}

type Resolver struct {
	cache           sync.Map
	ResolverTimeout time.Duration
	lock            sync.Mutex
}

func NewResolver(cacheTimes, refreshRate time.Duration) *Resolver {
	resolver := &Resolver{
		ResolverTimeout: cacheTimes,
	}
	if refreshRate > 0 {
		go resolver.autoRefresh(refreshRate)
	}
	return resolver
}

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

func (r *Resolver) Refresh() {
	addresses := make([]string, 0)
	r.cache.Range(func(key, value interface{}) bool {
		addresses = append(addresses, key.(string))
		return true
	})
	for _, host := range addresses {
		ctx, _ := context.WithTimeout(context.Background(), r.ResolverTimeout)
		r.Lookup(ctx, host)
	}
}

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

func (r *Resolver) autoRefresh(rate time.Duration) {
	for {
		time.Sleep(rate)
		r.Refresh()
	}
}
