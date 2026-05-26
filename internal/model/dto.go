package model

// RegisterFileRequest 文件上传注册请求
type RegisterFileRequest struct {
	FileName   string `json:"file_name"   binding:"required"`
	FileSize   uint64 `json:"file_size"   binding:"required"`
	FileType   string `json:"file_type"`
	MD5        string `json:"md5"         binding:"required,len=32"`
	ChunkCount uint32 `json:"chunk_count" binding:"required"`
	UploadedBy string `json:"uploaded_by"`
}

// RegisterFileResponse 文件上传注册响应
type RegisterFileResponse struct {
	FileID    uint64 `json:"file_id"`
	FileName  string `json:"file_name"`
	MD5       string `json:"md5"`
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
}

// FileDetailResponse 文件详情响应（包含分片信息）
type FileDetailResponse struct {
	File  FileIndex   `json:"file"`
	Chunks []FileChunk `json:"chunks"`
}

// FileQueryRequest 文件检索请求参数
type FileQueryRequest struct {
	FileName    string `form:"file_name"`
	StartTime   string `form:"start_time"`
	EndTime     string `form:"end_time"`
	Page        int    `form:"page,default=1"`
	PageSize    int    `form:"page_size,default=10"`
}

// FileListResponse 文件列表响应
type FileListResponse struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Files    []FileIndex `json:"files"`
}

// ChunkMappingRequest 分片映射注册请求
type ChunkMappingRequest struct {
	FileID     uint64 `json:"file_id"     binding:"required"`
	ChunkIndex uint32 `json:"chunk_index" binding:"required"`
	ChunkSize  uint64 `json:"chunk_size"  binding:"required"`
	ChunkMD5   string `json:"chunk_md5"   binding:"required,len=32"`
	NodeID     uint64 `json:"node_id"     binding:"required"`
	ObjectKey  string `json:"object_key"  binding:"required"`
}

// ChunkMappingResponse 分片映射响应
type ChunkMappingResponse struct {
	ChunkID    uint64 `json:"chunk_id"`
	FileID     uint64 `json:"file_id"`
	ChunkIndex uint32 `json:"chunk_index"`
	NodeID     uint64 `json:"node_id"`
	ObjectKey  string `json:"object_key"`
	Status     int    `json:"status"`
}

// ChunkStatusInfo 分片状态信息
type ChunkStatusInfo struct {
	ChunkIndex uint32 `json:"chunk_index"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
}

// UploadProgressResponse 上传进度响应
type UploadProgressResponse struct {
	FileID         uint64            `json:"file_id"`
	FileName       string            `json:"file_name"`
	ChunkCount     uint32            `json:"chunk_count"`
	Registered     uint32            `json:"registered"`
	Progress       float64           `json:"progress"`
	Status         int               `json:"status"`
	StatusText     string            `json:"status_text"`
	MissingChunks  []uint32          `json:"missing_chunks"`
	ChunksStatus   []ChunkStatusInfo `json:"chunks_status"`
}

// CompleteFileRequest 完成文件上传请求
type CompleteFileRequest struct {
	FileID uint64 `json:"file_id" binding:"required"`
}

// CompleteFileResponse 完成文件上传响应
type CompleteFileResponse struct {
	FileID     uint64 `json:"file_id"`
	FileName   string `json:"file_name"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
	UploadedAt string `json:"uploaded_at"`
}

// SoftDeleteFileRequest 软删除文件请求
type SoftDeleteFileRequest struct {
	FileID uint64 `json:"file_id" binding:"required"`
}

// RestoreFileRequest 恢复文件请求
type RestoreFileRequest struct {
	FileID uint64 `json:"file_id" binding:"required"`
}

// DeleteFileRequest 删除文件请求（支持软删除和强制删除）
type DeleteFileRequest struct {
	FileID  uint64 `json:"file_id" binding:"required"`
	Force   bool   `json:"force"`
}

// DeleteFileResponse 删除文件响应
type DeleteFileResponse struct {
	FileID     uint64 `json:"file_id"`
	FileName   string `json:"file_name"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
	Deleted    bool   `json:"deleted"`
}

// RestoreFileResponse 恢复文件响应
type RestoreFileResponse struct {
	FileID     uint64 `json:"file_id"`
	FileName   string `json:"file_name"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
	Restored   bool   `json:"restored"`
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
