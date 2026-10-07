package authlimit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimitsNormalizeAccountsAndExpireWithoutExtendingLockout(t *testing.T) {
	now := time.Unix(100, 0)
	l := New(Config{IPLimit: 2, AccountLimit: 2, Window: time.Minute})
	l.now = func() time.Time { return now }
	for _, account := range []string{" Person@Example.com ", "person@example.com"} {
		if retry := l.AllowAccount(account); retry != 0 {
			t.Fatalf("early throttle: %s", retry)
		}
	}
	now = now.Add(10 * time.Second)
	if retry := l.AllowAccount("PERSON@example.com"); retry != 50*time.Second {
		t.Fatalf("retry = %s", retry)
	}
	if retry := l.AllowAccount("other@example.com"); retry != 0 {
		t.Fatalf("unrelated account throttled: %s", retry)
	}
	for range 2 {
		if retry := l.AllowIP("192.0.2.1"); retry != 0 {
			t.Fatal(retry)
		}
	}
	if l.AllowIP("192.0.2.1") == 0 {
		t.Fatal("IP limit not enforced")
	}
	if l.AllowIP("192.0.2.2") != 0 {
		t.Fatal("unrelated IP throttled")
	}
	now = now.Add(50 * time.Second)
	if l.AllowAccount("person@example.com") != 0 {
		t.Fatal("blocked retry extended the window")
	}
}

func TestStorageCapFailsClosedAndRecovers(t *testing.T) {
	now := time.Unix(100, 0)
	l := New(Config{MaxEntries: 2, AccountLimit: 1, Window: time.Minute})
	l.now = func() time.Time { return now }
	l.AllowAccount("a")
	l.AllowAccount("b")
	if l.AllowAccount("c") == 0 || len(l.buckets) != 2 {
		t.Fatal("storage cap bypassed")
	}
	if l.AllowAccount("a") == 0 {
		t.Fatal("active limit evicted")
	}
	now = now.Add(time.Minute)
	if l.AllowAccount("c") != 0 || len(l.buckets) != 1 {
		t.Fatal("expired entries not reclaimed")
	}
}

func TestConcurrentAdmissionAndSlotRelease(t *testing.T) {
	l := New(Config{AccountLimit: 3, MaxConcurrent: 2})
	var admitted atomic.Int32
	var wg sync.WaitGroup
	done := make(chan struct{})
	ready := make(chan struct{}, 32)
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := l.Acquire(context.Background())
			if err == nil {
				admitted.Add(1)
			}
			ready <- struct{}{}
			if err == nil {
				<-done
				release()
			} else if !errors.Is(err, ErrLimited) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	for range 32 {
		<-ready
	}
	if admitted.Load() != 2 {
		t.Errorf("admitted = %d, want 2", admitted.Load())
	}
	close(done)
	wg.Wait()
	release, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatal("slot leaked", err)
	}
	release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request admitted: %v", err)
	}
}

func TestConcurrentAccountLimit(t *testing.T) {
	l := New(Config{AccountLimit: 5})
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.AllowAccount("same") == 0 {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 5 {
		t.Fatalf("allowed = %d, want 5", allowed.Load())
	}
}
