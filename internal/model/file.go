package model

import "gorm.io/gorm"

var (
	db *gorm.DB
)

// File represents the file entity in the database
type File struct {
	gorm.Model
	Filename      string
	Filepath      string
	Size          int64
	PasswordHash  string
	DownloadCount int
}

// ActivityLog represents the activity log entity in the database
type ActivityLog struct {
	gorm.Model
	FileID  uint
	Action  string
	Details string
}

type UploadResponse struct {
	ID       uint   `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type ShareLink struct {
	gorm.Model
	FileID uint
	Token  string
}
