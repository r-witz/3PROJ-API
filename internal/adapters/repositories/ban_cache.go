package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const bannedUsersKey = "banned_users"

type BanCache struct {
	client *redis.Client
}

func NewBanCache(client *redis.Client) *BanCache {
	return &BanCache{client: client}
}

func (c *BanCache) IsBanned(ctx context.Context, userID uuid.UUID) (bool, error) {
	return c.client.SIsMember(ctx, bannedUsersKey, userID.String()).Result()
}

func (c *BanCache) GetBannedUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	members, err := c.client.SMembers(ctx, bannedUsersKey).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		id, err := uuid.Parse(m)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *BanCache) SetBanned(ctx context.Context, userID uuid.UUID) error {
	return c.client.SAdd(ctx, bannedUsersKey, userID.String()).Err()
}

func (c *BanCache) RemoveBanned(ctx context.Context, userID uuid.UUID) error {
	return c.client.SRem(ctx, bannedUsersKey, userID.String()).Err()
}

func (c *BanCache) SyncBannedUsers(ctx context.Context, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		c.client.Del(ctx, bannedUsersKey)
		return nil
	}

	pipe := c.client.Pipeline()
	pipe.Del(ctx, bannedUsersKey)
	members := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		members[i] = id.String()
	}
	pipe.SAdd(ctx, bannedUsersKey, members...)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync banned users to cache: %w", err)
	}
	return nil
}
