package handlers

import (
	"log"
	"net/http"
	"time"

	jwtoken "github.com/emarifer/go-frameworkless-htmx/internal/utils/jwt"

	"github.com/golang-jwt/jwt/v5"
)

// authMiddleware is a handler that verifies if the token
// exists (in a cookie) and if it is invalid (due to the
// signature not being verified or having expired).
// If the jsonwebtoken is valid, it extracts the user data
// and injects it with the context of the request
// that will be passed to the next handler in the chain.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// flagMiddleware is middleware for unprotected routes
// that manages a boolean flag (fromProtected) for
// conditional rendering on the pages corresponding to said routes.
// Checks if the user is authenticated: if it is, inject
// the fromProtected flag into the context to true, false otherwise.
// Basically, this flag is intended to prevent
// an authenticated user from logging in/registering again.
func flagMiddleware(next http.Handler) http.Handler {
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

func LatencyLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("latency_human: %v", time.Since(start))
	})
}
