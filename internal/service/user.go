package service

import (
	"pasteAPI/internal/repository/models"
	"pasteAPI/pkg/validator"
)

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateLogin(v *validator.Validator, login string) {
	v.Check(login != "", "login", "must be provided")
	v.Check(validator.Matches(login, validator.LoginRX), "login", "contains incorrect symbols")
}

func ValidateUser(v *validator.Validator, user *models.User) {
	ValidateLogin(v, user.Login)
	ValidateEmail(v, user.Email)

	if user.Password.Plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.Plaintext)
	}

	if user.Password.Hash == nil {
		panic("missing password hash for user")
	}
}
