package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	jwtoken "github.com/emarifer/go-frameworkless-htmx/internal/utils/jwt"

	"github.com/golang-jwt/jwt/v5"
)

// Middleware is a definition of what a middleware is,
// take in one type Handler interface and
// wrap it within another Handler interface
type Middleware func(http.Handler) http.Handler

// CreateStack chains the middlewares that
// we pass to it in a slice wrapping the handle
// that we pass as the first parameter.
// This function allows us, therefore, to add as many middlewares
// as we want (included in a slice) that wrap our handler
// without the code becoming cumbersome to read.
func CreateStack(xs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(xs) - 1; i >= 0; i-- {
			x := xs[i]
			next = x(next)
		}

		return next
	}
}

// AuthMiddleware is a handler that verifies if the token
// exists (in a cookie) and if it is invalid (due to the
// signature not being verified or having expired).
// If the jsonwebtoken is valid, it extracts the user data
// and injects it with the context of the request
// that will be passed to the next handler in the chain.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Excludes everything other than this from the middleware action.
		// It also refers to the routes corresponding to the assets.
		p := r.URL.Path
		if p != "/todo" && p != "/create" && p != "/edit" && p != "/logout" {
			next.ServeHTTP(w, r)
			return
		}

		// Get the JWT(cookie) by name
		cookie, err := r.Cookie("jwt")
		if err != nil {
			fm := []byte("You are not authorized")
			SetFlash(w, "error", fm)

			http.Redirect(w, r, "/login", http.StatusSeeOther)

			return
		}

		claims := &jwtoken.AuthClaims{}

		// Parse the cookie & check for errors
		token, err := jwt.ParseWithClaims(
			cookie.Value,
			claims,
			func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtoken.SecretKey), nil
			},
		)
		if err != nil {
			fm := []byte("You are not authorized")
			SetFlash(w, "error", fm)

			http.Redirect(w, r, "/login", http.StatusSeeOther)

			return
		}

		// Parse the custom claims & check jwt is valid
		_, ok := token.Claims.(*jwtoken.AuthClaims)
		if !ok || !token.Valid {
			fm := []byte("You are not authorized")
			SetFlash(w, "error", fm)

			http.Redirect(w, r, "/login", http.StatusSeeOther)

			return
		}

		// We inject the user data from the token into the context.
		u := UserData{
			ID:       claims.Id,
			Username: claims.Username,
			Tzone:    claims.Tzone,
		}

		ctx := withRequestUserData(r.Context(), u)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FlagMiddleware is middleware for unprotected routes
// that manages a boolean flag (fromProtected) for
// conditional rendering on the pages corresponding to said routes.
// Checks if the user is authenticated: if it is, inject
// the fromProtected flag into the context to true, false otherwise.
// Basically, this flag is intended to prevent
// an authenticated user from logging in/registering again.
func FlagMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the JWT(cookie) by name
		cookie, err := r.Cookie("jwt")
		if err != nil {
			// If the user is not authenticated,
			// we inject the value of `fromProtected` as false into the context.
			ctx := withRequestFromProtected(r.Context(), false)

			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		// Parse the cookie & check for errors
		token, err := jwt.ParseWithClaims(
			cookie.Value,
			&jwtoken.AuthClaims{},
			func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtoken.SecretKey), nil
			},
		)
		if err != nil {
			// If the user is not authenticated,
			// we inject the value of `fromProtected` as false into the context.
			ctx := withRequestFromProtected(r.Context(), false)

			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		// Parse the custom claims & check jwt is valid
		_, ok := token.Claims.(*jwtoken.AuthClaims)
		if !ok || !token.Valid {
			// If the user is not authenticated,
			// we inject the value of `fromProtected` as false into the context.
			ctx := withRequestFromProtected(r.Context(), false)

			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		// We inject the value of `fromProtected` as true into the context.
		ctx := withRequestFromProtected(r.Context(), true)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logging is a structure to support the `LoggingMiddleware` middleware
// and be able to pass it (as a method receiver)
// the `*slog.Logger` pointer without altering
// the middleware signature [func(http.Handler) http.Handler].
type logging struct {
	l *slog.Logger
}

func NewLogging(l *slog.Logger) *logging {
	return &logging{l}
}

// wrappedWriter is a wrapper on top of `ResponseWriter`
// with an additional field to store
// the `statusCode` so it can be retrieved later.
type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

// LoggingMiddleware is the middleware that wraps the others
// and collects all the info/error sent by the handlers
// and traverses the middleware stack to log it.
func (lg *logging) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default status
		}

		next.ServeHTTP(wrapped, r)

		handler := ""
		errMsg := ""
		if h, ok := wrapped.Header()[HEADER_KEY_HANDLER]; ok {
			handler = h[0]
		}
		if e, ok := wrapped.Header()[HEADER_KEY_ERRMSG]; ok {
			errMsg = e[0]
		}

		dataLog := []any{
			"host", r.Host,
			"latency", fmt.Sprintf(
				"%.2fμs", float64(time.Since(start).Nanoseconds())/1000,
			),
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode, // ResponseWriter with statusCode
			"user_agent", r.Header.Get("User-Agent"),
		}
		if errMsg != "" {
			dataLog = append(dataLog, "error", errMsg, "handler", handler)
			lg.l.Error("🔴 Handler Error", dataLog...)
			return
		}

		if handler == "" {
			lg.l.Info("📂 Assets Info", dataLog...)
			return
		}

		dataLog = append(dataLog, "handler", handler)
		lg.l.Info("🔵 Handler Info", dataLog...)
	})
}
