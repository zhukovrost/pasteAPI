package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
	"time"
)

type PasteModel struct {
	DB postgres.Database
}

// === CRUD OPERATIONS ===

func (m *PasteModel) Create(p *models.Paste) error {
	query := `
		INSERT INTO pastes (title, category, text, expires_at)
		VALUES (TRIM($1), $2, TRIM($3), NOW() + interval '1 minute' * $4)
		RETURNING id, created_at, expires_at`

	args := []interface{}{p.Title, p.Category, p.Text, p.Minutes}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&p.Id, &p.CreatedAt, &p.ExpiresAt)
}

func (m *PasteModel) Read(pasteId uint16, user *models.User) (*models.Paste, error) {
	if pasteId == 0 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT 
			p.id, p.title, p.category, p.text, p.created_at, p.expires_at, p.version,
			COALESCE((SELECT true FROM write_permissions wp WHERE wp.paste_id = p.id AND wp.user_id = $2), false) as can_edit
		FROM pastes p
		WHERE p.id = $1 AND p.expires_at >= NOW()`

	var paste models.Paste

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, pasteId, user.ID).Scan(
		&paste.Id,
		&paste.Title,
		&paste.Category,
		&paste.Text,
		&paste.CreatedAt,
		&paste.ExpiresAt,
		&paste.Version,
		&paste.CanEdit,
	)

	if !user.Activated {
		paste.CanEdit = false
	}

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &paste, nil
}

func (m *PasteModel) ReadAll(title string, category uint8, user *models.User, filters models.Filters) ([]*models.Paste, *models.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) OVER(), p.id, p.title, p.category, p.text, p.created_at, p.expires_at, p.version,
			COALESCE((SELECT true FROM write_permissions wp WHERE wp.paste_id = p.id AND wp.user_id = $5), false) as can_edit
		FROM pastes p
		WHERE p.expires_at >= NOW()
		AND ($1 = '' OR (to_tsvector('english', p.title) @@ plainto_tsquery($1)) OR (to_tsvector('russian', p.title) @@ plainto_tsquery($1)))
		AND (p.category = $2 OR $2 = 0)
		ORDER BY %s %s, p.id ASC
		LIMIT $3 OFFSET $4`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, title, category, filters.Limit(), filters.Offset(), user.ID)
	if err != nil {
		return nil, &models.Metadata{}, err
	}

	defer rows.Close()

	pastes := make([]*models.Paste, 0)
	var totalRecords uint32

	for rows.Next() {
		var paste models.Paste

		err := rows.Scan(
			&totalRecords,
			&paste.Id,
			&paste.Title,
			&paste.Category,
			&paste.Text,
			&paste.CreatedAt,
			&paste.ExpiresAt,
			&paste.Version,
			&paste.CanEdit,
		)

		if !user.Activated {
			paste.CanEdit = false
		}

		if err != nil {
			return nil, &models.Metadata{}, err
		}

		pastes = append(pastes, &paste)
	}

	if err = rows.Err(); err != nil {
		return nil, &models.Metadata{}, err
	}

	metadata := models.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return pastes, &metadata, nil
}

func (m *PasteModel) Update(p *models.Paste) error {
	query := `
        UPDATE pastes
        SET title = TRIM($1), category = $2, text = TRIM($3), expires_at = expires_at + interval '1 minute' * $4, version = version + 1
        WHERE id = $5 AND expires_at >= NOW() AND version=$6
        RETURNING version`

	args := []interface{}{
		p.Title,
		p.Category,
		p.Text,
		p.Minutes,
		p.Id,
		p.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&p.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (m *PasteModel) Delete(id uint16) error {
	query := `
		DELETE FROM pastes
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rws, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rws == 0 {
		return ErrRecordNotFound
	}

	return nil
}

/*
type MockPasteModel struct{}

func (m MockPasteModel) Create(Paste *Paste) error {
	return nil
}
func (m MockPasteModel) Read(id uint8) (*Paste, error) {
	return nil, nil
}
func (m MockPasteModel) Update(Paste *Paste) error {
	return nil
}
func (m MockPasteModel) Delete(id uint8) error {
	return nil
}
*/
