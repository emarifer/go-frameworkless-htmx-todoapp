package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// openssl rand -base64 32 (command)
// https://www.tecmint.com/generate-pre-shared-key-in-linux/
const SecretKey = "R51f5xI/WBGF3yiR78mXKwgnINQLAco1A65qLdMxNOI="

type AuthClaims struct {
	Id                   int    `json:"id"`
	Username             string `json:"username"`
	Tzone                string `json:"tzone"`
	jwt.RegisteredClaims `json:"claims"`
}

func CreateNewAuthToken(id int, username, tz string) (string, error) {
	claims := AuthClaims{
		Id:       id,
		Username: username,
		Tzone:    tz,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    "https://github.com/emarifer",
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	signedToken, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", fmt.Errorf("error signing the token: %s", err)
	}

	return signedToken, nil
}
