package handler

import (
	"net/http"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	"github.com/fumkob/ezqrin-server/internal/usecase/event"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
)

const (
	defaultPerPage = 20
)

// EventHandler handles event-related endpoints.
// Implements generated.ServerInterface for OpenAPI compliance.
type EventHandler struct {
	usecase event.Usecase
	logger  *logger.Logger
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(usecase event.Usecase, logger *logger.Logger) *EventHandler {
	return &EventHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// GetEvents handles listing events (GET /events).
func (h *EventHandler) GetEvents(c *gin.Context, params generated.GetEventsParams) {
	var organizerID *uuid.UUID
	role := h.getUserRole(c)
	userID := h.getUserID(c)

	// If not admin, only show own events
	if role != string(entity.RoleAdmin) {
		organizerID = &userID
	}

	input := event.ListEventsInput{
		OrganizerID: organizerID,
		Search:      "",
		Page:        1,
		PerPage:     defaultPerPage,
	}

	if params.Name != nil {
		input.Search = *params.Name
	}
	if params.Page != nil {
		input.Page = int(*params.Page)
	}
	if params.PerPage != nil {
		input.PerPage = int(*params.PerPage)
	}
	if params.Status != nil {
		status := entity.EventStatus(*params.Status)
		input.Status = &status
	}
	if params.Sort != nil {
		input.Sort = string(*params.Sort)
	}
	if params.Order != nil {
		input.Order = string(*params.Order)
	}

	output, err := h.usecase.List(c.Request.Context(), input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	events := make([]generated.Event, len(output.Events))
	for i, e := range output.Events {
		events[i] = h.toGeneratedEvent(e)
	}

	resp := generated.EventListResponse{
		Data: events,
		Meta: generated.PaginationMeta{
			Page:       input.Page,
			PerPage:    input.PerPage,
			Total:      int(output.TotalCount),
			TotalPages: int((output.TotalCount + int64(input.PerPage) - 1) / int64(input.PerPage)),
		},
	}

	response.Data(c, http.StatusOK, resp)
}

// PostEvents handles event creation (POST /events).
func (h *EventHandler) PostEvents(c *gin.Context) {
	var req generated.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	userID := h.getUserID(c)

	input := event.CreateEventInput{
		OrganizerID: userID,
		Name:        req.Name,
		StartDate:   req.StartDate,
		Status:      entity.EventStatus(req.Status),
	}

	if req.Description != nil {
		input.Description = *req.Description
	}
	if req.EndDate != nil {
		input.EndDate = req.EndDate
	}
	if req.Location != nil {
		input.Location = *req.Location
	}
	if req.Timezone != nil {
		input.Timezone = *req.Timezone
	}

	evt, err := h.usecase.Create(c.Request.Context(), input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.Data(c, http.StatusCreated, h.toGeneratedEvent(evt))
}

// GetEventsId handles getting event details (GET /events/{id}).
func (h *EventHandler) GetEventsId(c *gin.Context, id generated.EventIDParam) {
	eventID := uuid.UUID(id)

	evt, err := h.usecase.GetByID(c.Request.Context(), eventID)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Authorization check: only owner or admin can view details (unless public)
	// For now, let's assume all authenticated users can view, but only owner/admin see organizer details
	// The requirement says "Organizers see their own events, admins see all" for List.
	// For detail, let's keep it consistent.
	role := h.getUserRole(c)
	userID := h.getUserID(c)
	if role != string(entity.RoleAdmin) && evt.OrganizerID != userID {
		response.ProblemFromError(c, apperrors.Forbidden("you do not have permission to view this event"))
		return
	}

	response.Data(c, http.StatusOK, h.toGeneratedEvent(evt))
}

// PutEventsId handles event update (PUT /events/{id}).
func (h *EventHandler) PutEventsId(c *gin.Context, id generated.EventIDParam) {
	eventID := uuid.UUID(id)

	var req generated.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	role := h.getUserRole(c)
	userID := h.getUserID(c)

	input := event.UpdateEventInput{}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.StartDate != nil {
		input.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		input.EndDate = req.EndDate
	}
	if req.Location != nil {
		input.Location = req.Location
	}
	if req.Timezone != nil {
		input.Timezone = req.Timezone
	}
	if req.Status != nil {
		status := entity.EventStatus(*req.Status)
		input.Status = &status
	}

	evt, err := h.usecase.Update(c.Request.Context(), eventID, userID, role == string(entity.RoleAdmin), input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.Data(c, http.StatusOK, h.toGeneratedEvent(evt))
}

// DeleteEventsId handles event deletion (DELETE /events/{id}).
func (h *EventHandler) DeleteEventsId(c *gin.Context, id generated.EventIDParam) {
	eventID := uuid.UUID(id)

	role := h.getUserRole(c)
	userID := h.getUserID(c)

	err := h.usecase.Delete(c.Request.Context(), eventID, userID, role == string(entity.RoleAdmin))
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetEventsIdStats handles getting event statistics (GET /events/{id}/stats).
func (h *EventHandler) GetEventsIdStats(c *gin.Context, id generated.EventIDParam) {
	eventID := uuid.UUID(id)

	role := h.getUserRole(c)
	userID := h.getUserID(c)

	output, err := h.usecase.GetStats(c.Request.Context(), eventID, userID, role == string(entity.RoleAdmin))
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	eventIDStr := openapi_types.UUID(output.EventID)
	byStatus := make(map[string]int) // Placeholder
	resp := generated.EventStatsResponse{
		EventId:               eventIDStr,
		TotalParticipants:     int(output.TotalParticipants),
		CheckedInParticipants: int(output.CheckedInParticipants),
		CheckinRate:           float32(output.CheckinRate),
		ByStatus:              &byStatus,
	}

	response.Data(c, http.StatusOK, resp)
}

// Helpers

func (h *EventHandler) getUserID(c *gin.Context) uuid.UUID {
	val, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		return uuid.Nil
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

func (h *EventHandler) getUserRole(c *gin.Context) string {
	val, exists := c.Get(middleware.ContextKeyUserRole)
	if !exists {
		return ""
	}
	role, ok := val.(string)
	if !ok {
		return ""
	}
	return role
}

func (h *EventHandler) toGeneratedEvent(e *entity.Event) generated.Event {
	id := openapi_types.UUID(e.ID)
	organizerID := openapi_types.UUID(e.OrganizerID)

	genEvent := generated.Event{
		Id:          &id,
		OrganizerId: &organizerID,
		Name:        e.Name,
		StartDate:   e.StartDate,
		Status:      generated.EventStatus(e.Status),
		CreatedAt:   &e.CreatedAt,
		UpdatedAt:   &e.UpdatedAt,
	}

	if e.Description != "" {
		desc := e.Description
		genEvent.Description = &desc
	}
	if e.EndDate != nil {
		genEvent.EndDate = e.EndDate
	}
	if e.Location != "" {
		loc := e.Location
		genEvent.Location = &loc
	}
	if e.Timezone != "" {
		tz := e.Timezone
		genEvent.Timezone = &tz
	}

	return genEvent
}
