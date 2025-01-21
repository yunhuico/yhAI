package sentrygo

import (
	"math"
	"testing"

	"github.com/getsentry/sentry-go"
)

func Test_randomGenerator_Float64(t *testing.T) {
	generator := newRandomGenerator()

	for i := 0; i < 5000; i++ {
		got := generator.Float64()
		if got < 0 || got >= 1 {
			t.Fatalf("unexpected random number %f", got)
		}
	}
}

func Test_sampleEventByLevel(t *testing.T) {
	// Law of large numbers
	const (
		eps   = 0.01
		round = 100000
	)

	tests := []struct {
		level          sentry.Level
		wantSampleRate float64
	}{
		{
			level:          "unexpected",
			wantSampleRate: 1,
		},
		{
			level:          sentry.LevelDebug,
			wantSampleRate: debugSampleRate,
		},
		{
			level:          sentry.LevelInfo,
			wantSampleRate: infoSampleRate,
		},
		{
			level:          sentry.LevelWarning,
			wantSampleRate: warningSampleRate,
		},
		{
			level:          sentry.LevelError,
			wantSampleRate: 1,
		},
		{
			level:          sentry.LevelFatal,
			wantSampleRate: 1,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			var sampled int
			for i := 0; i < round; i++ {
				if sampleEventByLevel(tt.level) {
					sampled++
				}
			}

			gotSampleRate := float64(sampled) / round
			if math.Abs(tt.wantSampleRate-gotSampleRate) > eps {
				t.Fatalf("level %s want sample rate %f, got %f", tt.level, tt.wantSampleRate, gotSampleRate)
			}

			t.Logf("level %s, want sample rate %f, got %f", tt.level, tt.wantSampleRate, gotSampleRate)
		})
	}
}
