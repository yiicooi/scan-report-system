package service

import (
	"errors"

	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/middleware"
	"github.com/sui/scan-report/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("用户名或密码错误")
var ErrUserInactive = errors.New("账号已禁用")

func Login(name, password string) (accessToken, refreshToken string, user *model.User, err error) {
	user = &model.User{}
	if err = database.DB.Preload("Role.Permissions").
		Where("name = ?", name).First(user).Error; err != nil {
		return "", "", nil, ErrInvalidCredentials
	}
	if !user.IsActive {
		return "", "", nil, ErrUserInactive
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}
	accessToken, refreshToken, err = middleware.GenerateToken(user)
	return
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
