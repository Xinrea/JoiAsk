package database

import "gorm.io/gorm"

// WithAdminManagementLock serializes account-management decisions across server
// processes. The first statement takes row locks on MySQL and the write lock on
// SQLite without modifying credentials, session versions, or timestamps.
func WithAdminManagementLock(db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Raw SQL also bypasses model callbacks: this is a lock, not a mutation.
		if err := tx.Exec("UPDATE admins SET session_version = session_version WHERE id > 0").Error; err != nil {
			return err
		}
		return fn(tx)
	})
}
