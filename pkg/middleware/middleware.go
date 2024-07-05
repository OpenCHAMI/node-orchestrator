package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func AuthenticatorWithRequiredClaims(ja *jwtauth.JWTAuth, requiredClaims []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, claims, err := jwtauth.FromContext(r.Context())

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token == nil || jwt.Validate(token, ja.ValidateOptions()...) != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			for _, claim := range requiredClaims {
				if _, ok := claims[claim]; !ok {
					err := fmt.Errorf("missing required claim %s", claim)
					log.Error().Err(err).Msg("Missing required claim")
					http.Error(w, "missing required claim", http.StatusUnauthorized)
					return
				}
			}

			// Token is authenticated and all required claims are present, pass it through
			next.ServeHTTP(w, r)
		})
	}
}

const LoggerKey = "logger"

// OpenCHAMILogger is a chi middleware that adds a sublogger to the context.
func OpenCHAMILogger(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sublogger := logger.With().
				Str("request_id", middleware.GetReqID(r.Context())).
				Str("request_uri", r.RequestURI).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Str("method", r.Method).
				Logger()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			ctx := context.WithValue(r.Context(), LoggerKey, &sublogger)

			// Use the modified context with the sublogger
			r = r.WithContext(ctx)

			defer func() {
				duration := time.Since(start)
				// Extract the sublogger from the context again
				sublogger := r.Context().Value(LoggerKey).(*zerolog.Logger)
				sublogger.Info().
					Str("status", http.StatusText(ww.Status())).
					Int("status_code", ww.Status()).
					Int64("bytes_in", r.ContentLength).
					Int("bytes_out", ww.BytesWritten()).
					Dur("duration", duration).
					Msg("Request")
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
