package database

import (
	"errors"
	"joiask-backend/internal/adminpassword"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrAdminExists   = errors.New("administrator already exists; use reset-password to change its password")
	ErrAdminMissing  = errors.New("administrator does not exist; use create to provision an account")
	errAdminDatabase = errors.New("administrator database operation failed; check database configuration, permissions and availability")
)

// OpenAdminDatabase is for offline operator commands: all backend instances must
// be stopped. It does not start the server or seed site data. Even migration and
// connection errors must not log SQL or secrets.
func OpenAdminDatabase() (*gorm.DB, error) {
	db, err := openConfiguredDB(&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, errAdminDatabase
	}
	err = db.AutoMigrate(&Admin{})
	if err == nil {
		err = initializeAdminRoles(db)
	}
	if err != nil {
		if sqlDB, openErr := db.DB(); openErr == nil {
			sqlDB.Close()
		}
		return nil, errAdminDatabase
	}
	return db, nil
}

// SetAdminPassword is exclusively for trusted offline provisioning/recovery.
// reset=false creates a super administrator but never overwrites an account;
// reset=true never creates one and preserves its role.
func SetAdminPassword(db *gorm.DB, username, password string, reset bool) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("administrator username must not be empty")
	}
	hash, err := adminpassword.Hash(password)
	if err != nil {
		return errors.New("unable to hash password; password must not be empty")
	}
	// GORM's normal error logger includes SQL values, including password hashes.
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	err = WithAdminManagementLock(db, func(tx *gorm.DB) error {
		if reset {
			result := tx.Model(&Admin{}).Where("username = ?", username).Updates(map[string]interface{}{
				"password":        hash,
				"session_version": gorm.Expr("session_version + 1"),
			})
			if result.Error != nil {
				return errAdminDatabase
			}
			if result.RowsAffected == 0 {
				return ErrAdminMissing
			}
			return nil
		}
		var count int64
		if err := tx.Model(&Admin{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return errAdminDatabase
		}
		if count != 0 {
			return ErrAdminExists
		}
		// The unique username constraint also protects concurrent creates.
		return tx.Create(&Admin{Username: username, Password: hash, Role: AdminRoleSuperAdmin}).Error
	})
	if errors.Is(err, ErrAdminExists) || errors.Is(err, ErrAdminMissing) {
		return err
	}
	if err != nil {
		return errAdminDatabase
	}
	return nil
}

// PromoteAdmin is an explicit offline recovery operation. Repeating it for an
// existing super administrator is harmless; a changed role revokes old sessions.
func PromoteAdmin(db *gorm.DB, username string) error {
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	err := WithAdminManagementLock(db, func(tx *gorm.DB) error {
		var admin Admin
		if err := tx.Select("id", "role").Where("username = ?", username).First(&admin).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAdminMissing
			}
			return errAdminDatabase
		}
		if admin.Role == AdminRoleSuperAdmin {
			return nil
		}
		return tx.Model(&Admin{}).Where("id = ?", admin.ID).Updates(map[string]interface{}{
			"role":            AdminRoleSuperAdmin,
			"session_version": gorm.Expr("session_version + 1"),
		}).Error
	})
	if errors.Is(err, ErrAdminMissing) {
		return ErrAdminMissing
	}
	if err != nil {
		return errAdminDatabase
	}
	return nil
}

func initializeAdminRoles(db *gorm.DB) error {
	return WithAdminManagementLock(db, migrateLegacyAdminRoles)
}

func migrateLegacyAdminRoles(tx *gorm.DB) error {
	var admins []Admin
	if err := tx.Select("id", "username").Where("role = ?", "").Find(&admins).Error; err != nil {
		return err
	}
	for _, admin := range admins {
		role := AdminRoleAdmin
		// Compare in Go so case-insensitive MySQL collations cannot promote
		// other spellings. After this migration usernames have no authority.
		if admin.Username == "admin" {
			role = AdminRoleSuperAdmin
		}
		if err := tx.Model(&Admin{}).Where("id = ? AND role = ?", admin.ID, "").UpdateColumn("role", role).Error; err != nil {
			return err
		}
	}
	return nil
}
