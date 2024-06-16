package data

import "errors"

type Categories []string

var CategoriesList Categories = []string{
	"Sport",
	"Home",
	"Work",
}

func (c Categories) GetCategory(categoryId uint8) (string, error) {
	if !c.IsValidCategory(categoryId) {
		return "", errors.New("category ID is out of range")
	}
	return c[categoryId-1], nil
}

func (c Categories) IsValidCategory(categoryId uint8) bool {
	if categoryId == 0 || int(categoryId) > len(c) {
		return false
	}
	return true
}
