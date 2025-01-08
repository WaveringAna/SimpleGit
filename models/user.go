//models/user.go

package models

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Email     string    `gorm:"unique" json:"email"`
	Password  string    `json:"-"`
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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

func (s *UserService) CreateUser(email, password string, isAdmin bool) (*User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create new user with UUID
	user := &User{
		ID:        uuid.New().String(),
		Email:     email,
		Password:  string(hashedPassword),
		IsAdmin:   isAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *UserService) AuthenticateUser(email, password string) (*User, string, error) {
	var user User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", err
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"admin":   user.IsAdmin,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
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
