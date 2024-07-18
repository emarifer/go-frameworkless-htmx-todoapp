package handlers

import (
	"encoding/base64"
	"net/http"
	"time"
)

// SetFlash sets a cookie with the flash message (base64 encoded)
// which is made available for the next request.
func SetFlash(w http.ResponseWriter, name string, value []byte) {
	c := &http.Cookie{Name: name, Value: encode(value)}

	http.SetCookie(w, c)
}

// getFlash tries to retrieve the message (as a []byte)
// set in a cookie by a handler in the previous request,
// decodes it, and overrides the cookie so that
// the message can only be read once.
func getFlash(
	w http.ResponseWriter, r *http.Request, name string,
) ([]byte, error) {
	c, err := r.Cookie(name)

	if err != nil {
		switch err {
		case http.ErrNoCookie:
			return nil, nil
		default:
			return nil, err
		}
	}

	value, err := decode(c.Value)
	if err != nil {
		return nil, err
	}

	dc := &http.Cookie{
		Name:    name,
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(1, 0),
	}

	http.SetCookie(w, dc)

	return value, nil
}

// GetMessages is a convenience function that uses the getFlash function,
// ignoring the errors it may return and transforming
// the byte slices of the error/success messages into strings.
// If the slices are <nil>, return their respective empty strings.
func GetMessages(w http.ResponseWriter, r *http.Request) (string, string) {
	fmErr, _ := getFlash(w, r, "error")
	fmSucc, _ := getFlash(w, r, "success")

	return string(fmErr), string(fmSucc)
}

// -------------------------

func encode(src []byte) string {
	return base64.URLEncoding.EncodeToString(src)
}

func decode(src string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(src)
}

/* SIMPLE FLASH MESSAGES IN GO:
https://www.alexedwards.net/blog/simple-flash-messages-in-golang
*/
