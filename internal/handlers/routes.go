package handlers

import (
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	HEADER_KEY_HANDLER = "X-Handler"
	HEADER_KEY_ERRMSG  = "X-Errmsg"
)

var tmpl *template.Template

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string {
	return e.message
}

// Use as a wrapper around the handler functions.
// Basically, they use the Adapter design pattern.
type adapterHandle func(http.ResponseWriter, *http.Request) error

// adapterHandle implements the http.Handler interface.
// Because handlers return errors, the ServeHTTP implementation
// handles errors based on their type. Therefore, this function
// also performs centralized error handling in addition
// to successfully logging the handlers' output.
func (a adapterHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := a(w, r)
	if err == nil {
		return
	}

	var e apiError
	if errors.As(err, &e) {
		caller := ""
		if h, ok := w.Header()[HEADER_KEY_HANDLER]; ok {
			caller = h[0]
		}

		data := map[string]any{
			"isError":       true,
			"fromProtected": requestFromProtected(r.Context()),
		}

		// We handle the case that occurs when a logged in user cannot
		// connect to the database table `todos` through these handlers
		// when an error occurs with code 500: the user is
		// automatically logged out and therefore the `fromProtected` flag
		// is set to FALSE. In your application
		// you can handle this situation as you see fit.
		if (strings.Contains(caller, "todoListHandle") ||
			strings.Contains(caller, "createTodoPostHandle") ||
			strings.Contains(caller, "editTodoHandle") ||
			strings.Contains(caller, "editTodoPostHandle") ||
			strings.Contains(caller, "deleteTodoHandle")) &&
			e.status == 500 {
			data["fromProtected"] = false
		}

		switch e.status {
		case 400:
			data["title"] = "| Error 400"
			tmpl.ExecuteTemplate(w, "error_400.tmpl", data)
			return
		case 404:
			data["title"] = "| Error 404"
			tmpl.ExecuteTemplate(w, "error_404.tmpl", data)
			return
		case 500:
			data["title"] = "| Error 500"
			tmpl.ExecuteTemplate(w, "error_500.tmpl", data)
			return
		}
	}

	// In case the error returned by the handler is not of
	// type apiError (it has to be an error while
	// rendering the template), we return a JSON response with a code 500.
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "failure",
		"message": "Unknown server error",
		"code":    http.StatusInternalServerError,
	})
}

// asCaller is a convenience function that gets the caller
// from which this function is called.
// The index `1` returns the name of the handler we are looking for.
func asCaller() string {
	pc, _, _, _ := runtime.Caller(1) // index = 1
	hs := strings.Split(runtime.FuncForPC(pc).Name(), "/")

	return hs[len(hs)-1]
}

// clearCookie is a convenience function that deletes
// the cookie containing the authentication token.
func clearCookie(w http.ResponseWriter) {
	dc := &http.Cookie{
		Name:    "jwt",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(1, 0),
	}
	http.SetCookie(w, dc)
}

// LoadRoutes starts the `tmpl` variable,
// necessary to execute the various templates that
// the handlers will execute, while registering
// the routes of the various endpoints.
func LoadRoutes(
	l *slog.Logger, r *http.ServeMux, ah *AuthHandle, th *TodoHandle,
) {
	if tmpl == nil {
		tmpl = template.Must(tmpl.ParseGlob("views/*.tmpl"))
	}

	// global middleware stack
	s := CreateStack(
		NewLogging(l).LoggingMiddleware,
		FlagMiddleware,
		AuthMiddleware,
	)
	// "/{$}" only matches the slash
	r.Handle("GET /{$}", s(adapterHandle(ah.homeHandle)))
	r.Handle("GET /register", s(adapterHandle(ah.registerHandle)))
	r.Handle("POST /register", s(adapterHandle(ah.registerPostHandle)))
	r.Handle("GET /login", s(adapterHandle(ah.loginHandle)))
	r.Handle("POST /login", s(adapterHandle(ah.loginPostHandle)))
	r.Handle("POST /logout", s(adapterHandle(ah.logoutHandle)))

	r.Handle("GET /todo", s(adapterHandle(th.todoListHandle)))
	r.Handle("GET /create", s(adapterHandle(th.createTodoHandle)))
	r.Handle("POST /create", s(adapterHandle(th.createTodoPostHandle)))
	r.Handle("GET /edit", s(adapterHandle(th.editTodoHandle)))
	r.Handle("POST /edit", s(adapterHandle(th.editTodoPostHandle)))
	r.Handle("DELETE /delete", s(adapterHandle(th.deleteTodoHandle)))

	// middleware stack without AuthMiddleware
	nfs := CreateStack(
		NewLogging(l).LoggingMiddleware,
		FlagMiddleware,
	)
	// "/" matches anything
	r.Handle("/", nfs(adapterHandle(notFoundHandle)))
}
