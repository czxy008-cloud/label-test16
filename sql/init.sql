-- ============================================================
-- 文件存储服务 - 元数据管理模块
-- 数据库初始化脚本 (MySQL 8.0+)
-- ============================================================

-- -----------------------------------------------------------
-- 创建数据库
-- -----------------------------------------------------------
CREATE DATABASE IF NOT EXISTS `filestore`
    DEFAULT CHARACTER SET utf8mb4
    DEFAULT COLLATE utf8mb4_unicode_ci;

USE `filestore`;

-- ============================================================
-- 1. 存储节点表
--    记录集群中每个存储节点的基本信息与状态
-- ============================================================
DROP TABLE IF EXISTS `storage_node`;
CREATE TABLE `storage_node` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '节点主键ID',
    `node_name`   VARCHAR(128)    NOT NULL                COMMENT '节点名称，用于人类识别',
    `node_addr`   VARCHAR(256)    NOT NULL                COMMENT '节点网络地址，格式 host:port',
    `total_space` BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '节点总容量(字节)',
    `used_space`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '节点已用空间(字节)',
    `status`      TINYINT         NOT NULL DEFAULT 1      COMMENT '节点状态: 1=在线, 2=离线, 3=维护中',
    `heartbeat`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最近一次心跳时间',
    `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
    `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_node_name` (`node_name`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='存储节点信息表';

-- ============================================================
-- 2. 文件索引表
--    记录每个上传文件的元数据（名称、大小、MD5、类型等）
-- ============================================================
DROP TABLE IF EXISTS `file_index`;
CREATE TABLE `file_index` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '文件主键ID',
    `file_name`   VARCHAR(512)    NOT NULL                COMMENT '原始文件名',
    `file_size`   BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '文件大小(字节)',
    `file_type`   VARCHAR(128)    NOT NULL DEFAULT ''     COMMENT '文件 MIME 类型',
    `md5`         CHAR(32)        NOT NULL                COMMENT '文件内容 MD5 校验值(32位十六进制)',
    `chunk_count` INT UNSIGNED    NOT NULL DEFAULT 0      COMMENT '文件分片总数',
    `uploaded_by` VARCHAR(128)    NOT NULL DEFAULT ''     COMMENT '上传者标识',
    `status`      TINYINT         NOT NULL DEFAULT 1      COMMENT '文件状态: 1=上传中, 2=已完成, 3=已删除',
    `uploaded_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '上传完成时间',
    `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
    `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_md5` (`md5`),
    KEY `idx_file_name` (`file_name`),
    KEY `idx_uploaded_at` (`uploaded_at`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文件索引表';

-- ============================================================
-- 3. 文件分片表
--    记录每个文件分片在集群中的物理位置（对应哪个存储节点）
-- ============================================================
DROP TABLE IF EXISTS `file_chunk`;
CREATE TABLE `file_chunk` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '分片主键ID',
    `file_id`      BIGINT UNSIGNED NOT NULL                COMMENT '所属文件ID，关联 file_index.id',
    `chunk_index`  INT UNSIGNED    NOT NULL                COMMENT '分片序号(从0开始)',
    `chunk_size`   BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '分片大小(字节)',
    `chunk_md5`    CHAR(32)        NOT NULL                COMMENT '分片内容 MD5 校验值',
    `node_id`      BIGINT UNSIGNED NOT NULL                COMMENT '存储该分片的节点ID，关联 storage_node.id',
    `object_key`   VARCHAR(512)    NOT NULL                COMMENT '分片在节点上的对象键/路径',
    `status`       TINYINT         NOT NULL DEFAULT 1      COMMENT '分片状态: 1=正常, 2=已损坏, 3=已迁移',
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
    `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_file_chunk` (`file_id`, `chunk_index`),
    KEY `idx_node_id` (`node_id`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_chunk_file`  FOREIGN KEY (`file_id`) REFERENCES `file_index`   (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_chunk_node`  FOREIGN KEY (`node_id`) REFERENCES `storage_node` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文件分片映射表';

-- ============================================================
-- 初始化数据: 示例存储节点
-- ============================================================
INSERT INTO `storage_node` (`node_name`, `node_addr`, `total_space`, `used_space`, `status`) VALUES
    ('node-01', '192.168.1.101:9000', 1099511627776, 0, 1),
    ('node-02', '192.168.1.102:9000', 1099511627776, 0, 1),
    ('node-03', '192.168.1.103:9000', 1099511627776, 0, 1);
