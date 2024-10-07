package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/zorth44/zorth-file-center-service/internal/model"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/files/upload", h.uploadFileHandler)
		api.GET("/files/:id/download", h.downloadFileHandler)
		api.DELETE("/files/:id", h.deleteFileHandler)
		api.PUT("/files/:id/rename", h.renameFileHandler)
		api.GET("/files", h.getFilesHandler)
		api.GET("/files/search", h.searchFilesHandler)
		api.POST("/files/:id/share", h.generateShareLinkHandler)
	}
}

func (h *Handler) uploadFileHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving file"})
		return
	}
	defer file.Close()

	// 创建上传目录（如果不存在）
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create upload directory"})
		return
	}

	// 在上传目录中创建新文件
	dst, err := os.Create(filepath.Join("./uploads", header.Filename))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create the file"})
		return
	}
	defer dst.Close()

	// 将上传的文件复制到目标文件
	size, err := io.Copy(dst, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to save the file"})
		return
	}

	// 获取密码（如果提供）
	password := c.PostForm("password")
	var passwordHash string
	if password != "" {
		hash := sha256.Sum256([]byte(password))
		passwordHash = hex.EncodeToString(hash[:])
	}

	// 将文件信息保存到数据库
	fileRecord := model.File{
		Filename:     header.Filename,
		Filepath:     filepath.Join("./uploads", header.Filename),
		Size:         size,
		PasswordHash: passwordHash,
	}
	result := h.db.Create(&fileRecord)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to save file information to database"})
		return
	}

	// 记录活动日志
	activityLog := model.ActivityLog{
		FileID:  fileRecord.ID,
		Action:  "upload",
		Details: c.ClientIP(),
	}
	h.db.Create(&activityLog)

	// 发送响应
	c.JSON(http.StatusOK, model.UploadResponse{
		ID:       fileRecord.ID,
		Filename: fileRecord.Filename,
		Size:     fileRecord.Size,
	})
}

func (h *Handler) downloadFileHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var file model.File
	result := h.db.First(&file, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// 检查文件是否有密码保护
	if file.PasswordHash != "" {
		password := c.Query("password")
		hash := sha256.Sum256([]byte(password))
		if hex.EncodeToString(hash[:]) != file.PasswordHash {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}
	}

	// 设置 Content-Disposition 头部
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.Filename))

	// 提供文件下载
	c.File(file.Filepath)

	// 更新下载次数
	if err := h.db.Model(&file).Update("download_count", gorm.Expr("download_count + ?", 1)).Error; err != nil {
		fmt.Printf("Error updating download count: %v\n", err)
	}

	// 记录活动日志
	activityLog := model.ActivityLog{
		FileID:  file.ID,
		Action:  "download",
		Details: c.ClientIP(),
	}
	if err := h.db.Create(&activityLog).Error; err != nil {
		fmt.Printf("Error creating activity log: %v\n", err)
	}
}

func (h *Handler) deleteFileHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
		return
	}

	var file model.File
	result := h.db.First(&file, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件未找到"})
		return
	}

	// 从文件系统中删除文件
	if err := os.Remove(file.Filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法删除文件"})
		return
	}

	// 从数据库中删除文件记录
	if err := h.db.Delete(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法从数据库中删除文件记录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件已成功删除"})
}

func (h *Handler) renameFileHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
		return
	}

	var file model.File
	result := h.db.First(&file, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件未找到"})
		return
	}

	var input struct {
		NewFilename string `json:"new_filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 重命名文件系统中的文件
	newFilepath := filepath.Join(filepath.Dir(file.Filepath), input.NewFilename)
	if err := os.Rename(file.Filepath, newFilepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法重命名文件"})
		return
	}

	// 更新数据库中的文件记录
	file.Filename = input.NewFilename
	file.Filepath = newFilepath
	if err := h.db.Save(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法更新数据库中的文件记录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件已成功重命名", "new_filename": input.NewFilename})
}

func (h *Handler) getFilesHandler(c *gin.Context) {
	var files []model.File
	result := h.db.Order("created_at DESC").Find(&files)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取文件列表"})
		return
	}

	c.JSON(http.StatusOK, files)
}

func (h *Handler) searchFilesHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索查询不能为空"})
		return
	}

	var files []model.File
	result := h.db.Where("filename LIKE ?", "%"+query+"%").Find(&files)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索文件时出错"})
		return
	}

	c.JSON(http.StatusOK, files)
}

func (h *Handler) generateShareLinkHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
		return
	}

	var file model.File
	result := h.db.First(&file, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件未找到"})
		return
	}

	// 生成唯一的分享链接
	shareToken := generateUniqueToken()

	// 创建分享记录
	shareLink := model.ShareLink{
		FileID: file.ID,
		Token:  shareToken,
	}
	if err := h.db.Create(&shareLink).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建分享链接"})
		return
	}

	// 构建完整的分享URL
	shareURL := fmt.Sprintf("%s/api/files/share/%s", c.Request.Host, shareToken)

	c.JSON(http.StatusOK, gin.H{"share_url": shareURL})
}

func generateUniqueToken() string {
	// 生成一个唯一的令牌，例如使用UUID
	return uuid.New().String()
}
