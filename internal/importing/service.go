package importing

import "errors"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ImportFromImage(path string) error {
	_ = path
	return errors.New("image import is not implemented yet")
}
