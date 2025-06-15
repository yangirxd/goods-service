package models

import "time"

// ListMeta содержит мета-информацию для списка
type ListMeta struct {
	Total   int `json:"total"`
	Removed int `json:"removed"`
	Limit   int `json:"limit"`
	Offset  int `json:"offset"`
}

// ListResponse представляет ответ со списком товаров
type ListResponse struct {
	Meta  ListMeta `json:"meta"`
	Goods []Good   `json:"goods"`
}

type Good struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Priority    int       `json:"priority"`
	Removed     bool      `json:"removed"`
	CreatedAt   time.Time `json:"created_at"`
}

type GoodCreate struct {
	ProjectID   int64  `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type GoodUpdate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

// ReprioritizeRequest представляет запрос на изменение приоритета
type ReprioritizeRequest struct {
	NewPriority int `json:"newPriority" binding:"required"`
}

// ReprioritizeResponse представляет ответ с обновлёнными приоритетами
type ReprioritizeResponse struct {
	Priorities []PriorityInfo `json:"priorities"`
}

// PriorityInfo содержит информацию о приоритете товара
type PriorityInfo struct {
	ID       int64 `json:"id"`
	Priority int   `json:"priority"`
}
