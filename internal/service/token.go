package service

import (
	"github.com/zhukovrost/pasteAPI/pkg/validator"
)

func ValidateTokenPlaintext(v *validator.MyValidator, plaintext string) {
	v.Check(plaintext != "", "token", "token must be provided")
	v.Check(len(plaintext) == 26, "token", "token must be 26 bytes long")
}
