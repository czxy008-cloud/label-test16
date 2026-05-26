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
	ErrIncompleteChunks = errors.New("incomplete chunks")
	ErrInvalidStatus    = errors.New("invalid file status")
)

func chunkStatusText(status int) string {
	switch status {
	case model.ChunkStatusNormal:
		return "正常"
	case model.ChunkStatusCorrupt:
		return "损坏"
	case model.ChunkStatusMigrate:
		return "已迁移"
	default:
		return "未知"
	}
}

func fileStatusText(status int) string {
	switch status {
	case model.FileStatusUploading:
		return "上传中"
	case model.FileStatusCompleted:
		return "已完成"
	case model.FileStatusDeleted:
		return "已删除"
	default:
		return "未知"
	}
}

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

	// 检查文件是否存在，并拿到分片总数和状态
	var fileExists bool
	var chunkCount uint32
	var fileStatus int
	err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM file_index WHERE id = ?), COALESCE((SELECT chunk_count FROM file_index WHERE id = ?), 0), COALESCE((SELECT status FROM file_index WHERE id = ?), 0)", req.FileID, req.FileID, req.FileID).Scan(&fileExists, &chunkCount, &fileStatus)
	if err != nil {
		return nil, fmt.Errorf("check file: %w", err)
	}
	if !fileExists {
		return nil, fmt.Errorf("file_id %d: %w", req.FileID, ErrFileNotFound)
	}
	if fileStatus == model.FileStatusDeleted {
		return nil, fmt.Errorf("file_id %d is deleted, cannot record chunk: %w", req.FileID, ErrInvalidStatus)
	}
	if chunkCount > 0 && req.ChunkIndex >= chunkCount {
		return nil, fmt.Errorf("chunk_index %d out of range [0, %d): %w", req.ChunkIndex, chunkCount, ErrInvalidChunkInfo)
	}

	// 先查询是否已存在，避免 ON DUPLICATE KEY UPDATE 时 LastInsertId 返回 0
	var existingID uint64
	err = s.db.QueryRowContext(ctx, "SELECT id FROM file_chunk WHERE file_id = ? AND chunk_index = ?", req.FileID, req.ChunkIndex).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing chunk: %w", err)
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

	chunkID := existingID
	if chunkID == 0 {
		if lid, lidErr := result.LastInsertId(); lidErr == nil && lid > 0 {
			chunkID = uint64(lid)
		}
	}

	return &model.ChunkMappingResponse{
		ChunkID:    chunkID,
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

// GetUploadProgress 获取文件上传进度
func (s *FileService) GetUploadProgress(ctx context.Context, fileID uint64) (*model.UploadProgressResponse, error) {
	var file model.FileIndex
	err := s.db.QueryRowContext(ctx, `
		SELECT id, file_name, chunk_count, status
		FROM file_index WHERE id = ?
	`, fileID).Scan(&file.ID, &file.FileName, &file.ChunkCount, &file.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("query file: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT chunk_index, status
		FROM file_chunk WHERE file_id = ? ORDER BY chunk_index
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("query chunks: %w", err)
	}
	defer rows.Close()

	registeredSet := make(map[uint32]int)
	chunksStatus := make([]model.ChunkStatusInfo, 0)
	for rows.Next() {
		var idx uint32
		var status int
		if err := rows.Scan(&idx, &status); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		registeredSet[idx] = status
		chunksStatus = append(chunksStatus, model.ChunkStatusInfo{
			ChunkIndex: idx,
			Status:     status,
			StatusText: chunkStatusText(status),
		})
	}

	registered := uint32(len(registeredSet))
	if file.ChunkCount > 0 && registered > file.ChunkCount {
		registered = file.ChunkCount
	}

	missing := make([]uint32, 0)
	for i := uint32(0); i < file.ChunkCount; i++ {
		if _, exists := registeredSet[i]; !exists {
			missing = append(missing, i)
		}
	}

	progress := 0.0
	if file.ChunkCount > 0 {
		progress = float64(registered) / float64(file.ChunkCount) * 100
		if progress > 100 {
			progress = 100
		}
	}

	return &model.UploadProgressResponse{
		FileID:        file.ID,
		FileName:      file.FileName,
		ChunkCount:    file.ChunkCount,
		Registered:    registered,
		Progress:      progress,
		Status:        file.Status,
		StatusText:    fileStatusText(file.Status),
		MissingChunks: missing,
		ChunksStatus:  chunksStatus,
	}, nil
}

// SoftDeleteFile 软删除文件，将状态更新为已删除，级联更新所有关联分片
func (s *FileService) SoftDeleteFile(ctx context.Context, fileID uint64) (resp *model.DeleteFileResponse, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var fileName string
	var status int
	err = tx.QueryRowContext(ctx, `
		SELECT file_name, status FROM file_index WHERE id = ? FOR UPDATE
	`, fileID).Scan(&fileName, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrFileNotFound
			return nil, err
		}
		return nil, fmt.Errorf("query file: %w", err)
	}

	if status == model.FileStatusDeleted {
		_ = tx.Rollback()
		return &model.DeleteFileResponse{
			FileID:     fileID,
			FileName:   fileName,
			Status:     model.FileStatusDeleted,
			StatusText: fileStatusText(model.FileStatusDeleted),
			Deleted:    true,
		}, nil
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE file_index SET status = ?, updated_at = ? WHERE id = ?
	`, model.FileStatusDeleted, now, fileID)
	if err != nil {
		return nil, fmt.Errorf("update file status: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE file_chunk SET status = ?, updated_at = ? WHERE file_id = ?
	`, model.ChunkStatusMigrate, now, fileID)
	if err != nil {
		return nil, fmt.Errorf("update chunk status: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &model.DeleteFileResponse{
		FileID:     fileID,
		FileName:   fileName,
		Status:     model.FileStatusDeleted,
		StatusText: fileStatusText(model.FileStatusDeleted),
		Deleted:    true,
	}, nil
}

// RestoreFile 恢复已删除的文件，将状态更新为上传中，级联恢复所有关联分片
func (s *FileService) RestoreFile(ctx context.Context, fileID uint64) (resp *model.RestoreFileResponse, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var fileName string
	var status int
	err = tx.QueryRowContext(ctx, `
		SELECT file_name, status FROM file_index WHERE id = ? FOR UPDATE
	`, fileID).Scan(&fileName, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrFileNotFound
			return nil, err
		}
		return nil, fmt.Errorf("query file: %w", err)
	}

	if status != model.FileStatusDeleted {
		err = fmt.Errorf("file status is %s, expected 已删除: %w", fileStatusText(status), ErrInvalidStatus)
		return nil, err
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE file_index SET status = ?, updated_at = ? WHERE id = ?
	`, model.FileStatusUploading, now, fileID)
	if err != nil {
		return nil, fmt.Errorf("update file status: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE file_chunk SET status = ?, updated_at = ? WHERE file_id = ?
	`, model.ChunkStatusNormal, now, fileID)
	if err != nil {
		return nil, fmt.Errorf("update chunk status: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &model.RestoreFileResponse{
		FileID:     fileID,
		FileName:   fileName,
		Status:     model.FileStatusUploading,
		StatusText: fileStatusText(model.FileStatusUploading),
		Restored:   true,
	}, nil
}

// ForceDeleteFile 强制删除文件，从数据库中永久删除文件及其所有分片记录
func (s *FileService) ForceDeleteFile(ctx context.Context, fileID uint64) (resp *model.DeleteFileResponse, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var fileName string
	err = tx.QueryRowContext(ctx, `
		SELECT file_name FROM file_index WHERE id = ? FOR UPDATE
	`, fileID).Scan(&fileName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrFileNotFound
			return nil, err
		}
		return nil, fmt.Errorf("query file: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM file_chunk WHERE file_id = ?
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("delete chunks: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM file_index WHERE id = ?
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("delete file: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &model.DeleteFileResponse{
		FileID:     fileID,
		FileName:   fileName,
		Status:     0,
		StatusText: "已永久删除",
		Deleted:    true,
	}, nil
}

// CompleteFile 标记文件上传完成
func (s *FileService) CompleteFile(ctx context.Context, fileID uint64) (resp *model.CompleteFileResponse, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var fileName string
	var chunkCount uint32
	var status int
	err = tx.QueryRowContext(ctx, `
		SELECT file_name, chunk_count, status FROM file_index WHERE id = ? FOR UPDATE
	`, fileID).Scan(&fileName, &chunkCount, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrFileNotFound
			return nil, err
		}
		return nil, fmt.Errorf("query file: %w", err)
	}

	if status != model.FileStatusUploading {
		err = fmt.Errorf("file status is %s, expected 上传中: %w", fileStatusText(status), ErrInvalidStatus)
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT chunk_index FROM file_chunk WHERE file_id = ? ORDER BY chunk_index
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("query chunks: %w", err)
	}

	registeredSet := make(map[uint32]bool)
	for rows.Next() {
		var idx uint32
		if err = rows.Scan(&idx); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		registeredSet[idx] = true
	}
	rows.Close()

	registered := uint32(len(registeredSet))
	if registered != chunkCount {
		missing := make([]uint32, 0)
		for i := uint32(0); i < chunkCount; i++ {
			if !registeredSet[i] {
				missing = append(missing, i)
			}
		}
		err = fmt.Errorf("registered chunks %d != chunk_count %d, missing chunks: %v: %w",
			registered, chunkCount, missing, ErrIncompleteChunks)
		return nil, err
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE file_index SET status = ?, uploaded_at = ? WHERE id = ?
	`, model.FileStatusCompleted, now, fileID)
	if err != nil {
		return nil, fmt.Errorf("update file status: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &model.CompleteFileResponse{
		FileID:     fileID,
		FileName:   fileName,
		Status:     model.FileStatusCompleted,
		StatusText: fileStatusText(model.FileStatusCompleted),
		UploadedAt: now.Format(time.RFC3339),
	}, nil
}
