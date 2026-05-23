package service

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"filestore/internal/model"
)

var (
	ErrFileNotFound     = errors.New("file not found")
	ErrDuplicateMD5     = errors.New("file with same md5 already exists")
	ErrInvalidChunkInfo = errors.New("invalid chunk information")
	ErrNodeNotFound     = errors.New("storage node not found")
)

// FileService 文件元数据服务
type FileService struct {
	db *sql.DB
}

func NewFileService(db *sql.DB) *FileService {
	return &FileService{db: db}
}

// RegisterFile 注册文件元数据，校验 MD5 并入库
func (s *FileService) RegisterFile(ctx context.Context, req model.RegisterFileRequest) (*model.RegisterFileResponse, error) {
	if len(req.MD5) != 32 {
		return nil, fmt.Errorf("invalid md5 length: %w", ErrInvalidChunkInfo)
	}

	// 去重检查
	var existingID uint64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM file_index WHERE md5 = ?", req.MD5).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("md5 %s: %w", req.MD5, ErrDuplicateMD5)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check duplicate md5: %w", err)
	}

	now := time.Now()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO file_index (file_name, file_size, file_type, md5, chunk_count, uploaded_by, status, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, req.FileName, req.FileSize, req.FileType, req.MD5, req.ChunkCount, req.UploadedBy, model.FileStatusUploading, now)
	if err != nil {
		return nil, fmt.Errorf("insert file_index: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return &model.RegisterFileResponse{
		FileID:    uint64(id),
		FileName:  req.FileName,
		MD5:       req.MD5,
		Status:    model.FileStatusUploading,
		CreatedAt: now.Format(time.RFC3339),
	}, nil
}

// RecordChunk 记录分片映射信息
func (s *FileService) RecordChunk(ctx context.Context, req model.ChunkMappingRequest) (*model.ChunkMappingResponse, error) {
	// 检查节点是否存在
	var nodeExists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM storage_node WHERE id = ?)", req.NodeID).Scan(&nodeExists)
	if err != nil {
		return nil, fmt.Errorf("check node: %w", err)
	}
	if !nodeExists {
		return nil, fmt.Errorf("node_id %d: %w", req.NodeID, ErrNodeNotFound)
	}

	// 检查文件是否存在
	var fileExists bool
	err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM file_index WHERE id = ?)", req.FileID).Scan(&fileExists)
	if err != nil {
		return nil, fmt.Errorf("check file: %w", err)
	}
	if !fileExists {
		return nil, fmt.Errorf("file_id %d: %w", req.FileID, ErrFileNotFound)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO file_chunk (file_id, chunk_index, chunk_size, chunk_md5, node_id, object_key, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE chunk_size=VALUES(chunk_size), chunk_md5=VALUES(chunk_md5),
			node_id=VALUES(node_id), object_key=VALUES(object_key), status=VALUES(status)
	`, req.FileID, req.ChunkIndex, req.ChunkSize, req.ChunkMD5, req.NodeID, req.ObjectKey, model.ChunkStatusNormal)
	if err != nil {
		return nil, fmt.Errorf("insert file_chunk: %w", err)
	}

	chunkID, _ := result.LastInsertId()

	return &model.ChunkMappingResponse{
		ChunkID:    uint64(chunkID),
		FileID:     req.FileID,
		ChunkIndex: req.ChunkIndex,
		NodeID:     req.NodeID,
		ObjectKey:  req.ObjectKey,
		Status:     model.ChunkStatusNormal,
	}, nil
}

// GetFileDetail 获取文件详情（含分片列表）
func (s *FileService) GetFileDetail(ctx context.Context, fileID uint64) (*model.FileDetailResponse, error) {
	var file model.FileIndex
	err := s.db.QueryRowContext(ctx, `
		SELECT id, file_name, file_size, file_type, md5, chunk_count, uploaded_by, status, uploaded_at, created_at, updated_at
		FROM file_index WHERE id = ?
	`, fileID).Scan(&file.ID, &file.FileName, &file.FileSize, &file.FileType, &file.MD5, &file.ChunkCount,
		&file.UploadedBy, &file.Status, &file.UploadedAt, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("query file: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, chunk_index, chunk_size, chunk_md5, node_id, object_key, status, created_at, updated_at
		FROM file_chunk WHERE file_id = ? ORDER BY chunk_index
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("query chunks: %w", err)
	}
	defer rows.Close()

	chunks := make([]model.FileChunk, 0)
	for rows.Next() {
		var c model.FileChunk
		if err := rows.Scan(&c.ID, &c.FileID, &c.ChunkIndex, &c.ChunkSize, &c.ChunkMD5,
			&c.NodeID, &c.ObjectKey, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return &model.FileDetailResponse{File: file, Chunks: chunks}, nil
}

// QueryFiles 按文件名和时间范围检索文件
func (s *FileService) QueryFiles(ctx context.Context, req model.FileQueryRequest) (*model.FileListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	where := "WHERE 1=1"
	args := make([]interface{}, 0)

	if req.FileName != "" {
		where += " AND file_name LIKE ?"
		args = append(args, "%"+req.FileName+"%")
	}
	if req.StartTime != "" {
		where += " AND uploaded_at >= ?"
		args = append(args, req.StartTime)
	}
	if req.EndTime != "" {
		where += " AND uploaded_at <= ?"
		args = append(args, req.EndTime)
	}

	var total int64
	countSQL := "SELECT COUNT(*) FROM file_index " + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}

	offset := (req.Page - 1) * req.PageSize
	querySQL := `
		SELECT id, file_name, file_size, file_type, md5, chunk_count, uploaded_by, status, uploaded_at, created_at, updated_at
		FROM file_index ` + where + ` ORDER BY uploaded_at DESC LIMIT ? OFFSET ?
	`
	queryArgs := append(append([]interface{}{}, args...), req.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	files := make([]model.FileIndex, 0)
	for rows.Next() {
		var f model.FileIndex
		if err := rows.Scan(&f.ID, &f.FileName, &f.FileSize, &f.FileType, &f.MD5, &f.ChunkCount,
			&f.UploadedBy, &f.Status, &f.UploadedAt, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}

	return &model.FileListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Files:    files,
	}, nil
}

// ComputeMD5 计算数据的 MD5 十六进制字符串
func ComputeMD5(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}
