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

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
