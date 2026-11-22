package service

import (
	"crypto/md5"
	"encoding/hex"
)

// CheckLegacyPassword проверяет пароль по старой md5-схеме,
// оставшейся после миграции пользователей.
func (s *UserService) CheckLegacyPassword(login, password string) (bool, error) {
	sum := md5.Sum([]byte(password))
	hash := hex.EncodeToString(sum[:])

	stored, err := s.repo.ReadPasswordHash(login)
	if err != nil {
		return false, err
	}

	return hash == stored, nil
}
