package api

import (
	"context"

	"github.com/laaaaksh/clipfolio/internal/analytics"
	"github.com/laaaaksh/clipfolio/internal/db"
)

func (s *Server) computeAnalytics(ctx context.Context, video db.Video) (analytics.Summary, error) {
	sessions, err := s.store.ViewerSessionsForVideo(ctx, video.ID)
	if err != nil {
		return analytics.Summary{}, err
	}

	analyticsSessions := make([]analytics.Session, len(sessions))
	for i, sess := range sessions {
		analyticsSessions[i] = analytics.Session{MaxTimeSeconds: sess.MaxTimeSeconds, Played: sess.Played}
	}

	return analytics.Summarize(analyticsSessions, video.DurationSeconds), nil
}
