package repository

import (
	"context"

	"github.com/kerucko/diploma/gateway/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db      *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db:      db,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	query, args, err := r.builder.
		Insert("users").
		Columns("name", "password_hash", "role").
		Values(u.Name, u.PasswordHash, u.Role).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return err
	}
	return r.db.QueryRow(ctx, query, args...).Scan(&u.ID)
}

func (r *UserRepository) GetByName(ctx context.Context, name string) (*models.User, error) {
	query, args, err := r.builder.
		Select("id", "name", "password_hash", "role").
		From("users").
		Where(squirrel.Eq{"name": name}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var u models.User
	err = r.db.QueryRow(ctx, query, args...).Scan(&u.ID, &u.Name, &u.PasswordHash, &u.Role)
	if err != nil {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]models.User, error) {
	query, args, err := r.builder.
		Select("id", "name", "role").
		From("users").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
