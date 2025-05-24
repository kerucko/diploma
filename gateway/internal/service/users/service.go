package users

import (
	"context"
	"errors"

	"github.com/kerucko/diploma/gateway/internal/models"
	"github.com/kerucko/diploma/gateway/internal/utils"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type userRepository interface {
	Register(ctx context.Context, input models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, input models.LoginRequest) (string, error)
	GetAll(ctx context.Context) ([]models.User, error)
	Create(ctx context.Context, input *models.User) (*models.User, error)
	GetByName(ctx context.Context, name string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
}

type userService struct {
	repo userRepository
	auth *utils.AuthManager
}

func NewService(r userRepository, auth *utils.AuthManager) *userService {
	return &userService{repo: r, auth: auth}
}

func (s *userService) Register(ctx context.Context, input models.RegisterRequest) (*models.User, error) {
	hash, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Name:         input.Name,
		PasswordHash: hash,
		Role:         "user",
	}
	return s.repo.Create(ctx, user)
}

func (s *userService) Login(ctx context.Context, input models.LoginRequest) (string, error) {
	user, err := s.repo.GetByName(ctx, input.Name)
	if err != nil || !utils.CheckPasswordHash(input.Password, user.PasswordHash) {
		return "", ErrUnauthorized
	}
	return s.auth.GenerateToken(user)
}

func (s *userService) GetAll(ctx context.Context) ([]models.User, error) {
	return s.repo.GetAll(ctx)
}

func (s *userService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}
