package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yangirxd/goods-service/internal/cache"
	"github.com/yangirxd/goods-service/internal/models"
	"github.com/yangirxd/goods-service/internal/queue"
	"github.com/yangirxd/goods-service/internal/repository"
)

type GoodsHandler struct {
	repo  *repository.GoodsRepository
	cache *cache.GoodsCache
	log   *queue.Logger
}

func NewGoodsHandler(repo *repository.GoodsRepository, cache *cache.GoodsCache, log *queue.Logger) *GoodsHandler {
	return &GoodsHandler{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

// Create godoc
// @Summary      Create a new good
// @Description  Create a new good with the provided data
// @Tags         goods
// @Accept       json
// @Produce      json
// @Param        input body models.GoodCreate true "Good data"
// @Success      201 {object} models.Good
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /goods/create [post]
func (h *GoodsHandler) Create(c *gin.Context) {
	var input models.GoodCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: err.Error(),
		})
		return
	}

	good, err := h.repo.Create(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	if err := h.log.Log("create", good.ID, good); err != nil {
		println("Error logging create event:", err.Error())
	}

	c.JSON(http.StatusCreated, good)
}

// Get godoc
// @Summary      Get a good by ID
// @Description  Get a good by its ID with Redis caching
// @Tags         goods
// @Accept       json
// @Produce      json
// @Param        id path int true "Good ID"
// @Success      200 {object} models.Good
// @Failure      400 {object} models.ErrorResponse
// @Failure      404 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /goods/get/{id} [get]
func (h *GoodsHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid id",
		})
		return
	}

	cacheKey := cache.GoodKey(id)
	good, err := h.cache.Get(c.Request.Context(), cacheKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	if good == nil {
		good, err = h.repo.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    2,
				Message: "errors.internal",
				Details: err.Error(),
			})
			return
		}

		if good == nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:    3,
				Message: "errors.common.notFound",
				Details: struct{}{},
			})
			return
		}

		if err := h.cache.Set(c.Request.Context(), cacheKey, good); err != nil {
			println("Error caching good:", err.Error())
		}
	}

	c.JSON(http.StatusOK, good)
}

// Update godoc
// @Summary      Update a good
// @Description  Update a good by its ID
// @Tags         goods
// @Accept       json
// @Produce      json
// @Param        id path int true "Good ID"
// @Param        input body models.GoodUpdate true "Good update data"
// @Success      200 {object} models.Good
// @Failure      400 {object} models.ErrorResponse
// @Failure      404 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /goods/update/{id} [patch]
func (h *GoodsHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid id",
		})
		return
	}

	var input models.GoodUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: err.Error(),
		})
		return
	}

	good, err := h.repo.Update(c.Request.Context(), id, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	if good == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    3,
			Message: "errors.common.notFound",
			Details: struct{}{},
		})
		return
	}

	if err := h.cache.Delete(c.Request.Context(), cache.GoodKey(id)); err != nil {
		println("Error invalidating cache:", err.Error())
	}

	if err := h.log.Log("update", id, good); err != nil {
		println("Error logging update event:", err.Error())
	}

	c.JSON(http.StatusOK, good)
}

// Delete godoc
// @Summary      Delete a good
// @Description  Mark a good as deleted by its ID
// @Tags         goods
// @Accept       json
// @Produce      json
// @Param        id path int true "Good ID"
// @Success      204 "No Content"
// @Failure      400 {object} models.ErrorResponse
// @Failure      404 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /goods/delete/{id} [delete]
func (h *GoodsHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid id",
		})
		return
	}

	good, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	if good == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    3,
			Message: "errors.common.notFound",
			Details: struct{}{},
		})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	if err := h.cache.Delete(c.Request.Context(), cache.GoodKey(id)); err != nil {
		println("Error invalidating cache:", err.Error())
	}

	if err := h.log.Log("delete", id, nil); err != nil {
		println("Error logging delete event:", err.Error())
	}

	c.Status(http.StatusNoContent)
}

// List godoc
// @Summary      List goods
// @Description  Get list of goods with pagination
// @Tags         goods
// @Accept       json
// @Produce      json
// @Param        limit query int false "Limit number of records (default: 10)"
// @Param        offset query int false "Offset for pagination (default: 0)"
// @Success      200 {object} models.ListResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /goods/list [get]
func (h *GoodsHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid limit",
		})
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid offset",
		})
		return
	}

	goods, total, removed, err := h.repo.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	goodsResponse := make([]models.Good, len(goods))
	for i, g := range goods {
		goodsResponse[i] = *g
	}

	response := models.ListResponse{
		Meta: models.ListMeta{
			Total:   total,
			Removed: removed,
			Limit:   limit,
			Offset:  offset,
		},
		Goods: goodsResponse,
	}

	c.JSON(http.StatusOK, response)
}

// Reprioritize godoc
// @Summary      Reprioritize a good
// @Description  Change priority of a good and update priorities of subsequent goods
// @Tags         goods
// @Accept       json
// @Produce      json
// @Param        id query int true "Good ID"
// @Param        projectId query int true "Project ID"
// @Param        input body models.ReprioritizeRequest true "New priority"
// @Success      200 {object} models.ReprioritizeResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      404 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /goods/reprioritize [patch]
func (h *GoodsHandler) Reprioritize(c *gin.Context) {
	id, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid id",
		})
		return
	}

	projectID, err := strconv.ParseInt(c.Query("projectId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: "invalid project_id",
		})
		return
	}

	var input models.ReprioritizeRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    1,
			Message: "errors.validation.failed",
			Details: err.Error(),
		})
		return
	}

	updatedGoods, err := h.repo.Reprioritize(c.Request.Context(), id, projectID, input.NewPriority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    2,
			Message: "errors.internal",
			Details: err.Error(),
		})
		return
	}

	if updatedGoods == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    3,
			Message: "errors.common.notFound",
			Details: struct{}{},
		})
		return
	}

	for _, good := range updatedGoods {
		if err := h.cache.Delete(c.Request.Context(), cache.GoodKey(good.ID)); err != nil {
			println("Error invalidating cache:", err.Error())
		}
	}

	if err := h.log.Log("reprioritize", id, map[string]interface{}{
		"project_id":   projectID,
		"new_priority": input.NewPriority,
		"updated_ids":  updatedGoods,
	}); err != nil {
		println("Error logging reprioritize event:", err.Error())
	}

	priorities := make([]models.PriorityInfo, len(updatedGoods))
	for i, good := range updatedGoods {
		priorities[i] = models.PriorityInfo{
			ID:       good.ID,
			Priority: good.Priority,
		}
	}

	c.JSON(http.StatusOK, models.ReprioritizeResponse{
		Priorities: priorities,
	})
}
