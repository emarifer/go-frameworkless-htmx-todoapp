package services

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

type UserService struct {
	User      User
	UserStore *sql.DB
}

func NewUserService(u User, uStore *sql.DB) *UserService {

	return &UserService{
		User:      u,
		UserStore: uStore,
	}
}

func (us *UserService) CreateUser(u User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), 8)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users(email, password, username) VALUES($1, $2, $3)`

	_, err = us.UserStore.Exec(
		stmt,
		u.Email,
		string(hashedPassword),
		u.Username,
	)

	return err
}

func (us *UserService) CheckEmail(email string) (User, error) {

	query := `SELECT id, email, password, username FROM users
		WHERE email = ?`

	stmt, err := us.UserStore.Prepare(query)
	if err != nil {
		return User{}, err
	}

	defer stmt.Close()

	us.User.Email = email
	err = stmt.QueryRow(
		us.User.Email,
	).Scan(
		&us.User.ID,
		&us.User.Email,
		&us.User.Password,
		&us.User.Username,
	)
	if err != nil {
		return User{}, err
	}

	return us.User, nil
}
