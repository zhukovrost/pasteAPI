package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"pasteAPI/internal/repository/models"
	"strings"
	"time"
)

var (
	ErrDuplicate = errors.New("duplicate email or login")
)

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Create(user *models.User) error {
	query := `
		INSERT INTO users (login, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, activated, version`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, user.Login, user.Email, user.Password.Hash).Scan(&user.ID, &user.CreatedAt, &user.Activated, &user.Version)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), `pq: duplicate key value`):
			return ErrDuplicate
		default:
			return err
		}
	}

	return nil
}

func (m *UserModel) GetByEmail(email string) (*models.User, error) {
	query := `
        SELECT id, created_at, login, email, password_hash, activated, version
        FROM users
		WHERE email = $1`

	var user models.User

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Login,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m *UserModel) Update(user *models.User) error {
	query := `
	UPDATE users
	SET login = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
	WHERE id = $5 AND version = $6
	RETURNING version`

	args := []interface{}{
		user.Login,
		user.Email,
		user.Password.Hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), `pq: duplicate key value`):
			return ErrDuplicate
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (m *UserModel) GetForToken(tokenScope, tokenPlaintext string) (*models.User, error) {
	query := `
		SELECT users.id, users.created_at, users.login, users.email, users.password_hash, users.activated, users.version
        FROM users
		INNER JOIN tokens 
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1 
		AND tokens.scope = $2
		AND tokens.expiry > NOW()`

	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, tokenHash[:], tokenScope).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Login,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
