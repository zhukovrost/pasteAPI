package repository

import (
	"context"
	"database/sql"
	"time"
)

type PermissionModel struct {
	DB *sql.DB
}

func (m *PermissionModel) SetWritePermission(userId int64, pasteId uint16) error {
	query := `
        INSERT INTO write_permissions (user_id, paste_id)
        VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userId, pasteId)
	return err
}

func (m *PermissionModel) GetWritePermission(userId int64, pasteId uint16) (bool, error) {
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
