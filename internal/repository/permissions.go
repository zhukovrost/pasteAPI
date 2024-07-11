package repository

import (
	"context"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
	"strings"
	"time"
)

type PermissionModel struct {
	DB postgres.Database
}

func (m *PermissionModel) SetWritePermissionByLogin(userLogin string, pasteId uint16) (int64, error) {
	query := `
        INSERT INTO write_permissions (user_id, paste_id)
		VALUES (
			(SELECT id FROM users WHERE login = $1),
			$2
		)
		RETURNING user_id;`

	var userId int64

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userLogin, pasteId).Scan(&userId)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), "pq: null value in column"):
			return 0, ErrUserNotFound
		default:
			return 0, err
		}
	}

	return userId, nil
}

func (m *PermissionModel) SetWritePermissionById(userId int64, pasteId uint16) error {
	query := `
        INSERT INTO write_permissions (user_id, paste_id)
        VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userId, pasteId)
	return err
}

func (m *PermissionModel) CheckWritePermission(userId int64, pasteId uint16) (bool, error) {
	query := `
		SELECT EXISTS (
            SELECT 1
            FROM write_permissions
            WHERE user_id = $1 AND paste_id = $2
        )`

	var exists bool

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userId, pasteId).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
