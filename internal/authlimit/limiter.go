// Package authlimit bounds authentication traffic and expensive password work.
package authlimit

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrLimited = errors.New("too many authentication attempts; please try again later")

type LimitError struct{ RetryAfter time.Duration }

func (e *LimitError) Error() string { return ErrLimited.Error() }
func (e *LimitError) Unwrap() error { return ErrLimited }

type Config struct {
	IPLimit       int
	AccountLimit  int
	Window        time.Duration
	MaxConcurrent int
	MaxEntries    int
}

type bucket struct {
	count int
	until time.Time
}

type Limiter struct {
	mu          sync.Mutex
	buckets     map[[32]byte]bucket
	nextCleanup time.Time
	cfg         Config
	slots       chan struct{}
	now         func() time.Time
}

func New(cfg Config) *Limiter {
	if cfg.IPLimit <= 0 {
		cfg.IPLimit = 30
	}
	if cfg.AccountLimit <= 0 {
		cfg.AccountLimit = 5
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 2
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 20000
	}
	return &Limiter{cfg: cfg, buckets: make(map[[32]byte]bucket), slots: make(chan struct{}, cfg.MaxConcurrent), now: time.Now}
}

// AllowIP and AllowAccount count every attempt, including successful ones. Once
// exhausted, a bucket stays blocked until its window ends; retries do not extend it.
func (l *Limiter) AllowIP(ip string) time.Duration {
	return l.allow("ip:"+ip, l.cfg.IPLimit)
}

func (l *Limiter) AllowAccount(account string) time.Duration {
	return l.allow("account:"+strings.ToLower(strings.TrimSpace(account)), l.cfg.AccountLimit)
}

func (l *Limiter) allow(key string, limit int) time.Duration {
	keyHash := sha256.Sum256([]byte(key))
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if !now.Before(l.nextCleanup) {
		for key, b := range l.buckets {
			if !now.Before(b.until) {
				delete(l.buckets, key)
			}
		}
		l.nextCleanup = now.Add(l.cfg.Window)
	}
	b, exists := l.buckets[keyHash]
	if !exists || !now.Before(b.until) {
		// Never evict active limits to accommodate attacker-controlled new keys.
		if !exists && len(l.buckets) >= l.cfg.MaxEntries {
			return l.nextCleanup.Sub(now)
		}
		b = bucket{until: now.Add(l.cfg.Window)}
	}
	if b.count >= limit {
		return b.until.Sub(now)
	}
	b.count++
	l.buckets[keyHash] = b
	return 0
}

// Acquire rejects immediately when full. There is no unbounded wait queue.
// The caller must hold the slot until password work has actually finished,
// even if its request context is canceled while Argon2 is running.
func (l *Limiter) Acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case l.slots <- struct{}{}:
		return func() { <-l.slots }, nil
	default:
		return nil, &LimitError{RetryAfter: time.Second}
	}
}
