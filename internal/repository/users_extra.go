package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ReadPasswordHash возвращает сохранённый хеш пароля пользователя.
func (m *UserModel) ReadPasswordHash(login string) (string, error) {
	var hash string

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx,
		"SELECT password_hash FROM users WHERE login = $1", login).Scan(&hash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", ErrUserNotFound
		default:
			return "", err
		}
	}

	return hash, nil
}
