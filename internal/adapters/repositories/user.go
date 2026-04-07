package repositories

import (
	"context"
	"errors"
	"fmt"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
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

var allowedSortColumns = map[string]string{
	"username":   "username",
	"created_at": "created_at",
}

func (r *UserRepository) SearchByUsername(ctx context.Context, params ports.UserSearchParams) ([]*domain.User, int, error) {
	countQuery := `SELECT COUNT(*) FROM users WHERE username ILIKE $1`
	var total int
	err := r.db.Pool.QueryRow(ctx, countQuery, "%"+params.Query+"%").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	orderBy := "username ASC"
	if params.SortField != "" {
		if col, ok := allowedSortColumns[params.SortField]; ok {
			order := "ASC"
			if params.SortOrder == "DESC" {
				order = "DESC"
			}
			orderBy = fmt.Sprintf("%s %s", col, order)
		}
	}

	searchQuery := fmt.Sprintf(`
		SELECT id, email, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE username ILIKE $1
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, orderBy)

	rows, err := r.db.Pool.Query(ctx, searchQuery, "%"+params.Query+"%", params.Limit, params.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.Username,
			&user.AvatarURL, &user.Bio, &user.Website, &user.Role, &user.Theme, &user.Locale,
			&user.CreatedAt, &user.UpdatedAt, &user.BannedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
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

func (r *UserRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, email, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE id = ANY($1)
	`
	rows, err := r.db.Pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.Username,
			&user.AvatarURL, &user.Bio, &user.Website, &user.Role, &user.Theme, &user.Locale,
			&user.CreatedAt, &user.UpdatedAt, &user.BannedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *UserRepository) ExistsByRole(ctx context.Context, role domain.UserRole) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = $1)`, role).Scan(&exists)
	return exists, err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *UserRepository) ListAll(ctx context.Context, offset, limit int, bannedOnly bool) ([]*domain.User, int, error) {
	whereClause := ""
	if bannedOnly {
		whereClause = " WHERE banned_at IS NOT NULL"
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM users" + whereClause
	if err := r.db.Pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, email, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users%s
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, whereClause)

	rows, err := r.db.Pool.Query(ctx, selectQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		if err := rows.Scan(
			&user.ID, &user.Email, &user.Username,
			&user.AvatarURL, &user.Bio, &user.Website, &user.Role, &user.Theme, &user.Locale,
			&user.CreatedAt, &user.UpdatedAt, &user.BannedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, total, rows.Err()
}
