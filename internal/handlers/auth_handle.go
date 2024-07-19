package handlers

import (
	"errors"
	"fmt"
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

func NewAuthHandle(us AuthService) *AuthHandle {
	return &AuthHandle{userService: us}
}

type AuthHandle struct {
	userService AuthService
}

func (ah *AuthHandle) homeHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
	errMsg, succMsg := GetMessages(w, r)

	data := map[string]any{
		"title":         "",
		"fromProtected": requestFromProtected(r.Context()),
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}
	return asCaller(), tmpl.ExecuteTemplate(w, "home.tmpl", data)
}

func (ah *AuthHandle) registerHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
	errMsg, succMsg := GetMessages(w, r)

	data := map[string]any{
		"title":         "| Register",
		"fromProtected": requestFromProtected(r.Context()),
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}
	return asCaller(), tmpl.ExecuteTemplate(w, "register.tmpl", data)
}

func (ah *AuthHandle) registerPostHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
	email := strings.Trim(r.FormValue("email"), " ")
	password := strings.Trim(r.FormValue("password"), " ")
	username := strings.Trim(r.FormValue("username"), " ")

	// Simple server-side validation...
	if email == "" || password == "" || username == "" {
		fm := []byte("Fields cannot be empty")
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/register", http.StatusSeeOther)

		return asCaller(), nil
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
			return asCaller(), apiError{
				status:  http.StatusInternalServerError,
				message: "error 500: database temporarily out of service",
			}
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			err = errors.New("the email is already in use")
		}
		fm := []byte(err.Error())
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/register", http.StatusSeeOther)

		return asCaller(), nil
	}

	fm := []byte("You have successfully registered!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/login", http.StatusSeeOther)

	return asCaller(), nil
}

func (ah *AuthHandle) loginHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
	errMsg, succMsg := GetMessages(w, r)

	data := map[string]any{
		"title":         "| Login",
		"fromProtected": requestFromProtected(r.Context()),
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}
	return asCaller(), tmpl.ExecuteTemplate(w, "login.tmpl", data)
}

func (ah *AuthHandle) loginPostHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
	email := strings.Trim(r.FormValue("email"), " ")
	password := strings.Trim(r.FormValue("password"), " ")
	tzone := r.Header.Get("X-Timezone")

	// Simple server-side validation...
	if email == "" || password == "" {
		fm := []byte("Fields cannot be empty")
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/login", http.StatusSeeOther)

		return asCaller(), nil
	}

	// Authentication goes here
	user, err := ah.userService.CheckEmail(email)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			return asCaller(), apiError{
				status:  http.StatusInternalServerError,
				message: "error 500: database temporarily out of service",
			}
		}
		if strings.Contains(err.Error(), "no rows in result set") {
			// In production you have to give the user a generic message
			err = errors.New("there is no user with that email")
		}
		fm := []byte(err.Error())
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/login", http.StatusSeeOther)

		return asCaller(), nil
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

		return asCaller(), nil
	}

	// Create JWT
	signedToken, err := jwt.CreateNewAuthToken(user.ID, user.Username, tzone)
	if err != nil {
		return asCaller(), apiError{
			status:  http.StatusInternalServerError,
			message: fmt.Sprintf("error 500: could not get the JWT: %s", err),
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

	return asCaller(), nil
}

func (ah *AuthHandle) logoutHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
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

	return asCaller(), nil
}
