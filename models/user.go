//models/user.go

package models

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Email     string    `gorm:"unique" json:"email"`
	Password  string    `json:"-"`
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SSHKey struct {
	ID          string    `gorm:"primarykey" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserService handles user-related operations and authentication
type UserService struct {
	db     *gorm.DB
	jwtKey []byte
}

func NewUserService(db *gorm.DB, jwtKey []byte) *UserService {
	return &UserService{
		db:     db,
		jwtKey: jwtKey,
	}
}

func (s *UserService) GetUserByEmail(email string) (*User, error) {
	var user User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUserSSHKeys(userID string) ([]SSHKey, error) {
	var keys []SSHKey
	if err := s.db.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *UserService) AddSSHKey(userID, name, publicKey string) (*SSHKey, error) {
	// Generate fingerprint
	h := sha256.New()
	h.Write([]byte(publicKey))
	fingerprint := base64.StdEncoding.EncodeToString(h.Sum(nil))

	key := &SSHKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(key).Error; err != nil {
		return nil, fmt.Errorf("failed to add SSH key: %w", err)
	}

	return key, nil
}

func (s *UserService) DeleteSSHKey(userID, keyID string) error {
	result := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&SSHKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete SSH key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("SSH key not found")
	}
	return nil
}

func (s *UserService) CreateUser(username, email, password string, isAdmin bool) (*User, error) {
	// Validate username (only allow letters, numbers, hyphens, and underscores)
	if !validateUsername(username) {
		return nil, fmt.Errorf("invalid username: must contain only letters, numbers, hyphens, and underscores")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:        uuid.New().String(),
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		IsAdmin:   isAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func validateUsername(username string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", username)
	return matched && len(username) >= 3 && len(username) <= 39
}

func (s *UserService) GetUserByUsername(username string) (*User, error) {
	var user User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) AuthenticateUser(login, password string) (*User, string, error) {
	var user User

	// Try to find user by email or username
	if err := s.db.Where("email = ? OR username = ?", login, login).First(&user).Error; err != nil {
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"admin":    user.IsAdmin,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return nil, "", err
	}

	return &user, tokenString, nil
}

func (s *UserService) VerifyToken(tokenString string) (*User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtKey, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, err
	}

	var user User
	if err := s.db.First(&user, "id = ?", claims["user_id"]).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) GetAdminCount() (int64, error) {
	var count int64
	err := s.db.Model(&User{}).Where("is_admin = ?", true).Count(&count).Error
	return count, err
}
