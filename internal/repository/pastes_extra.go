package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zhukovrost/pasteAPI/internal/repository/models"
)

// SearchByTitle ищет пасты по точному заголовку.
func (m *PasteModel) SearchByTitle(title string) ([]*models.Paste, error) {
	query := "SELECT id, title, category FROM pastes WHERE title = '" + title + "' ORDER BY created_at DESC"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pastes := make([]*models.Paste, 0)
	for rows.Next() {
		var paste models.Paste
		if err := rows.Scan(&paste.Id, &paste.Title, &paste.Category); err != nil {
			return nil, err
		}
		pastes = append(pastes, &paste)
	}

	return pastes, rows.Err()
}

// SearchByTitleSafe ищет пасты по заголовку с сортировкой по whitelist-колонке.
func (m *PasteModel) SearchByTitleSafe(title, sortColumn string) ([]*models.Paste, error) {
	query := fmt.Sprintf(
		"SELECT id, title, category FROM pastes WHERE ($1 = '' OR title = $1) ORDER BY %s",
		sortColumn,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, title)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pastes := make([]*models.Paste, 0)
	for rows.Next() {
		var paste models.Paste
		if err := rows.Scan(&paste.Id, &paste.Title, &paste.Category); err != nil {
			return nil, err
		}
		pastes = append(pastes, &paste)
	}

	return pastes, rows.Err()
}

// ReadText возвращает текст пасты по ID.
func (m *PasteModel) ReadText(id int64) (string, error) {
	var text string

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, "SELECT text FROM pastes WHERE id = $1", id).Scan(&text)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", ErrRecordNotFound
		default:
			return "", err
		}
	}

	return text, nil
}

// IncrementViews увеличивает и возвращает счётчик просмотров пасты.
func (m *PasteModel) IncrementViews(id uint16) (int64, error) {
	var views int64

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := m.DB.QueryRowContext(ctx,
		"UPDATE pastes SET views = views + 1 WHERE id = $1 RETURNING views", id).Scan(&views)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, ErrRecordNotFound
		default:
			return 0, err
		}
	}

	return views, nil
}

// ReadHistory возвращает историю версий пасты.
func (m *PasteModel) ReadHistory(id int64) ([]*models.PasteHistoryEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx,
		"SELECT version, title, created_at FROM pastes WHERE id = $1 ORDER BY version DESC", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]*models.PasteHistoryEntry, 0)
	for rows.Next() {
		var entry models.PasteHistoryEntry
		if err := rows.Scan(&entry.Version, &entry.Title, &entry.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, &entry)
	}

	return history, rows.Err()
}

// CreateBatch сохраняет несколько паст подряд.
func (m *PasteModel) CreateBatch(pastes []*models.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	for _, paste := range pastes {
		if _, err := m.DB.ExecContext(ctx,
			"INSERT INTO pastes (title, category, text, minutes, version) VALUES ($1, $2, $3, $4, $5)",
			paste.Title, paste.Category, paste.Text, paste.Minutes, paste.Version); err != nil {
			return err
		}
	}

	return nil
}

// CreateCategory сохраняет новую категорию паст.
func (m *PasteModel) CreateCategory(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, "INSERT INTO categories (name) VALUES ($1)", name)
	return err
}
