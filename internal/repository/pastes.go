package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pasteAPI/internal/repository/models"
	"pasteAPI/internal/service"
	"time"
)

type Pastes interface {
	Create(p *models.Paste) error
	Read(id uint16) (*models.Paste, error)
	ReadAll(title string, category uint8, filters service.Filters) ([]*models.Paste, *service.Metadata, error)
	Update(p *models.Paste) error
	Delete(id uint16) error
}

type PasteModel struct {
	DB *sql.DB
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

func (m *PasteModel) Read(id uint16) (*models.Paste, error) {
	if id == 0 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, title, category, text, created_at, expires_at, version 
		FROM pastes 
		WHERE id = $1 AND expires_at >= NOW()`

	var paste models.Paste

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&paste.Id,
		&paste.Title,
		&paste.Category,
		&paste.Text,
		&paste.CreatedAt,
		&paste.ExpiresAt,
		&paste.Version,
	)

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

func (m *PasteModel) ReadAll(title string, category uint8, filters service.Filters) ([]*models.Paste, *service.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) OVER(), id, title, category, text, created_at, expires_at, version 
		FROM pastes 
		WHERE expires_at >= NOW()
		AND ($1 = '' or (to_tsvector('english', title) @@ plainto_tsquery($1)) or (to_tsvector('russian', title) @@ plainto_tsquery($1)))
		AND (category = $2 or $2 = 0)
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, title, category, filters.Limit(), filters.Offset())
	if err != nil {
		return nil, &service.Metadata{}, err
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
		)
		if err != nil {
			return nil, &service.Metadata{}, err
		}

		pastes = append(pastes, &paste)
	}

	if err = rows.Err(); err != nil {
		return nil, &service.Metadata{}, err
	}

	metadata := service.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)

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
