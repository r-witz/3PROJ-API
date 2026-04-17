package handlers

import (
	"fmt"
	"strconv"
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CollectionHandler struct {
	collectionService ports.CollectionService
	blockService      ports.BlockService
	banCache          ports.BanCache
}

func NewCollectionHandler(collectionService ports.CollectionService, blockService ports.BlockService, banCache ports.BanCache) *CollectionHandler {
	return &CollectionHandler{collectionService: collectionService, blockService: blockService, banCache: banCache}
}

type CreateCollectionRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100" example:"My Favorites"`
	Description *string `json:"description" binding:"omitempty,max=500" example:"A collection of my favorite movies"`
	Visibility  string  `json:"visibility" binding:"omitempty,oneof=public private" example:"private"`
}

type UpdateCollectionRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100" example:"Updated Name"`
	Description *string `json:"description" binding:"omitempty,max=500" example:"Updated description"`
	Visibility  *string `json:"visibility" binding:"omitempty,oneof=public private" example:"public"`
}

type AddItemRequest struct {
	TMDBID int `json:"tmdb_id" binding:"required" example:"550"`
}

type CollectionResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID      string  `json:"user_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Name        string  `json:"name" example:"My Favorites"`
	Slug        string  `json:"slug" example:"my-favorites"`
	Type        string  `json:"type" example:"custom"`
	Visibility  string  `json:"visibility" example:"private"`
	Description *string `json:"description,omitempty" example:"A collection of my favorite movies"`
	HasMovie    *bool   `json:"has_movie,omitempty" example:"true"`
	ItemCount   int     `json:"item_count" example:"5"`
	CreatedAt   string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}


// @Summary      Create a collection
// @Description  Create a new custom collection for the authenticated user
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        request body CreateCollectionRequest true "Collection details"
// @Success      201 {object} response.Response{data=CollectionResponse} "Collection created"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      409 {object} response.Response "Collection with this name already exists"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections [post]
func (h *CollectionHandler) Create(c *gin.Context) {
	authUserID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if authUserID != userID {
		response.Forbidden(c, "You can only create collections for yourself")
		return
	}

	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.CreateCollectionInput{
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	}

	collection, err := h.collectionService.Create(c.Request.Context(), userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action:       middleware.ActivityCreate,
		Type:         domain.ActivityTypeCollectionCreated,
		UserID:       userID,
		CollectionID: &collection.ID,
	})

	response.Created(c, toCollectionResponse(ports.CollectionWithPresence{Collection: collection}))
}

// @Summary      Get collection by slug
// @Description  Get a collection by user ID and slug. Returns the collection if public or if the requester is the owner. Returns 404 if the collection owner is banned (non-admin callers). Returns 403 if there is a block between the authenticated user and the collection owner.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        slug path string true "Collection slug"
// @Success      200 {object} response.Response{data=CollectionResponse} "Collection details"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "Collection not found or owner banned"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections/{slug} [get]
func (h *CollectionHandler) GetBySlug(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if IsBannedForCaller(c, h.banCache, userID) {
		response.HandleError(c, domain.ErrUserNotFound)
		return
	}

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != userID && !IsCallerAdmin(c) {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), currentUserID, userID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	slug := c.Param("slug")

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	cwp, err := h.collectionService.GetBySlug(c.Request.Context(), userID, slug, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toCollectionResponse(*cwp))
}

// @Summary      Get user's collections
// @Description  Get all collections for a user. Returns all collections if the requester is the owner, only public ones otherwise. When tmdb_id is provided, each collection includes a has_movie flag indicating whether the movie is in that collection. Use the type parameter to filter by collection type. Returns 404 if the user is banned (non-admin callers). Returns 403 if there is a block between the authenticated user and the target user.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        tmdb_id query int false "TMDB movie ID - when provided, adds has_movie flag to each collection"
// @Param        type query string false "Filter by collection type" Enums(system, custom)
// @Success      200 {object} response.Response{data=[]CollectionResponse} "List of collections"
// @Failure      400 {object} response.Response "Invalid user ID or TMDB ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections [get]
func (h *CollectionHandler) GetByUserID(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if IsBannedForCaller(c, h.banCache, userID) {
		response.HandleError(c, domain.ErrUserNotFound)
		return
	}

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != userID && !IsCallerAdmin(c) {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), currentUserID, userID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	var collectionType *domain.CollectionType
	if typeStr := c.Query("type"); typeStr != "" {
		ct := domain.CollectionType(typeStr)
		if ct != domain.CollectionTypeSystem && ct != domain.CollectionTypeCustom {
			response.BadRequest(c, "Invalid collection type, must be 'system' or 'custom'", nil)
			return
		}
		collectionType = &ct
	}

	if tmdbIDStr := c.Query("tmdb_id"); tmdbIDStr != "" {
		tmdbID, err := strconv.Atoi(tmdbIDStr)
		if err != nil {
			response.BadRequest(c, "Invalid TMDB ID", nil)
			return
		}
		collectionsWithPresence, err := h.collectionService.GetByUserIDAndTMDBID(c.Request.Context(), userID, tmdbID, requestingUserID, collectionType)
		if err != nil {
			response.HandleError(c, err)
			return
		}

		resp := make([]CollectionResponse, len(collectionsWithPresence))
		for i, cwp := range collectionsWithPresence {
			resp[i] = toCollectionResponse(cwp)
			hasMovie := cwp.HasMovie
			resp[i].HasMovie = &hasMovie
		}

		response.Success(c, resp)
		return
	}

	collections, err := h.collectionService.GetByUserID(c.Request.Context(), userID, requestingUserID, collectionType)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]CollectionResponse, len(collections))
	for i, cwp := range collections {
		resp[i] = toCollectionResponse(cwp)
	}

	response.Success(c, resp)
}

// @Summary      Update a collection
// @Description  Update a collection's name, description, or visibility. System collections cannot have their name changed.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        slug path string true "Collection slug"
// @Param        request body UpdateCollectionRequest true "Fields to update"
// @Success      200 {object} response.Response{data=CollectionResponse} "Updated collection"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      409 {object} response.Response "Collection with this name already exists"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections/{slug} [patch]
func (h *CollectionHandler) Update(c *gin.Context) {
	authUserID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if authUserID != userID {
		response.Forbidden(c, "You can only update your own collections")
		return
	}

	slug := c.Param("slug")

	var req UpdateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.UpdateCollectionInput{
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	}

	collection, err := h.collectionService.Update(c.Request.Context(), userID, slug, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toCollectionResponse(ports.CollectionWithPresence{Collection: collection}))
}

// @Summary      Delete a collection
// @Description  Delete a collection. System collections cannot be deleted.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        slug path string true "Collection slug"
// @Success      204 "Collection deleted"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections/{slug} [delete]
func (h *CollectionHandler) Delete(c *gin.Context) {
	authUserID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if authUserID != userID {
		response.Forbidden(c, "You can only delete your own collections")
		return
	}

	slug := c.Param("slug")

	if err := h.collectionService.Delete(c.Request.Context(), userID, slug); err != nil {
		response.HandleError(c, err)
		return
	}

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action: middleware.ActivityDelete,
		Type:   domain.ActivityTypeCollectionCreated,
		UserID: userID,
	})

	c.Status(204)
}

// @Summary      Add item to collection
// @Description  Add a movie to a collection by TMDB ID. Runtime is automatically fetched from TMDB.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        slug path string true "Collection slug"
// @Param        request body AddItemRequest true "Movie to add"
// @Success      201 {object} response.Response "Item added"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      409 {object} response.Response "Item already in collection"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections/{slug}/items [post]
func (h *CollectionHandler) AddItem(c *gin.Context) {
	authUserID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if authUserID != userID {
		response.Forbidden(c, "You can only add items to your own collections")
		return
	}

	slug := c.Param("slug")

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	item, err := h.collectionService.AddItem(c.Request.Context(), userID, slug, req.TMDBID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if slug == "to-watch" {
		middleware.QueueActivity(c, middleware.ActivityEvent{
			Action:       middleware.ActivityCreate,
			Type:         domain.ActivityTypeWatchlistItemAdded,
			UserID:       userID,
			CollectionID: &item.CollectionID,
			TMDBID:       &req.TMDBID,
		})
	} else if slug != "watched" {
		middleware.QueueActivity(c, middleware.ActivityEvent{
			Action:       middleware.ActivityCreate,
			Type:         domain.ActivityTypeCollectionItemAdded,
			UserID:       userID,
			CollectionID: &item.CollectionID,
			TMDBID:       &req.TMDBID,
		})
	}

	response.Created(c, toSimpleCollectionItemResponse(item))
}

// @Summary      Get collection items
// @Description  Get all items in a collection with pagination and TMDB movie details. Respects visibility rules. Returns 404 if the collection owner is banned (non-admin callers). Returns 403 if there is a block between the authenticated user and the collection owner.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        slug path string true "Collection slug"
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        sort query string false "Sort field with optional +/- prefix (+ asc, - desc; default -added_at). Allowed fields: added_at, release_date, tmdb_rating, duskforge_rating, our_rating, collection_rating"
// @Success      200 {object} response.PaginatedResponse{data=[]CollectionItemResponse} "List of items with movie details and ratings"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "Collection not found or owner banned"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections/{slug}/items [get]
func (h *CollectionHandler) GetItems(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if IsBannedForCaller(c, h.banCache, userID) {
		response.HandleError(c, domain.ErrUserNotFound)
		return
	}

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != userID && !IsCallerAdmin(c) {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), currentUserID, userID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	slug := c.Param("slug")
	offset, limit := parsePagination(c)
	language := middleware.GetLocale(c)
	sortOpt, err := parseCollectionItemSort(c.Query("sort"))
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	if sortOpt.Field == ports.CollectionItemSortByOurRating && requestingUserID == nil {
		response.BadRequest(c, "sort by our_rating requires authentication", nil)
		return
	}

	items, total, err := h.collectionService.GetItems(c.Request.Context(), userID, slug, requestingUserID, offset, limit, language, sortOpt)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessPaginated(c, items, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Remove item from collection
// @Description  Remove a movie from a collection by TMDB ID
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        slug path string true "Collection slug"
// @Param        tmdbId path int true "TMDB movie ID"
// @Success      204 "Item removed"
// @Failure      400 {object} response.Response "Invalid user ID or TMDB ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Collection or item not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/collections/{slug}/items/{tmdbId} [delete]
func (h *CollectionHandler) RemoveItem(c *gin.Context) {
	authUserID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if authUserID != userID {
		response.Forbidden(c, "You can only remove items from your own collections")
		return
	}

	slug := c.Param("slug")

	tmdbIDStr := c.Param("tmdbId")
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid TMDB ID", nil)
		return
	}

	if err := h.collectionService.RemoveItem(c.Request.Context(), userID, slug, tmdbID); err != nil {
		response.HandleError(c, err)
		return
	}

	if slug == "to-watch" {
		middleware.QueueActivity(c, middleware.ActivityEvent{
			Action: middleware.ActivityDelete,
			Type:   domain.ActivityTypeWatchlistItemAdded,
			UserID: userID,
			TMDBID: &tmdbID,
		})
	} else if slug != "watched" {
		middleware.QueueActivity(c, middleware.ActivityEvent{
			Action: middleware.ActivityDelete,
			Type:   domain.ActivityTypeCollectionItemAdded,
			UserID: userID,
			TMDBID: &tmdbID,
		})
	}

	c.Status(204)
}

func toCollectionResponse(cwp ports.CollectionWithPresence) CollectionResponse {
	return CollectionResponse{
		ID:          cwp.Collection.ID.String(),
		UserID:      cwp.Collection.UserID.String(),
		Name:        cwp.Collection.Name,
		Slug:        cwp.Collection.Slug,
		Type:        string(cwp.Collection.Type),
		Visibility:  string(cwp.Collection.Visibility),
		Description: cwp.Collection.Description,
		ItemCount:   cwp.ItemCount,
		CreatedAt:   cwp.Collection.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   cwp.Collection.UpdatedAt.Format(time.RFC3339),
	}
}


type CollectionItemResponse struct {
	ID              int      `json:"id" example:"550"`
	Poster          *string  `json:"poster,omitempty" example:"/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"`
	Name            string   `json:"name" example:"Fight Club"`
	Date            string   `json:"date" example:"1999-10-15"`
	TMDBRating      *float64 `json:"tmdb_rating,omitempty" example:"4.3"`
	DuskforgeRating *float64 `json:"duskforge_rating,omitempty" example:"4.5"`
	UserRating      *float64 `json:"user_rating,omitempty" example:"4.0"`
}

type SimpleCollectionItemResponse struct {
	CollectionID string `json:"collection_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TMDBID       int    `json:"tmdb_id" example:"550"`
	AddedAt      string `json:"added_at" example:"2024-01-15T10:30:00Z"`
}

func toSimpleCollectionItemResponse(item *domain.CollectionItem) SimpleCollectionItemResponse {
	return SimpleCollectionItemResponse{
		CollectionID: item.CollectionID.String(),
		TMDBID:       item.TMDBID,
		AddedAt:      item.AddedAt.Format(time.RFC3339),
	}
}

func parseCollectionItemSort(s string) (ports.CollectionItemSort, error) {
	if s == "" {
		return ports.CollectionItemSort{Field: ports.CollectionItemSortByAddedAt, Asc: false}, nil
	}

	asc := false
	field := s
	switch s[0] {
	case '+':
		asc = true
		field = s[1:]
	case '-':
		asc = false
		field = s[1:]
	default:
		asc = true
	}

	switch ports.CollectionItemSortField(field) {
	case ports.CollectionItemSortByAddedAt,
		ports.CollectionItemSortByReleaseDate,
		ports.CollectionItemSortByTMDBRating,
		ports.CollectionItemSortByDuskforgeRating,
		ports.CollectionItemSortByOurRating,
		ports.CollectionItemSortByCollectionRating:
		return ports.CollectionItemSort{Field: ports.CollectionItemSortField(field), Asc: asc}, nil
	}

	return ports.CollectionItemSort{}, fmt.Errorf("invalid sort field: %s", field)
}
