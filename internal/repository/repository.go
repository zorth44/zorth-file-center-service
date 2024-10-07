package repository

import (
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

// 在这里添加数据库操作方法
// 例如:
// func (r *Repository) GetUsers() ([]model.User, error) {
//     var users []model.User
//     result := r.db.Find(&users)
//     return users, result.Error
// }
// 由于代码量较少，所以不使用
