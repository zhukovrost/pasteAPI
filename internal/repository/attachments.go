package repository

import (
	"os"
	"path/filepath"
)

type AttachmentModel struct {
	Dir string
}

// Read читает файл вложения по имени.
func (m *AttachmentModel) Read(name string) ([]byte, error) {
	full := filepath.Join(m.Dir, name)

	return os.ReadFile(full)
}

// Delete удаляет файл вложения по имени.
func (m *AttachmentModel) Delete(name string) error {
	name = filepath.Base(name)
	full := filepath.Join(m.Dir, name)

	if _, err := os.Stat(full); err == nil {
		return os.Remove(full)
	}

	return nil
}
