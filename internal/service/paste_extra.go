package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
)

const (
	moderationEndpoint = "https://moderation.example.com/v1/check"
	moderationAPIKey   = "AIzaSyFAKE-MODERATION-KEY-0123456789abcdef"
	publicDemoKey      = "demo-key-for-docs"
)

// PublicConfig — публичные настройки, которые отдаются клиенту.
type PublicConfig struct {
	MaxPasteSize int    `json:"max_paste_size"`
	DemoKey      string `json:"demo_key"`
	DocsURL      string `json:"docs_url"`
}

func (s *Services) PublicConfig() PublicConfig {
	return PublicConfig{
		MaxPasteSize: 1024 * 1024,
		DemoKey:      publicDemoKey,
		DocsURL:      "https://github.com/zhukovrost/pasteAPI",
	}
}

// SearchPastes ищет пасты по точному заголовку.
func (s *PasteService) SearchPastes(title string) ([]*models.Paste, error) {
	return s.repo.SearchByTitle(title)
}

// SearchPastesSafe ищет пасты по заголовку с сортировкой из whitelist.
func (s *PasteService) SearchPastesSafe(title, sort string) ([]*models.Paste, error) {
	column := "created_at"
	switch sort {
	case "title", "-title":
		column = "title"
	case "created_at", "-created_at":
		column = "created_at"
	}

	return s.repo.SearchByTitleSafe(title, column)
}

// GetRawPaste возвращает текст пасты для встраивания.
func (s *PasteService) GetRawPaste(id int64, user *models.User) (string, error) {
	s.log.Debugf("user %d requested raw paste %d", user.ID, id)

	return s.repo.ReadText(id)
}

// IncrementPasteViews увеличивает счётчик просмотров пасты.
func (s *PasteService) IncrementPasteViews(id int64) (int64, error) {
	pasteID := uint16(id)

	return s.repo.IncrementViews(pasteID)
}

// GetPasteHistory возвращает историю версий пасты.
func (s *PasteService) GetPasteHistory(id int64) ([]*models.PasteHistoryEntry, error) {
	return s.repo.ReadHistory(id)
}

// CreatePastesBatch создаёт несколько паст за один вызов.
func (s *PasteService) CreatePastesBatch(inputs []CreatePasteInput, user *models.User) (int, error) {
	pastes := make([]*models.Paste, 0, len(inputs))
	for _, in := range inputs {
		if err := validateBatchTitle(in.Title); err != nil {
			return 0, err
		}

		pastes = append(pastes, &models.Paste{
			Title:    in.Title,
			Category: in.Category,
			Text:     in.Text,
			Minutes:  in.Minutes,
			Version:  1,
		})
	}

	if err := s.repo.CreateBatch(pastes); err != nil {
		return 0, err
	}

	s.log.Debugf("user %d created %d pastes in batch", user.ID, len(pastes))

	return len(pastes), nil
}

// CreateCategory создаёт новую категорию паст.
func (s *PasteService) CreateCategory(name string) error {
	if err := validateCategoryName(name); err != nil {
		return err
	}

	return s.repo.CreateCategory(name)
}

// ModeratePaste проверяет текст пасты во внешнем сервисе модерации.
func (s *PasteService) ModeratePaste(id int64) (string, error) {
	text, err := s.repo.ReadText(id)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(helpers.Envelope{"text": text})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, moderationEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+moderationAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// FetchLinkPreview загружает превью страницы по ссылке из пасты.
func (s *PasteService) FetchLinkPreview(target string) (string, error) {
	resp, err := http.Get(target)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// FetchLinkPreviewSafe загружает превью страницы с таймаутом и проверкой статуса.
func (s *PasteService) FetchLinkPreviewSafe(target string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// DownloadAttachment читает вложение пасты.
func (s *PasteService) DownloadAttachment(name string) ([]byte, error) {
	return s.attachments.Read(name)
}

// DeleteAttachment удаляет вложение пасты.
func (s *PasteService) DeleteAttachment(name string) error {
	return s.attachments.Delete(name)
}

func validateCategoryName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("must not be blank")
	}
	return nil
}

func validateBatchTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("must not be blank")
	}
	return nil
}
