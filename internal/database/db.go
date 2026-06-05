package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Init 初始化SQLite数据库并建表
func Init(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 启用WAL模式，提高并发性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		url         TEXT NOT NULL,
		file_path   TEXT,
		file_size   INTEGER DEFAULT 0,
		downloaded  INTEGER DEFAULT 0,
		speed       INTEGER DEFAULT 0,
		status      TEXT DEFAULT 'pending',
		progress    REAL DEFAULT 0,
		source      TEXT DEFAULT 'manual',
		headers     TEXT,
		cookies     TEXT,
		error_msg   TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS search_history (
		id          TEXT PRIMARY KEY,
		query       TEXT NOT NULL,
		results     TEXT,
		model_used  TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		key         TEXT PRIMARY KEY,
		value       TEXT NOT NULL
	);
	`
	_, err := db.Exec(schema)
	return err
}
