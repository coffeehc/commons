package httpc

import (
	"context"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
	"net"
	"sync"
	"time"
)

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
	log.Debug("dns失效", zap.String("host", host))
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
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host) // 调用默认的resolver
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, nil
	}
	strIPs := make([]string, 0, len(ips))
	for _, ip := range ips {
		strIPs = append(strIPs, ip.String())
	}
	r.cache.Store(host, strIPs)
	return strIPs, nil
}

func (r *Resolver) autoRefresh(rate time.Duration) {
	for {
		time.Sleep(rate)
		r.Refresh()
	}
}
