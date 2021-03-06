package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kubemq-io/kubemq-bridges/pkg/ratelimit"
	"github.com/kubemq-io/kubemq-bridges/pkg/ratelimit/internal/clock"
	"go.uber.org/atomic"

	"github.com/stretchr/testify/assert"
)

func TestUnlimited(t *testing.T) {
	now := time.Now()
	rl := ratelimit.NewUnlimited()
	for i := 0; i < 1000; i++ {
		rl.Take()
	}
	assert.Condition(t, func() bool { return time.Since(now) < 1*time.Millisecond }, "no artificial delay")
}

func TestRateLimiter(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	clock := clock.NewMock()
	rl := ratelimit.New(100, ratelimit.WithClock(clock), ratelimit.WithoutSlack)

	count := atomic.NewInt32(0)

	// Until we're done...
	done := make(chan struct{})
	defer close(done)

	// Create copious counts concurrently.
	go job(rl, count, done)
	go job(rl, count, done)
	go job(rl, count, done)
	go job(rl, count, done)

	clock.AfterFunc(1*time.Second, func() {
		assert.InDelta(t, 100, count.Load(), 10, "count within rate limit")
	})

	clock.AfterFunc(2*time.Second, func() {
		assert.InDelta(t, 200, count.Load(), 10, "count within rate limit")
	})

	clock.AfterFunc(3*time.Second, func() {
		assert.InDelta(t, 300, count.Load(), 10, "count within rate limit")
		wg.Done()
	})

	clock.Add(4 * time.Second)

	clock.Add(5 * time.Second)
}

func TestDelayedRateLimiter(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	clock := clock.NewMock()
	slow := ratelimit.New(10, ratelimit.WithClock(clock))
	fast := ratelimit.New(100, ratelimit.WithClock(clock))

	count := atomic.NewInt32(0)

	// Until we're done...
	done := make(chan struct{})
	defer close(done)

	// Run a slow job
	go func() {
		for {
			slow.Take()
			fast.Take()
			count.Inc()
			select {
			case <-done:
				return
			default:
			}
		}
	}()

	// Accumulate slack for 10 seconds,
	clock.AfterFunc(20*time.Second, func() {
		// Then start working.
		go job(fast, count, done)
		go job(fast, count, done)
		go job(fast, count, done)
		go job(fast, count, done)
	})

	clock.AfterFunc(30*time.Second, func() {
		assert.InDelta(t, 1200, count.Load(), 10, "count within rate limit")
		wg.Done()
	})

	clock.Add(40 * time.Second)
}

func job(rl ratelimit.Limiter, count *atomic.Int32, done <-chan struct{}) {
	for {
		rl.Take()
		count.Inc()
		select {
		case <-done:
			return
		default:
		}
	}
}
