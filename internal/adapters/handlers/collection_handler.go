package handlers

import (
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
}

func NewCollectionHandler(collectionService ports.CollectionService) *CollectionHandler {
	return &CollectionHandler{collectionService: collectionService}
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
	TMDBID  int   `json:"tmdb_id" binding:"required" example:"550"`
	Runtime int16 `json:"runtime" example:"139"`
}

type CollectionResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID      string  `json:"user_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Name        string  `json:"name" example:"My Favorites"`
	Slug        string  `json:"slug" example:"my-favorites"`
	Type        string  `json:"type" example:"custom"`
	Visibility  string  `json:"visibility" example:"private"`
	Description *string `json:"description,omitempty" example:"A collection of my favorite movies"`
	CreatedAt   string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type CollectionItemResponse struct {
	CollectionID string `json:"collection_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TMDBID       int    `json:"tmdb_id" example:"550"`
	AddedAt      string `json:"added_at" example:"2024-01-15T10:30:00Z"`
	Runtime      int16  `json:"runtime" example:"139"`
}

// @Summary      Create a collection
// @Description  Create a new custom collection for the authenticated user
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateCollectionRequest true "Collection details"
// @Success      201 {object} response.Response{data=CollectionResponse} "Collection created"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      409 {object} response.Response "Collection with this name already exists"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections [post]
func (h *CollectionHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
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

	response.Created(c, toCollectionResponse(collection))
}

// @Summary      Get collection by ID
// @Description  Get a collection by its ID. Returns the collection if public or if the requester is the owner.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Collection ID" format(uuid)
// @Success      200 {object} response.Response{data=CollectionResponse} "Collection details"
// @Failure      400 {object} response.Response "Invalid collection ID"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections/{id} [get]
func (h *CollectionHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collection ID", nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	collection, err := h.collectionService.GetByID(c.Request.Context(), id, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toCollectionResponse(collection))
}

// @Summary      Get user's collections
// @Description  Get all collections for a user. Returns all collections if the requester is the owner, only public ones otherwise.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} response.Response{data=[]CollectionResponse} "List of collections"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{id}/collections [get]
func (h *CollectionHandler) GetByUserID(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	collections, err := h.collectionService.GetByUserID(c.Request.Context(), userID, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]CollectionResponse, len(collections))
	for i, col := range collections {
		resp[i] = toCollectionResponse(col)
	}

	response.Success(c, resp)
}

// @Summary      Update a collection
// @Description  Update a collection's name, description, or visibility. System collections cannot have their name changed.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Collection ID" format(uuid)
// @Param        request body UpdateCollectionRequest true "Fields to update"
// @Success      200 {object} response.Response{data=CollectionResponse} "Updated collection"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Cannot modify system collection"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      409 {object} response.Response "Collection with this name already exists"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections/{id} [patch]
func (h *CollectionHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collection ID", nil)
		return
	}

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

	collection, err := h.collectionService.Update(c.Request.Context(), collectionID, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toCollectionResponse(collection))
}

// @Summary      Delete a collection
// @Description  Delete a collection. System collections cannot be deleted.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Collection ID" format(uuid)
// @Success      204 "Collection deleted"
// @Failure      400 {object} response.Response "Invalid collection ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Cannot delete system collection"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections/{id} [delete]
func (h *CollectionHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collection ID", nil)
		return
	}

	if err := h.collectionService.Delete(c.Request.Context(), collectionID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Add item to collection
// @Description  Add a movie to a collection by TMDB ID
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Collection ID" format(uuid)
// @Param        request body AddItemRequest true "Movie to add"
// @Success      201 {object} response.Response{data=CollectionItemResponse} "Item added"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Not the collection owner"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      409 {object} response.Response "Item already in collection"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections/{id}/items [post]
func (h *CollectionHandler) AddItem(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collection ID", nil)
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	item, err := h.collectionService.AddItem(c.Request.Context(), collectionID, userID, req.TMDBID, req.Runtime)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, toCollectionItemResponse(item))
}

// @Summary      Get collection items
// @Description  Get all items in a collection. Respects visibility rules.
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Collection ID" format(uuid)
// @Success      200 {object} response.Response{data=[]CollectionItemResponse} "List of items"
// @Failure      400 {object} response.Response "Invalid collection ID"
// @Failure      404 {object} response.Response "Collection not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections/{id}/items [get]
func (h *CollectionHandler) GetItems(c *gin.Context) {
	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collection ID", nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	items, err := h.collectionService.GetItems(c.Request.Context(), collectionID, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]CollectionItemResponse, len(items))
	for i, item := range items {
		resp[i] = toCollectionItemResponse(item)
	}

	response.Success(c, resp)
}

// @Summary      Remove item from collection
// @Description  Remove a movie from a collection by TMDB ID
// @Tags         collections
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Collection ID" format(uuid)
// @Param        tmdbId path int true "TMDB movie ID"
// @Success      204 "Item removed"
// @Failure      400 {object} response.Response "Invalid collection or TMDB ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Not the collection owner"
// @Failure      404 {object} response.Response "Collection or item not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /collections/{id}/items/{tmdbId} [delete]
func (h *CollectionHandler) RemoveItem(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collection ID", nil)
		return
	}

	tmdbIDStr := c.Param("tmdbId")
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid TMDB ID", nil)
		return
	}

	if err := h.collectionService.RemoveItem(c.Request.Context(), collectionID, userID, tmdbID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

func toCollectionResponse(collection *domain.Collection) CollectionResponse {
	return CollectionResponse{
		ID:          collection.ID.String(),
		UserID:      collection.UserID.String(),
		Name:        collection.Name,
		Slug:        collection.Slug,
		Type:        string(collection.Type),
		Visibility:  string(collection.Visibility),
		Description: collection.Description,
		CreatedAt:   collection.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   collection.UpdatedAt.Format(time.RFC3339),
	}
}

func toCollectionItemResponse(item *domain.CollectionItem) CollectionItemResponse {
	return CollectionItemResponse{
		CollectionID: item.CollectionID.String(),
		TMDBID:       item.TMDBID,
		AddedAt:      item.AddedAt.Format(time.RFC3339),
		Runtime:      item.Runtime,
	}
}
