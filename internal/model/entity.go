package model

import "time"

const (
	NodeStatusOnline   = 1
	NodeStatusOffline  = 2
	NodeStatusMaintain = 3

	FileStatusUploading = 1
	FileStatusCompleted = 2
	FileStatusDeleted   = 3

	ChunkStatusNormal  = 1
	ChunkStatusCorrupt = 2
	ChunkStatusMigrate = 3
)

// StorageNode 表示集群中的一个存储节点
type StorageNode struct {
	ID         uint64    `db:"id"          json:"id"`
	NodeName   string    `db:"node_name"   json:"node_name"`
	NodeAddr   string    `db:"node_addr"   json:"node_addr"`
	TotalSpace uint64    `db:"total_space" json:"total_space"`
	UsedSpace  uint64    `db:"used_space"  json:"used_space"`
	Status     int       `db:"status"      json:"status"`
	Heartbeat  time.Time `db:"heartbeat"   json:"heartbeat"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updated_at"`
}

// FileIndex 文件元数据索引
type FileIndex struct {
	ID          uint64    `db:"id"           json:"id"`
	FileName    string    `db:"file_name"    json:"file_name"`
	FileSize    uint64    `db:"file_size"    json:"file_size"`
	FileType    string    `db:"file_type"    json:"file_type"`
	MD5         string    `db:"md5"          json:"md5"`
	ChunkCount  uint32    `db:"chunk_count"  json:"chunk_count"`
	UploadedBy  string    `db:"uploaded_by"  json:"uploaded_by"`
	Status      int       `db:"status"       json:"status"`
	UploadedAt  time.Time `db:"uploaded_at"  json:"uploaded_at"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
}

// FileChunk 文件分片映射
type FileChunk struct {
	ID         uint64    `db:"id"          json:"id"`
	FileID     uint64    `db:"file_id"     json:"file_id"`
	ChunkIndex uint32    `db:"chunk_index" json:"chunk_index"`
	ChunkSize  uint64    `db:"chunk_size"  json:"chunk_size"`
	ChunkMD5   string    `db:"chunk_md5"   json:"chunk_md5"`
	NodeID     uint64    `db:"node_id"     json:"node_id"`
	ObjectKey  string    `db:"object_key"  json:"object_key"`
	Status     int       `db:"status"      json:"status"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updated_at"`
}
