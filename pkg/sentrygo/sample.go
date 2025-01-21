package sentrygo

import (
	"math/rand"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

var rng *randomGenerator

func init() {
	rng = newRandomGenerator()
}

type randomGenerator struct {
	rand *rand.Rand
	mu   sync.Mutex
}

func newRandomGenerator() *randomGenerator {
	return &randomGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Float64 returns, as a float64, a pseudo-random number in the half-open interval [0.0,1.0).
func (g *randomGenerator) Float64() (f float64) {
	g.mu.Lock()
	f = float64(g.rand.Int63n(1<<53)) / (1 << 53) // ref: https://cs.opensource.google/go/go/+/refs/tags/go1.19.4:src/math/rand/rand.go;l=179
	g.mu.Unlock()

	return
}

const (
	debugSampleRate   = 0.02
	infoSampleRate    = 0.05
	warningSampleRate = 0.10
)

func sampleEventByLevel(level sentry.Level) (shouldKeep bool) {
	var chance float64
	switch level {
	case sentry.LevelDebug:
		chance = debugSampleRate
	case sentry.LevelInfo:
		chance = infoSampleRate
	case sentry.LevelWarning:
		chance = warningSampleRate
	case sentry.LevelFatal, sentry.LevelError:
		// never drops severe errors
		return true
	default:
		// unexpected level, keep all events for investigation
		return true
	}

	return chance > rng.Float64()
}
