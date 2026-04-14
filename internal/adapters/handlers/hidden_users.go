package handlers

import (
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func IsCallerAdmin(c *gin.Context) bool {
	role, _ := middleware.GetRole(c)
	return role == string(domain.UserRoleAdmin) || role == string(domain.UserRoleSuperAdmin)
}

func GetHiddenUserIDs(c *gin.Context, blockService ports.BlockService, banCache ports.BanCache) map[uuid.UUID]struct{} {
	hiddenSet := make(map[uuid.UUID]struct{})

	isAdmin := IsCallerAdmin(c)

	if !isAdmin {
		if bannedIDs, err := banCache.GetBannedUserIDs(c.Request.Context()); err == nil {
			for _, id := range bannedIDs {
				hiddenSet[id] = struct{}{}
			}
		}
	}

	currentUserID, ok := middleware.GetUserID(c)
	if !ok || isAdmin {
		return hiddenSet
	}

	ctx := c.Request.Context()
	if blockerIDs, err := blockService.GetBlockerIDs(ctx, currentUserID); err == nil {
		for _, id := range blockerIDs {
			hiddenSet[id] = struct{}{}
		}
	}
	if blockedIDs, err := blockService.GetBlockedIDs(ctx, currentUserID); err == nil {
		for _, id := range blockedIDs {
			hiddenSet[id] = struct{}{}
		}
	}

	return hiddenSet
}

func IsBannedForCaller(c *gin.Context, banCache ports.BanCache, userID uuid.UUID) bool {
	if IsCallerAdmin(c) {
		return false
	}
	banned, err := banCache.IsBanned(c.Request.Context(), userID)
	return err == nil && banned
}
