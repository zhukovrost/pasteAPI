package models

import (
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"time"
)

var AnonymousUser = &User{}

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Login     string    `json:"login"`
	Email     string    `json:"email"`
	Password  Password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

func ValidateEmail(v *validator.MyValidator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.MyValidator, password string) {
	v.Check(password != "", "Password", "must be provided")
	v.Check(len(password) >= 8, "Password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "Password", "must not be more than 72 bytes long")
}

func ValidateLogin(v *validator.MyValidator, login string) {
	v.Check(login != "", "login", "must be provided")
	v.Check(validator.Matches(login, validator.LoginRX), "login", "contains incorrect symbols")
}

func ValidateUser(v *validator.MyValidator, user *User) {
	ValidateLogin(v, user.Login)
	ValidateEmail(v, user.Email)

	if user.Password.Plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.Plaintext)
	}

	if user.Password.Hash == nil {
		panic("missing Password hash for user")
	}
}
