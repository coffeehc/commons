package redisservice

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"

	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var luaRefresh = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`)
var luaRelease = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)
var (
	ErrLockUnlockFailed     = errors.New("lock unlock failed")
	ErrLockNotObtained      = errors.New("lock not obtained")
	ErrLockDurationExceeded = errors.New("lock duration exceeded")
)

type Locker interface {
	Lock(ctx context.Context) (bool, error)
	LockWithContext(ctx context.Context) (bool, error)
	Unlock(ctx context.Context) error
}

type LockOption struct {
	RetryCount  int64
	RetryDelay  time.Duration
	LockTimeout time.Duration
}

var DefaultLockOption = &LockOption{
	RetryCount:  3,
	RetryDelay:  time.Second * 2,
	LockTimeout: time.Second * 15,
}

type lockerImpl struct {
	redisService Service
	key          string
	opt          *LockOption
	token        string
	mutex        sync.Mutex
}

// NewLocker creates a new distributed locker on a given key.
func NewLocker(redisService Service, key string, opt *LockOption) Locker {
	if opt == nil {
		opt = DefaultLockOption
	}
	return &lockerImpl{
		redisService: redisService,
		key:          key,
		opt:          opt,
	}
}

// Run runs a callback handler with a Redis lock. It may return ErrLockNotObtained
// if a lock was not successfully acquired.
func Run(ctx context.Context, redisService Service, key string, opt *LockOption, handler func()) error {
	locker, err := Obtain(ctx, redisService, key, opt)
	if err != nil {
		return err
	}

	sem := make(chan struct{})
	go func() {
		handler()
		close(sem)
	}()

	select {
	case <-sem:
		return locker.Unlock(ctx)
	case <-time.After(opt.LockTimeout):
		return ErrLockDurationExceeded
	}
}

// Obtain is a shortcut for NewLocker().Lock(). It may return ErrLockNotObtained
// if a lock was not successfully acquired.
func Obtain(ctx context.Context, redisService Service, key string, opt *LockOption) (Locker, error) {
	locker := NewLocker(redisService, key, opt)
	if ok, err := locker.Lock(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrLockNotObtained
	}
	return locker, nil
}

// IsLocked returns true if a lock is still being held.
func (l *lockerImpl) IsLocked() bool {
	l.mutex.Lock()
	locked := l.token != ""
	l.mutex.Unlock()

	return locked
}

// Lock applies the lock, don't forget to defer the Unlock() function to release the lock after usage.
func (l *lockerImpl) Lock(ctx context.Context) (bool, error) {
	return l.LockWithContext(ctx)
}

// LockWithContext is like Lock but allows to pass an additional context which allows cancelling
// lock attempts prematurely.
func (l *lockerImpl) LockWithContext(ctx context.Context) (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token != "" {
		return l.refresh(ctx)
	}
	return l.create(ctx)
}

// Unlock releases the lock
func (l *lockerImpl) Unlock(ctx context.Context) error {
	l.mutex.Lock()
	err := l.release(ctx)
	l.mutex.Unlock()

	return err
}

// Helpers

func (l *lockerImpl) create(ctx context.Context) (bool, error) {
	l.reset()

	// Create a random token
	token, err := randomToken()
	if err != nil {
		return false, err
	}

	// Calculate the timestamp we are willing to wait for
	attempts := l.opt.RetryCount + 1
	var retryDelay *time.Timer

	for {

		// Try to obtain a lock
		ok, err := l.obtain(ctx, token)
		if err != nil {
			return false, err
		} else if ok {
			l.token = token
			return true, nil
		}

		if attempts--; attempts <= 0 {
			return false, nil
		}

		if retryDelay == nil {
			retryDelay = time.NewTimer(l.opt.RetryDelay)
			defer retryDelay.Stop()
		} else {
			retryDelay.Reset(l.opt.RetryDelay)
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-retryDelay.C:
		}
	}
}

func (l *lockerImpl) refresh(ctx context.Context) (bool, error) {
	ttl := strconv.FormatInt(int64(l.opt.LockTimeout/time.Millisecond), 10)
	status, err := luaRefresh.Run(ctx, l.redisService, []string{l.key}, l.token, ttl).Result()
	if err != nil {
		return false, err
	} else if status == int64(1) {
		return true, nil
	}
	return l.create(ctx)
}

func (l *lockerImpl) obtain(ctx context.Context, token string) (bool, error) {
	ok, err := l.redisService.SetNX(ctx, l.key, token, l.opt.LockTimeout).Result()
	if err == redis.Nil {
		err = nil
	}
	return ok, err
}

func (l *lockerImpl) release(ctx context.Context) error {
	defer l.reset()

	res, err := luaRelease.Run(ctx, l.redisService, []string{l.key}, l.token).Result()
	if err == redis.Nil {
		return ErrLockUnlockFailed
	}

	if i, ok := res.(int64); !ok || i != 1 {
		return ErrLockUnlockFailed
	}

	return err
}

func (l *lockerImpl) reset() {
	l.token = ""
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
