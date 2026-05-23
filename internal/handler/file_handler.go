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

// Health 健康检查接口
// GET /api/v1/health
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "ok",
	})
}
