package database

import "database/sql"

// SettingsStore 系统设置键值存储
type SettingsStore struct {
	DB *sql.DB
}

// NewSettingsStore 创建设置存储
func NewSettingsStore(db *sql.DB) *SettingsStore {
	return &SettingsStore{DB: db}
}

// GetAll 获取所有设置
func (s *SettingsStore) GetAll() (map[string]string, error) {
	rows, err := s.DB.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

// Set 设置单个键值
func (s *SettingsStore) Set(key, value string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// SetMulti 批量设置
func (s *SettingsStore) SetMulti(kv map[string]string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for k, v := range kv {
		if _, err := stmt.Exec(k, v); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete 删除设置
func (s *SettingsStore) Delete(key string) error {
	_, err := s.DB.Exec("DELETE FROM settings WHERE key = ?", key)
	return err
}