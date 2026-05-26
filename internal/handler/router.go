package handler

import (
	"github.com/gin-gonic/gin"

	"filestore/internal/config"
	"filestore/internal/middleware"
	"filestore/internal/service"
)

// SetupRouter 配置所有路由并返回 gin Engine
func SetupRouter(cfg *config.Config, fileSvc *service.FileService) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()

	// 健康检查（无需鉴权）
	r.GET("/api/v1/health", Health)

	// API v1 路由组（需要鉴权）
	v1 := r.Group("/api/v1")
	v1.Use(middleware.TokenAuth(cfg.Auth))
	{
		fh := NewFileHandler(fileSvc)

		// 文件上传注册
		v1.POST("/files/register", fh.RegisterFile)
		// 分片映射注册
		v1.POST("/files/chunks", fh.RecordChunk)
		// 文件详情（含分片）
		v1.GET("/files/:id", fh.GetFileDetail)
		// 文件检索
		v1.GET("/files", fh.QueryFiles)
		// 上传进度跟踪
		v1.GET("/files/:id/progress", fh.GetUploadProgress)
		// 文件状态完成
		v1.POST("/files/complete", fh.CompleteFile)
		// 软删除文件
		v1.POST("/files/soft-delete", fh.SoftDeleteFile)
		// 恢复已删除文件
		v1.POST("/files/restore", fh.RestoreFile)
		// 删除文件（支持软删除和强制删除）
		v1.DELETE("/files/:id", fh.DeleteFile)
	}

	return r
}
