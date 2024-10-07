package service

import (
	"github.com/zorth44/zorth-file-center-service/internal/repository"
)

type Service struct {
	repo *repository.Repository
}

func NewService(repo *repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// 在这里添加服务方法
// 例如:
// func (s *Service) GetUsers() ([]model.User, error) {
//     return s.repo.GetUsers()
// }
// 由于代码量较少，所以不使用
