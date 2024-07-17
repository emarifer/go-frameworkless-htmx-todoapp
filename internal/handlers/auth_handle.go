package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/emarifer/go-frameworkless-htmx/internal/services"
	"github.com/emarifer/go-frameworkless-htmx/internal/utils/jwt"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	CreateUser(u services.User) error
	CheckEmail(email string) (services.User, error)
}

func NewAuthHandle(l *slog.Logger, us AuthService) *AuthHandle {

	return &AuthHandle{
		logger:      l,
		userService: us,
	}
}

type AuthHandle struct {
	logger      *slog.Logger
	userService AuthService
}

func (ah *AuthHandle) homeHandle(w http.ResponseWriter, r *http.Request) error {
	errMsg, succMsg := GetMessages(w, r)

	data := map[string]any{
		"title":         "",
		"fromProtected": requestFromProtected(r.Context()),
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}

	return renderView(
		w, r, asCaller(), ah.logger, "home.tmpl", data,
	)
}

func (ah *AuthHandle) registerHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	errMsg, succMsg := GetMessages(w, r)

	data := map[string]any{
		"title":         "| Register",
		"fromProtected": requestFromProtected(r.Context()),
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}

	return renderView(
		w, r, asCaller(), ah.logger, "register.tmpl", data,
	)
}

func (ah *AuthHandle) registerPostHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	email := strings.Trim(r.FormValue("email"), " ")
	password := strings.Trim(r.FormValue("password"), " ")
	username := strings.Trim(r.FormValue("username"), " ")

	if email == "" || password == "" || username == "" {
		fm := []byte("Fields cannot be empty")
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/register", http.StatusSeeOther)

		return nil

		// TODO: Example error response.
		// It is necessary to handle the rendering
		// with htmx/response-targets extension
		// which will affect the entire body.
		/* return apiError{
			message: "error 500: something went wrong",
			status:  http.StatusInternalServerError,
			handler: asCaller(),
			logger:  ah.logger,
		} */
	}

	user := services.User{
		Email:    email,
		Password: password,
		Username: username,
	}

	if err := ah.userService.CreateUser(user); err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  ah.logger,
			}
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			err = errors.New("the email is already in use")
		}
		fm := []byte(err.Error())
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/register", http.StatusSeeOther)

		return nil
	}

	fm := []byte("You have successfully registered!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/login", http.StatusSeeOther)

	return nil
}

func (ah *AuthHandle) loginHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	errMsg, succMsg := GetMessages(w, r)

	data := map[string]any{
		"title":         "| Login",
		"fromProtected": requestFromProtected(r.Context()),
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}

	return renderView(
		w, r, asCaller(), ah.logger, "login.tmpl", data,
	)
}

func (ah *AuthHandle) loginPostHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	email := strings.Trim(r.FormValue("email"), " ")
	password := strings.Trim(r.FormValue("password"), " ")
	tzone := r.Header.Get("X-Timezone")

	if email == "" || password == "" {
		fm := []byte("Fields cannot be empty")
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/login", http.StatusSeeOther)

		return nil
	}

	// Authentication goes here
	user, err := ah.userService.CheckEmail(email)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  ah.logger,
			}
		}
		if strings.Contains(err.Error(), "no rows in result set") {
			// In production you have to give the user a generic message
			err = errors.New("there is no user with that email")
		}
		fm := []byte(err.Error())
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/login", http.StatusSeeOther)

		return nil
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	)
	if err != nil {
		// In production you have to give the user a generic message
		fm := []byte("Incorrect password")
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/login", http.StatusSeeOther)

		return nil
	}

	// Create JWT
	signedToken, err := jwt.CreateNewAuthToken(user.ID, user.Username, tzone)
	if err != nil {
		return apiError{
			message: fmt.Sprintf("error 500: could not get the JWT: %s", err),
			status:  http.StatusInternalServerError,
			handler: asCaller(),
			logger:  ah.logger,
		}
	}

	// Create and set the cookie
	cookie := http.Cookie{
		Name:     "jwt",
		Value:    signedToken,
		Expires:  time.Now().Add(1 * time.Hour),
		Path:     "/",
		HttpOnly: true, // meant only for the server
	}
	http.SetCookie(w, &cookie)

	fm := []byte("You have successfully logged in!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/todo", http.StatusSeeOther)

	return nil
}

func (ah *AuthHandle) logoutHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	dc := &http.Cookie{
		Name:    "jwt",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(1, 0),
	}

	http.SetCookie(w, dc)

	fm := []byte("You have successfully logged out!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/login", http.StatusSeeOther)

	return nil
}
