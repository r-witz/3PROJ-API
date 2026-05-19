package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

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
		INSERT INTO users (id, email, email_verified, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		user.ID, user.Email, user.EmailVerified, user.PasswordHash, user.Username,
		user.AvatarURL, user.Bio, user.Website, user.Role, user.Theme, user.Locale,
		user.CreatedAt, user.UpdatedAt, user.BannedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, email_verified, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE id = $1
	`
	user := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.PasswordHash, &user.Username,
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
		SELECT id, email, email_verified, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE email = $1
	`
	user := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.PasswordHash, &user.Username,
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
		SELECT id, email, email_verified, password_hash, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users WHERE username = $1
	`
	user := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.PasswordHash, &user.Username,
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
	args := []interface{}{}
	argIndex := 1

	conditions := []string{}

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("username ILIKE $%d", argIndex))
		args = append(args, "%"+params.Query+"%")
		argIndex++
	}

	if len(params.ExcludeRoles) > 0 {
		conditions = append(conditions, fmt.Sprintf("role != ALL($%d::user_role[])", argIndex))
		roles := make([]string, len(params.ExcludeRoles))
		for i, r := range params.ExcludeRoles {
			roles[i] = string(r)
		}
		args = append(args, roles)
		argIndex++
	}

	if params.HideBanned {
		conditions = append(conditions, "banned_at IS NULL")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM users" + whereClause
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
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

	limitArg := argIndex
	offsetArg := argIndex + 1
	args = append(args, params.Limit, params.Offset)

	selectQuery := fmt.Sprintf(`
		SELECT id, email, email_verified, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
		FROM users%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, limitArg, offsetArg)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.Username,
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
		SET email = $2, email_verified = $3, password_hash = $4, username = $5, avatar_url = $6,
		    bio = $7, website = $8, role = $9, theme = $10, locale = $11, updated_at = $12, banned_at = $13
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query,
		user.ID, user.Email, user.EmailVerified, user.PasswordHash, user.Username,
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
		SELECT id, email, email_verified, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at, banned_at
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
			&user.ID, &user.Email, &user.EmailVerified, &user.Username,
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

func (r *UserRepository) SetEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	query := `UPDATE users SET email_verified = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id, verified)
	return err
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

func (r *UserRepository) GetUnverifiedBefore(ctx context.Context, before time.Time) ([]*domain.User, error) {
	query := `SELECT id, email FROM users WHERE email_verified = FALSE AND created_at < $1`
	rows, err := r.db.Pool.Query(ctx, query, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		if err := rows.Scan(&user.ID, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *UserRepository) DeleteUnverifiedBefore(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM users WHERE email_verified = FALSE AND created_at < $1`
	result, err := r.db.Pool.Exec(ctx, query, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *UserRepository) GetBannedUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT id FROM users WHERE banned_at IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

