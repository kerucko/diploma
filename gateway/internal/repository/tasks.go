package repository

import (
	"context"

	"github.com/kerucko/diploma/gateway/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
	db      *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{
		db:      db,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *TaskRepository) Create(ctx context.Context, t *models.Task) (*models.Task, error) {
	query, args, err := r.builder.
		Insert("tasks").
		Columns("user_id", "name", "status", "description", "created_at").
		Values(t.UserID, t.Name, t.Status, t.Description, t.CreatedAt).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return err
	}
	return r.db.QueryRow(ctx, query, args...).Scan(&t.ID)
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	query, args, err := r.builder.
		Select("*").
		From("tasks").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var t models.Task
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Status, &t.Description,
		&t.Solution, &t.CreatedAt, &t.SolvedAt,
	)
	if err != nil {
		return nil, ErrNotFound
	}
	return &t, nil
}

func (r *TaskRepository) GetByUser(ctx context.Context, userID int64) ([]models.Task, error) {
	query, args, err := r.builder.
		Select("*").
		From("tasks").
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Status, &t.Description, &t.Solution, &t.CreatedAt, &t.SolvedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}


func (r *TaskRepository) GetAll(ctx context.Context) ([]models.Task, error) {
	query, args, err := r.builder.
		Select("*").
		From("tasks").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Status, &t.Description, &t.Solution, &t.CreatedAt, &t.SolvedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
