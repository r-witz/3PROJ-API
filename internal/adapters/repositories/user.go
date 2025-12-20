package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Username,
		user.AvatarURL, user.Bio, user.Website, user.Role, user.Theme, user.Locale,
		user.CreatedAt, user.UpdatedAt, user.BannedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE id = $1
	`
	user := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Username,
		&user.AvatarURL, &user.Bio, &user.Website, &user.Role, &user.Theme, &user.Locale,
		&user.CreatedAt, &user.UpdatedAt, &user.BannedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE email = $1
	`
	user := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Username,
		&user.AvatarURL, &user.Bio, &user.Website, &user.Role, &user.Theme, &user.Locale,
		&user.CreatedAt, &user.UpdatedAt, &user.BannedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE username = $1
	`
	user := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Username,
		&user.AvatarURL, &user.Bio, &user.Website, &user.Role, &user.Theme, &user.Locale,
		&user.CreatedAt, &user.UpdatedAt, &user.BannedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, username = $4, avatar_url = $5,
		    bio = $6, website = $7, role = $8, theme = $9, locale = $10, updated_at = $11, banned_at = $12
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Username,
		user.AvatarURL, user.Bio, user.Website, user.Role, user.Theme, user.Locale,
		user.UpdatedAt, user.BannedAt,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
