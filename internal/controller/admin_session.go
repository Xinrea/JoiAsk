package controller

import (
	"errors"
	"joiask-backend/internal/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const adminSessionVersionKey = "admin_session_version"

// CurrentAdmin validates the cookie against the current account, including its
// credential version. Both required and optional administrator auth use this
// check; the cookie's authed flag alone never grants administrator privileges.
// A database failure is distinct from an absent or revoked session.
func CurrentAdmin(c *gin.Context) (database.Admin, bool, error) {
	session := sessions.Default(c)
	id, validID := session.Get("user").(uint)
	version, validVersion := session.Get(adminSessionVersionKey).(int)
	if session.Get("authed") != true || !validID || id == 0 || !validVersion || version < 0 {
		return database.Admin{}, false, nil
	}
	var admin database.Admin
	if err := database.DB.First(&admin, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return database.Admin{}, false, nil
		}
		return database.Admin{}, false, err
	}
	if version != admin.SessionVersion {
		return database.Admin{}, false, nil
	}
	return admin, true, nil
}
