package controller

import (
	"errors"
	"joiask-backend/internal/adminpassword"
	"joiask-backend/internal/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserController struct{}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserModifyRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Role     *string `json:"role"`
}

type UserAddRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (*UserController) Info(c *gin.Context) {
	Success(c, c.MustGet("user"))
}

func (*UserController) Login(c *gin.Context) {
	if !checkAdminLoginRate(c) {
		return
	}
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if request.Username == "" || request.Password == "" {
		Fail(c, 400, "用户名和密码不能为空")
		return
	}
	var user database.Admin
	err := database.DB.Where("username = ?", request.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, 401, "用户名或密码错误")
		} else {
			Fail(c, 500, "内部错误")
		}
		return
	}
	matches, err := verifyAndUpgradeAdminPassword(&user, request.Password)
	if err != nil {
		log.Errorf("failed to migrate admin password: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	if !matches {
		Fail(c, 401, "用户名或密码错误")
		return
	}
	session := sessions.Default(c)
	session.Set("user", user.ID)
	session.Set("authed", true)
	// Bind to the version read with the verified password, not a fresh version
	// that could belong to a concurrent password reset.
	session.Set(adminSessionVersionKey, user.SessionVersion)
	err = session.Save()
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, nil)
}

func verifyAndUpgradeAdminPassword(user *database.Admin, password string) (bool, error) {
	matches, needsMigration := adminpassword.Verify(user.Password, password)
	if !matches || !needsMigration {
		return matches, nil
	}
	hash, err := adminpassword.Hash(password)
	if err != nil {
		return false, err
	}
	// Do not overwrite a concurrent password reset, deletion, or migration.
	result := database.DB.Model(&database.Admin{}).
		Where("id = ? AND password = ?", user.ID, user.Password).
		Update("password", hash)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		if err := database.DB.First(user, user.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		matches, _ := adminpassword.Verify(user.Password, password)
		return matches, nil
	}
	user.Password = hash
	return true, nil
}

func (*UserController) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user")
	session.Delete("authed")
	session.Delete(adminSessionVersionKey)
	err := session.Save()
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, nil)
}

func (*UserController) Get(c *gin.Context) {
	if c.MustGet("user").(database.Admin).Role != database.AdminRoleSuperAdmin {
		Fail(c, 403, "仅超级管理员可管理后台账号")
		return
	}
	var users []database.Admin
	if err := database.DB.Find(&users).Error; err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, users)
}

func (*UserController) Put(c *gin.Context) {
	id, ok := parseRecordID(c, c.Param("id"))
	if !ok {
		return
	}
	var request UserModifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	actor := c.MustGet("user").(database.Admin)
	if actor.Role != database.AdminRoleSuperAdmin && (actor.ID != uint(id) || request.Role != nil) {
		Fail(c, 403, "仅超级管理员可管理后台账号及角色")
		return
	}
	if request.Role != nil && !validAdminRole(*request.Role) {
		Fail(c, 400, "管理员角色无效")
		return
	}
	updates := map[string]interface{}{}
	if request.Username != "" {
		updates["username"] = request.Username
	}
	// An omitted or empty password leaves the existing credential unchanged.
	if request.Password != "" {
		hash, err := adminpassword.Hash(request.Password)
		if err != nil {
			adminPasswordError(c, err)
			return
		}
		updates["password"] = hash
		// Revoke every previously issued session in the same write as the
		// password change. A database increment also preserves concurrent resets.
		updates["session_version"] = gorm.Expr("session_version + 1")
	}
	var user database.Admin
	err := withAdminActor(actor, func(tx *gorm.DB, current database.Admin) error {
		if current.Role != database.AdminRoleSuperAdmin && (current.ID != uint(id) || request.Role != nil) {
			return adminMutationError{403, "仅超级管理员可管理后台账号及角色"}
		}
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}
		if request.Role != nil && *request.Role != user.Role {
			if err := protectLastSuperAdmin(tx, user); err != nil {
				return err
			}
			updates["role"] = *request.Role
			updates["session_version"] = gorm.Expr("session_version + 1")
		}
		// Only update requested fields, never a stale full credential record.
		if err := tx.Model(&database.Admin{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&user, id).Error
	})
	if respondAdminMutationError(c, err) {
		return
	}
	Success(c, user)
}

func (*UserController) Post(c *gin.Context) {
	actor := c.MustGet("user").(database.Admin)
	if actor.Role != database.AdminRoleSuperAdmin {
		Fail(c, 403, "仅超级管理员可管理后台账号")
		return
	}
	var request UserAddRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if request.Username == "" {
		Fail(c, 400, "用户名不能为空")
		return
	}
	if request.Role == "" {
		request.Role = database.AdminRoleAdmin
	}
	if !validAdminRole(request.Role) {
		Fail(c, 400, "管理员角色无效")
		return
	}
	hash, err := adminpassword.Hash(request.Password)
	if err != nil {
		adminPasswordError(c, err)
		return
	}
	var user database.Admin
	user.Username = request.Username
	user.Password = hash
	user.Role = request.Role
	err = withAdminActor(actor, func(tx *gorm.DB, current database.Admin) error {
		if current.Role != database.AdminRoleSuperAdmin {
			return adminMutationError{403, "仅超级管理员可管理后台账号"}
		}
		return tx.Create(&user).Error
	})
	if respondAdminMutationError(c, err) {
		return
	}
	Success(c, user)
}

func adminPasswordError(c *gin.Context, err error) {
	if errors.Is(err, adminpassword.ErrEmptyPassword) {
		Fail(c, 400, err.Error())
		return
	}
	log.Errorf("failed to hash admin password: %v", err)
	Fail(c, 500, "内部错误")
}

func (*UserController) Delete(c *gin.Context) {
	userModel := c.MustGet("user").(database.Admin)
	id, ok := parseRecordID(c, c.Param("id"))
	if !ok {
		return
	}
	if userModel.Role != database.AdminRoleSuperAdmin {
		Fail(c, 403, "仅超级管理员可管理后台账号")
		return
	}
	if uint(id) == userModel.ID {
		Fail(c, 403, "不能删除当前登录账号")
		return
	}
	if respondAdminMutationError(c, deleteAdminAccount(userModel, uint(id))) {
		return
	}
	Success(c, nil)
}

var errAdminMutationSessionExpired = errors.New("administrator session expired during account mutation")

// Revalidate the actor while all account mutations are serialized. Middleware's
// snapshot alone is insufficient if a concurrent request deleted/demoted them.
func withAdminActor(actor database.Admin, mutate func(*gorm.DB, database.Admin) error) error {
	return database.WithAdminManagementLock(database.DB, func(tx *gorm.DB) error {
		var current database.Admin
		if err := tx.First(&current, actor.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errAdminMutationSessionExpired
			}
			return err
		}
		if current.SessionVersion != actor.SessionVersion {
			return errAdminMutationSessionExpired
		}
		return mutate(tx, current)
	})
}

func deleteAdminAccount(actor database.Admin, id uint) error {
	return withAdminActor(actor, func(tx *gorm.DB, current database.Admin) error {
		if current.Role != database.AdminRoleSuperAdmin {
			return adminMutationError{403, "仅超级管理员可管理后台账号"}
		}
		if current.ID == id {
			return adminMutationError{403, "不能删除当前登录账号"}
		}
		var target database.Admin
		if err := tx.First(&target, id).Error; err != nil {
			return err
		}
		if err := protectLastSuperAdmin(tx, target); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
}

func validAdminRole(role string) bool {
	return role == database.AdminRoleAdmin || role == database.AdminRoleSuperAdmin
}

func protectLastSuperAdmin(tx *gorm.DB, target database.Admin) error {
	if target.Role != database.AdminRoleSuperAdmin {
		return nil
	}
	var count int64
	if err := tx.Model(&database.Admin{}).Where("role = ?", database.AdminRoleSuperAdmin).Count(&count).Error; err != nil {
		return err
	}
	if count <= 1 {
		return adminMutationError{403, "必须保留至少一个超级管理员"}
	}
	return nil
}

type adminMutationError struct {
	code    int
	message string
}

func (e adminMutationError) Error() string { return e.message }

func respondAdminMutationError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var rejected adminMutationError
	switch {
	case errors.As(err, &rejected):
		Fail(c, rejected.code, rejected.message)
	case errors.Is(err, errAdminMutationSessionExpired):
		Fail(c, 408, "请先登录")
	case errors.Is(err, gorm.ErrRecordNotFound):
		Fail(c, 404, "用户不存在")
	default:
		log.Error("failed to mutate administrator account")
		Fail(c, 500, "内部错误")
	}
	return true
}
