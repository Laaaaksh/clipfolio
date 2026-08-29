package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/laaaaksh/clipfolio/internal/auth"
	"github.com/laaaaksh/clipfolio/internal/db"
)

type contextKey string

const userContextKey contextKey = "user"

const sessionDuration = 30 * 24 * time.Hour

// handleSetupStatus tells the dashboard whether to show the setup form or
// the login form.
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check setup status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"needsSetup": count == 0})
}

// handleSetup creates the first admin account. It's only reachable while no
// user exists yet, and (when CLIPFOLIO_SETUP_TOKEN is set) requires that
// token, so a self-hoster's fresh deploy can't be claimed by a stranger who
// finds it before the operator does.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	count, err := s.store.CountUsers(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check existing users")
		return
	}
	if count > 0 {
		writeError(w, http.StatusForbidden, "setup already completed")
		return
	}

	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		SetupToken string `json:"setupToken"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if s.cfg.SetupToken != "" && req.SetupToken != s.cfg.SetupToken {
		writeError(w, http.StatusForbidden, "invalid setup token")
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "email is required and password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	user, err := s.store.CreateUser(ctx, strings.ToLower(req.Email), hash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	s.startSession(w, r, user.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"id": user.ID, "email": user.Email})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.store.GetUserByEmail(ctx, strings.ToLower(req.Email))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	s.startSession(w, r, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "email": user.Email})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil {
		_ = s.store.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "email": user.Email})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) {
	token, err := auth.NewSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start session")
		return
	}
	expiresAt := time.Now().Add(sessionDuration)
	if err := s.store.CreateSession(r.Context(), token, userID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.SessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}

		sess, err := s.store.GetSession(r.Context(), c.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "session expired or invalid")
			return
		}

		user, err := s.store.GetUserByID(r.Context(), sess.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "session expired or invalid")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userFromContext(ctx context.Context) db.User {
	u, _ := ctx.Value(userContextKey).(db.User)
	return u
}
