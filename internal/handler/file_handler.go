package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"filestore/internal/model"
	"filestore/internal/service"
)

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

// RegisterFile 文件上传注册接口
// POST /api/v1/files/register
func (h *FileHandler) RegisterFile(c *gin.Context) {
	var req model.RegisterFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.svc.RegisterFile(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateMD5) {
			c.JSON(http.StatusConflict, model.Response{
				Code:    http.StatusConflict,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "register file failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// RecordChunk 分片映射注册接口
// POST /api/v1/files/chunks
func (h *FileHandler) RecordChunk(c *gin.Context) {
	var req model.ChunkMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.svc.RecordChunk(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) || errors.Is(err, service.ErrNodeNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		if errors.Is(err, service.ErrInvalidStatus) || errors.Is(err, service.ErrInvalidChunkInfo) {
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "record chunk failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// GetFileDetail 获取文件详情接口
// GET /api/v1/files/:id
func (h *FileHandler) GetFileDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid file id",
		})
		return
	}

	resp, err := h.svc.GetFileDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "query file detail failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// QueryFiles 文件检索接口
// GET /api/v1/files?file_name=xxx&start_time=xxx&end_time=xxx&page=1&page_size=10
func (h *FileHandler) QueryFiles(c *gin.Context) {
	var req model.FileQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid query params: " + err.Error(),
		})
		return
	}

	resp, err := h.svc.QueryFiles(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "query files failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// GetUploadProgress 获取上传进度接口
// GET /api/v1/files/:id/progress
func (h *FileHandler) GetUploadProgress(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid file id",
		})
		return
	}

	resp, err := h.svc.GetUploadProgress(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "get upload progress failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// CompleteFile 完成文件上传接口
// POST /api/v1/files/complete
func (h *FileHandler) CompleteFile(c *gin.Context) {
	var req model.CompleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.svc.CompleteFile(c.Request.Context(), req.FileID)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		if errors.Is(err, service.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}
		if errors.Is(err, service.ErrIncompleteChunks) {
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "complete file failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// SoftDeleteFile 软删除文件接口
// POST /api/v1/files/soft-delete
func (h *FileHandler) SoftDeleteFile(c *gin.Context) {
	var req model.SoftDeleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.svc.SoftDeleteFile(c.Request.Context(), req.FileID)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "soft delete file failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// RestoreFile 恢复已删除文件接口
// POST /api/v1/files/restore
func (h *FileHandler) RestoreFile(c *gin.Context) {
	var req model.RestoreFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.svc.RestoreFile(c.Request.Context(), req.FileID)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		if errors.Is(err, service.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "restore file failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// DeleteFile 删除文件接口（支持软删除和强制删除）
// DELETE /api/v1/files/:id
func (h *FileHandler) DeleteFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    http.StatusBadRequest,
			Message: "invalid file id",
		})
		return
	}

	force := c.Query("force") == "true"

	var resp *model.DeleteFileResponse
	if force {
		resp, err = h.svc.ForceDeleteFile(c.Request.Context(), id)
	} else {
		resp, err = h.svc.SoftDeleteFile(c.Request.Context(), id)
	}

	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    http.StatusInternalServerError,
			Message: "delete file failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// Health 健康检查接口
// GET /api/v1/health
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "ok",
	})
}
