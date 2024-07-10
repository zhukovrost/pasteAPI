package service

import (
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"strings"
	"sync"
	"time"
)

// pasteKey generates the key for searched paste
func pasteKey(id uint16) string {
	return fmt.Sprintf("paste:%d", id)
}

// searchKey generates the key for search result
// pattern: search:title:category:onlyUsers:sort:page:pagesize
func searchKey(settings SearchSettings) string {
	return fmt.Sprintf("search:%s:%d:%v:%s:%d:%d", settings.Title, settings.Category, settings.OnlyUsers, settings.Filters.Sort, settings.Filters.Page, settings.Filters.PageSize)
}

type PasteService struct {
	repo            repository.Pastes
	permissionsRepo repository.Permissions
	cache           cache.Cache
	log             *logrus.Logger
	wg              *sync.WaitGroup
}

func newPasteService(repo repository.Pastes, permissionsRepo repository.Permissions, log *logrus.Logger, cache cache.Cache, wg *sync.WaitGroup) *PasteService {
	return &PasteService{
		repo:            repo,
		permissionsRepo: permissionsRepo,
		log:             log,
		cache:           cache,
		wg:              wg,
	}
}

func (s *PasteService) GetList(settings SearchSettings, user *models.User) (*ListPastesOutput, error) {
	cacheKey := searchKey(settings)
	listOutput := &ListPastesOutput{}
	err := s.cache.Get(cacheKey, listOutput)

	if err == nil {
		v := validator.New()
		for _, paste := range listOutput.Pastes {
			validateTime(v, paste)
		}

		if v.Valid() {
			s.log.Debugf("pastes list got from cache")
			return listOutput, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.log.Error(err)
	}

	var (
		pastes   []*models.Paste
		metadata *models.Metadata
	)

	if settings.OnlyUsers && !user.IsAnonymous() {
		pastes, metadata, err = s.repo.ReadUserPastes(settings.Title, settings.Category, user, settings.Filters)
	} else {
		pastes, metadata, err = s.repo.ReadAll(settings.Title, settings.Category, user, settings.Filters)
	}

	if err != nil {
		return nil, err
	}

	listOutput = &ListPastesOutput{pastes, metadata}

	helpers.Background(s.wg, s.log, func() {
		in, err := s.cache.Exists(cacheKey)
		if err != nil {
			s.log.Error(err)
			return
		}

		if !in && s.cache.Set(cacheKey, listOutput) != nil {
			s.log.Error(err)
		}
	})
	return listOutput, nil
}

func (s *PasteService) GetPaste(id uint16, user *models.User) (*models.Paste, error) {
	paste := &models.Paste{}

	err := s.cache.Get(pasteKey(id), paste)
	if err == nil {
		v := validator.New()
		validateTime(v, paste)
		if v.Valid() {
			s.log.Debugf("paste (ID: %d) got from cache", paste.Id)
			return paste, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.log.Error(err)
	}

	return s.repo.Read(id, user)
}

func (s *PasteService) Delete(id uint16) error {
	// Delete paste from cache as well
	go func() {
		key := pasteKey(id)
		in, err := s.cache.Exists(key)
		if err != nil {
			s.log.Error(err)
			return
		}

		if in && s.cache.Delete(key) != nil {
			s.log.Error(err)
		}

		if in && s.cache.Invalidate(key) != nil {
			s.log.Error(err)
		}
	}()

	return s.repo.Delete(id)
}

func (s *PasteService) Create(paste *models.Paste, creator *models.User) error {
	err := s.repo.Create(paste)

	if err != nil {
		return err
	}

	if !creator.IsAnonymous() {
		paste.CanEdit = true
		err = s.permissionsRepo.SetWritePermission(creator.ID, paste.Id)
		if err != nil {
			return err
		}
	}

	helpers.Background(s.wg, s.log, func() {
		if err = s.cache.Set(pasteKey(paste.Id), paste); err == nil {
			if err = s.cache.Invalidate("search:*"); err == nil {
				s.log.Debugf("paste (ID: %d) added to cache", paste.Id)
				return
			}
		}
		s.log.Errorf("paste (ID: %d) not added to cache due to the error: %s", paste.Id, err)
	})

	return nil
}

func (s *PasteService) GetPasteForUpdate(pasteId uint16, in UpdatePasteInput) (*models.Paste, error) {
	paste := &models.Paste{}
	err := s.cache.Get(pasteKey(pasteId), paste)
	if err != nil {
		paste, err = s.repo.Read(pasteId, models.AnonymousUser)
		if err != nil {
			return nil, err
		}
	}

	if in.Title != nil {
		paste.Title = strings.TrimSpace(*in.Title)
	}
	if in.Category != nil {
		paste.Category = *in.Category
	}
	if in.Text != nil {
		paste.Text = strings.TrimSpace(*in.Text)
	}
	if in.Minutes != nil {
		paste.Minutes = *in.Minutes
		paste.ExpiresAt = paste.ExpiresAt.Add(time.Duration(paste.Minutes) * time.Minute)
	}

	return paste, nil
}

func (s *PasteService) Update(paste *models.Paste) error {
	err := s.repo.Update(paste)
	if err != nil {
		return err
	}

	if err = s.cache.Set(pasteKey(paste.Id), paste); err != nil {
		s.log.Errorf("paste (ID: %d) not added to cache due to the error: %s", paste.Id, err)
	} else {
		s.log.Debugf("paste (ID: %d) added to cache", paste.Id)
	}

	helpers.Background(s.wg, s.log, func() {
		if err = s.cache.Invalidate("search:*"); err != nil {
			s.log.Error(err)
		}
	})

	return nil
}

func (s *PasteService) GivePermission(pasteId uint16, userId int64) (*PastePermissionResponse, error) {
	if err := s.permissionsRepo.SetWritePermission(userId, pasteId); err != nil {
		return nil, err
	}
	return &PastePermissionResponse{Permission: models.Permission{PasteId: pasteId, UserId: userId}}, nil
}

func ValidatePaste(v *validator.MyValidator, p *models.Paste) {
	v.Check(p.Title != "", "title", "must be provided")
	v.Check(len(p.Title) <= 255, "title", "must not be more than 500 bytes long")

	v.Check(models.CategoriesList.IsValidCategory(p.Category), "category", "no such category")

	v.Check(p.Text != "", "text", "must be provided")
	v.Check(len(p.Title) <= 500, "title", "must not be more than 500 bytes long")
}

func validateTime(v *validator.MyValidator, p *models.Paste) {
	v.Check(p.ExpiresAt.After(time.Now()), "expiration", "this paste is expired")
}
