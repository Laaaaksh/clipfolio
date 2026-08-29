package analytics

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestSummarize_EmptySessions(t *testing.T) {
	s := Summarize(nil, 100)
	if s.Impressions != 0 || s.Plays != 0 {
		t.Fatalf("expected zero impressions/plays, got %+v", s)
	}
	if s.PlayRate != 0 || s.AvgWatchPercentage != 0 {
		t.Fatalf("expected zero rates on no data, got %+v", s)
	}
	if len(s.DropoffCurve) != defaultBuckets+1 {
		t.Fatalf("expected %d dropoff points even with no sessions, got %d", defaultBuckets+1, len(s.DropoffCurve))
	}
	for _, p := range s.DropoffCurve {
		if p.RetentionFraction != 0 {
			t.Fatalf("expected 0 retention with no played sessions, got %+v", p)
		}
	}
}

func TestSummarize_ZeroDuration(t *testing.T) {
	// A video still transcoding (duration not yet known) must not divide by zero.
	s := Summarize([]Session{{MaxTimeSeconds: 5, Played: true}}, 0)
	if s.AvgWatchPercentage != 0 {
		t.Fatalf("expected 0 avg watch percentage on zero duration, got %v", s.AvgWatchPercentage)
	}
	if s.DropoffCurve != nil {
		t.Fatalf("expected nil dropoff curve on zero duration, got %+v", s.DropoffCurve)
	}
}

func TestSummarize_PlayRate(t *testing.T) {
	sessions := []Session{
		{Played: true, MaxTimeSeconds: 10},
		{Played: true, MaxTimeSeconds: 20},
		{Played: false, MaxTimeSeconds: 0}, // impression only, never hit play
		{Played: false, MaxTimeSeconds: 0},
	}
	s := Summarize(sessions, 100)
	if s.Impressions != 4 {
		t.Fatalf("expected 4 impressions, got %d", s.Impressions)
	}
	if s.Plays != 2 {
		t.Fatalf("expected 2 plays, got %d", s.Plays)
	}
	if !almostEqual(s.PlayRate, 0.5) {
		t.Fatalf("expected play rate 0.5, got %v", s.PlayRate)
	}
}

func TestSummarize_AvgWatchPercentage(t *testing.T) {
	// Two viewers of a 100s video: one watched to 25s, one to 75s -> avg 50%.
	sessions := []Session{
		{Played: true, MaxTimeSeconds: 25},
		{Played: true, MaxTimeSeconds: 75},
	}
	s := Summarize(sessions, 100)
	if !almostEqual(s.AvgWatchPercentage, 0.5) {
		t.Fatalf("expected avg watch percentage 0.5, got %v", s.AvgWatchPercentage)
	}
}

func TestSummarize_WatchFractionClampedAtOne(t *testing.T) {
	// A viewer whose reported max time exceeds duration (e.g. a rounding
	// heartbeat landing a fraction of a second past "ended") must not produce
	// watch percentages above 100%.
	sessions := []Session{{Played: true, MaxTimeSeconds: 105}}
	s := Summarize(sessions, 100)
	if !almostEqual(s.AvgWatchPercentage, 1.0) {
		t.Fatalf("expected watch percentage clamped to 1.0, got %v", s.AvgWatchPercentage)
	}
}

func TestSummarize_DropoffCurveMonotonicallyNonIncreasing(t *testing.T) {
	sessions := []Session{
		{Played: true, MaxTimeSeconds: 10},
		{Played: true, MaxTimeSeconds: 40},
		{Played: true, MaxTimeSeconds: 90},
		{Played: true, MaxTimeSeconds: 100},
	}
	s := Summarize(sessions, 100)
	for i := 1; i < len(s.DropoffCurve); i++ {
		if s.DropoffCurve[i].RetentionFraction > s.DropoffCurve[i-1].RetentionFraction+1e-9 {
			t.Fatalf("dropoff curve increased at index %d: %+v -> %+v", i, s.DropoffCurve[i-1], s.DropoffCurve[i])
		}
	}
	// At t=0 every played session is still "retained".
	if !almostEqual(s.DropoffCurve[0].RetentionFraction, 1.0) {
		t.Fatalf("expected 100%% retention at t=0, got %v", s.DropoffCurve[0].RetentionFraction)
	}
	// At t=duration only the one session that reached the very end (100) remains.
	last := s.DropoffCurve[len(s.DropoffCurve)-1]
	if !almostEqual(last.RetentionFraction, 0.25) {
		t.Fatalf("expected 25%% retention at video end, got %v", last.RetentionFraction)
	}
}

func TestSummarize_UnplayedSessionsExcludedFromDropoffAndWatchPercentage(t *testing.T) {
	sessions := []Session{
		{Played: false, MaxTimeSeconds: 999}, // should never happen, but must not pollute stats
		{Played: true, MaxTimeSeconds: 50},
	}
	s := Summarize(sessions, 100)
	if !almostEqual(s.AvgWatchPercentage, 0.5) {
		t.Fatalf("expected only the played session to count, got avg %v", s.AvgWatchPercentage)
	}
	if !almostEqual(s.DropoffCurve[0].RetentionFraction, 1.0) {
		t.Fatalf("expected dropoff curve to only consider played sessions")
	}
}
