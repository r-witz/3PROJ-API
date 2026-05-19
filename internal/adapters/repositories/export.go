package repositories

import (
	"context"
	"encoding/json"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
)

type ExportRepository struct {
	db *database.DB
}

func NewExportRepository(db *database.DB) *ExportRepository {
	return &ExportRepository{db: db}
}

func (r *ExportRepository) GetAllUserData(ctx context.Context, userID uuid.UUID) (*domain.UserDataExport, error) {
	export := &domain.UserDataExport{
		ExportedAt: time.Now().UTC(),
	}

	user, err := r.getUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	export.User = *user

	if export.Reviews, err = r.getReviews(ctx, userID); err != nil {
		return nil, err
	}
	if export.Comments, err = r.getComments(ctx, userID); err != nil {
		return nil, err
	}
	if export.Collections, err = r.getCollections(ctx, userID); err != nil {
		return nil, err
	}
	if export.Messages, err = r.getMessages(ctx, userID); err != nil {
		return nil, err
	}
	if export.Followers, err = r.getFollowerIDs(ctx, userID); err != nil {
		return nil, err
	}
	if export.Following, err = r.getFollowingIDs(ctx, userID); err != nil {
		return nil, err
	}
	if export.BlockedUsers, err = r.getBlockedIDs(ctx, userID); err != nil {
		return nil, err
	}
	if export.ReviewLikes, err = r.getReviewLikes(ctx, userID); err != nil {
		return nil, err
	}
	if export.CommentLikes, err = r.getCommentLikes(ctx, userID); err != nil {
		return nil, err
	}
	if export.Activities, err = r.getActivities(ctx, userID); err != nil {
		return nil, err
	}
	if export.Notifications, err = r.getNotifications(ctx, userID); err != nil {
		return nil, err
	}
	if export.OAuthAccounts, err = r.getOAuthAccounts(ctx, userID); err != nil {
		return nil, err
	}

	return export, nil
}

func (r *ExportRepository) getUserProfile(ctx context.Context, userID uuid.UUID) (*domain.UserProfileExport, error) {
	query := `
		SELECT id, email, email_verified, username, avatar_url, bio, website, role, theme, locale, created_at, updated_at
		FROM users WHERE id = $1
	`
	var u domain.UserProfileExport
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Email, &u.EmailVerified, &u.Username,
		&u.AvatarURL, &u.Bio, &u.Website, &u.Role, &u.Theme, &u.Locale,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *ExportRepository) getReviews(ctx context.Context, userID uuid.UUID) ([]domain.Review, error) {
	query := `
		SELECT id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at
		FROM reviews WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []domain.Review
	for rows.Next() {
		var rv domain.Review
		if err := rows.Scan(&rv.ID, &rv.UserID, &rv.TMDBID, &rv.Rating, &rv.Content,
			&rv.ContainsSpoilers, &rv.FeaturedAt, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

func (r *ExportRepository) getComments(ctx context.Context, userID uuid.UUID) ([]domain.Comment, error) {
	query := `
		SELECT id, user_id, review_id, content, contains_spoilers, created_at, updated_at
		FROM comments WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.UserID, &c.ReviewID, &c.Content,
			&c.ContainsSpoilers, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *ExportRepository) getCollections(ctx context.Context, userID uuid.UUID) ([]domain.CollectionExport, error) {
	query := `
		SELECT id, user_id, name, slug, type, visibility, description, created_at, updated_at
		FROM collections WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []domain.CollectionExport
	for rows.Next() {
		var c domain.Collection
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Slug, &c.Type,
			&c.Visibility, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		collections = append(collections, domain.CollectionExport{Collection: c})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(collections) > 0 {
		seen := make(map[uuid.UUID]struct{}, len(collections))
		ids := make([]uuid.UUID, 0, len(collections))
		for i := range collections {
			id := collections[i].Collection.ID
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}

		itemsByCollection, err := r.getCollectionItemsByCollectionIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		for i := range collections {
			collections[i].Items = itemsByCollection[collections[i].Collection.ID]
		}
	}

	return collections, nil
}

func (r *ExportRepository) getCollectionItemsByCollectionIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]domain.CollectionItem, error) {
	result := make(map[uuid.UUID][]domain.CollectionItem, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	query := `
		SELECT collection_id, tmdb_id, added_at, runtime, metadata
		FROM collection_items WHERE collection_id = ANY($1) ORDER BY collection_id, added_at
	`
	rows, err := r.db.Pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.CollectionItem
		var metadata []byte
		if err := rows.Scan(&item.CollectionID, &item.TMDBID, &item.AddedAt, &item.Runtime, &metadata); err != nil {
			return nil, err
		}
		item.Metadata = json.RawMessage(metadata)
		result[item.CollectionID] = append(result[item.CollectionID], item)
	}
	return result, rows.Err()
}

func (r *ExportRepository) getCollectionItems(ctx context.Context, collectionID uuid.UUID) ([]domain.CollectionItem, error) {
	query := `
		SELECT collection_id, tmdb_id, added_at, runtime, metadata
		FROM collection_items WHERE collection_id = $1 ORDER BY added_at
	`
	rows, err := r.db.Pool.Query(ctx, query, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.CollectionItem
	for rows.Next() {
		var item domain.CollectionItem
		var metadata []byte
		if err := rows.Scan(&item.CollectionID, &item.TMDBID, &item.AddedAt, &item.Runtime, &metadata); err != nil {
			return nil, err
		}
		item.Metadata = json.RawMessage(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ExportRepository) getMessages(ctx context.Context, userID uuid.UUID) ([]domain.Message, error) {
	query := `
		SELECT id, sender_id, receiver_id, content, read_at, created_at, updated_at
		FROM messages WHERE sender_id = $1 OR receiver_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content,
			&m.ReadAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *ExportRepository) getFollowerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT follower_id FROM follows WHERE following_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, userID)
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

func (r *ExportRepository) getFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT following_id FROM follows WHERE follower_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, userID)
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

func (r *ExportRepository) getBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT blocked_id FROM user_blocks WHERE blocker_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, userID)
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

func (r *ExportRepository) getReviewLikes(ctx context.Context, userID uuid.UUID) ([]domain.ReviewLike, error) {
	query := `
		SELECT user_id, review_id, created_at
		FROM review_likes WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likes []domain.ReviewLike
	for rows.Next() {
		var l domain.ReviewLike
		if err := rows.Scan(&l.UserID, &l.ReviewID, &l.CreatedAt); err != nil {
			return nil, err
		}
		likes = append(likes, l)
	}
	return likes, rows.Err()
}

func (r *ExportRepository) getCommentLikes(ctx context.Context, userID uuid.UUID) ([]domain.CommentLike, error) {
	query := `
		SELECT user_id, comment_id, created_at
		FROM comment_likes WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likes []domain.CommentLike
	for rows.Next() {
		var l domain.CommentLike
		if err := rows.Scan(&l.UserID, &l.CommentID, &l.CreatedAt); err != nil {
			return nil, err
		}
		likes = append(likes, l)
	}
	return likes, rows.Err()
}

func (r *ExportRepository) getActivities(ctx context.Context, userID uuid.UUID) ([]domain.Activity, error) {
	query := `
		SELECT id, user_id, type, review_id, collection_id, comment_id, tmdb_id, target_user_id, created_at
		FROM activities WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []domain.Activity
	for rows.Next() {
		var a domain.Activity
		if err := rows.Scan(&a.ID, &a.UserID, &a.Type, &a.ReviewID, &a.CollectionID,
			&a.CommentID, &a.TMDBID, &a.TargetUserID, &a.CreatedAt); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

func (r *ExportRepository) getNotifications(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, review_id, comment_id, message, read_at, created_at
		FROM notifications WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.ReviewID,
			&n.CommentID, &n.Message, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *ExportRepository) getOAuthAccounts(ctx context.Context, userID uuid.UUID) ([]domain.OAuthAccountExport, error) {
	query := `
		SELECT provider, created_at
		FROM oauth_accounts WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []domain.OAuthAccountExport
	for rows.Next() {
		var a domain.OAuthAccountExport
		if err := rows.Scan(&a.Provider, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}
