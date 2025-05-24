package tasks

import (
	"context"
	"errors"
	"time"

	"github.com/kerucko/diploma/gateway/internal/models"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type taskRepository interface {
	Create(ctx context.Context, input *models.Task) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.Task, error)
	GetByUser(ctx context.Context, userID int64) ([]models.Task, error)
	GetAll(ctx context.Context) ([]models.Task, error)
}

type taskService struct {
	repo taskRepository
}

func NewService(r taskRepository) *taskService {
	return &taskService{repo: r}
}

func (s *taskService) Create(ctx context.Context, input models.Task) (*models.Task, error) {
	task := &models.Task{
		UserID:      input.UserID,
		Name:        input.Name,
		Status:      input.Status,
		Description: input.Description,
		Solution:    input.Solution,
		CreatedAt:   time.Now(),
	}
	return s.repo.Create(ctx, task)
}

func (s *taskService) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *taskService) GetByUser(ctx context.Context, userID int64) ([]models.Task, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *taskService) GetAll(ctx context.Context) ([]models.Task, error) {
	return s.repo.GetAll(ctx)
}
