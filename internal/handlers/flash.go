package handlers

import (
	"encoding/base64"
	"net/http"
	"time"
)

func SetFlash(w http.ResponseWriter, name string, value []byte) {
	c := &http.Cookie{Name: name, Value: encode(value)}

	http.SetCookie(w, c)
}

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
