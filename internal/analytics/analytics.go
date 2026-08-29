// Package analytics turns raw viewer-session watch data into the metrics
// clipfolio's dashboard shows: play rate, average watch percentage, and a
// drop-off (retention) curve.
package analytics

import "math"

// Point is one sample on a retention curve: at TimeSeconds into the video,
// RetentionFraction of played sessions were still watching (had reached at
// least that timestamp).
type Point struct {
	TimeSeconds       float64 `json:"timeSeconds"`
	RetentionFraction float64 `json:"retentionFraction"`
}

// Summary is the full analytics payload for one video.
type Summary struct {
	Impressions        int     `json:"impressions"`
	Plays              int     `json:"plays"`
	PlayRate           float64 `json:"playRate"`
	AvgWatchPercentage float64 `json:"avgWatchPercentage"`
	DropoffCurve       []Point `json:"dropoffCurve"`
}

// Session is the minimal shape analytics needs from a stored viewer session.
type Session struct {
	MaxTimeSeconds float64
	Played         bool
}

const defaultBuckets = 20

// Summarize computes a Summary from raw viewer sessions and the video's
// known duration. It never panics on empty input or a zero/unknown duration -
// self-hosted videos with a still-transcoding duration are a normal case, not
// an error.
func Summarize(sessions []Session, durationSeconds float64) Summary {
	s := Summary{DropoffCurve: dropoffCurve(sessions, durationSeconds, defaultBuckets)}

	s.Impressions = len(sessions)
	var watchedFractionSum float64
	for _, sess := range sessions {
		if sess.Played {
			s.Plays++
			watchedFractionSum += watchFraction(sess.MaxTimeSeconds, durationSeconds)
		}
	}
	if s.Impressions > 0 {
		s.PlayRate = float64(s.Plays) / float64(s.Impressions)
	}
	if s.Plays > 0 {
		s.AvgWatchPercentage = watchedFractionSum / float64(s.Plays)
	}
	return s
}

func watchFraction(maxTime, duration float64) float64 {
	if duration <= 0 {
		return 0
	}
	f := maxTime / duration
	return clamp01(f)
}

// dropoffCurve buckets played sessions into numBuckets even timestamps across
// the video's duration and reports, at each timestamp, the fraction of played
// sessions that reached at least that point - a monotonically non-increasing
// retention curve, the same shape Wistia's own drop-off chart shows.
func dropoffCurve(sessions []Session, duration float64, numBuckets int) []Point {
	if duration <= 0 || numBuckets <= 0 {
		return nil
	}

	var played []float64
	for _, sess := range sessions {
		if sess.Played {
			played = append(played, sess.MaxTimeSeconds)
		}
	}

	points := make([]Point, numBuckets+1)
	for i := 0; i <= numBuckets; i++ {
		t := duration * float64(i) / float64(numBuckets)
		points[i] = Point{TimeSeconds: round2(t), RetentionFraction: retentionAt(played, t)}
	}
	return points
}

func retentionAt(playedMaxTimes []float64, t float64) float64 {
	if len(playedMaxTimes) == 0 {
		return 0
	}
	reached := 0
	for _, maxTime := range playedMaxTimes {
		if maxTime >= t {
			reached++
		}
	}
	return float64(reached) / float64(len(playedMaxTimes))
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
