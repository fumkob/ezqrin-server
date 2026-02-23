package handler

import (
	"net/http"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	"github.com/fumkob/ezqrin-server/internal/usecase/checkin"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
)

const (
	defaultCheckinPerPage = 20
)

// CheckinHandler handles check-in-related endpoints.
// Implements generated.ServerInterface for OpenAPI compliance.
type CheckinHandler struct {
	usecase checkin.Usecase
	logger  *logger.Logger
}

// NewCheckinHandler creates a new CheckinHandler
func NewCheckinHandler(usecase checkin.Usecase, logger *logger.Logger) *CheckinHandler {
	return &CheckinHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// CheckInParticipant handles participant check-in (POST /events/{id}/checkin).
func (h *CheckinHandler) CheckInParticipant(c *gin.Context, eventID generated.EventIDParam) {
	var req generated.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	// Convert request to usecase input
	input := checkin.CheckInInput{
		EventID:     uuid.UUID(eventID),
		Method:      entity.CheckinMethod(req.Method),
		CheckedInBy: userID,
	}

	// Set QR code or participant ID based on method
	if err := h.setCheckinMethodFields(&input, &req); err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Set device info if provided
	if req.DeviceInfo != nil {
		input.DeviceInfo = *req.DeviceInfo
	}

	// Execute check-in
	result, err := h.usecase.CheckIn(c.Request.Context(), userID, isAdmin, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Convert to response
	resp := h.toCheckInResponse(result)
	response.Data(c, http.StatusOK, resp)
}

// ListCheckIns handles listing check-ins for an event (GET /events/{id}/checkins).
func (h *CheckinHandler) ListCheckIns(
	c *gin.Context,
	eventID generated.EventIDParam,
	params generated.ListCheckInsParams,
) {
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	input := h.buildListCheckInsInput(eventID, params)

	output, err := h.usecase.List(c.Request.Context(), userID, isAdmin, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	resp := h.buildCheckInListResponse(output, input)
	response.Data(c, http.StatusOK, resp)
}

// GetCheckInStatus handles getting check-in status for a participant (GET /participants/{id}/checkin-status).
func (h *CheckinHandler) GetCheckInStatus(c *gin.Context, participantID generated.ParticipantIDParam) {
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	output, err := h.usecase.GetStatus(c.Request.Context(), userID, isAdmin, uuid.UUID(participantID))
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Convert to response
	resp := h.buildCheckInStatusResponse(output)
	response.Data(c, http.StatusOK, resp)
}

// CancelCheckIn handles canceling a check-in (DELETE /events/{id}/checkins/{cid}).
func (h *CheckinHandler) CancelCheckIn(c *gin.Context, id generated.EventIDParam, cid openapi_types.UUID) {
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	err := h.usecase.Cancel(c.Request.Context(), userID, isAdmin, uuid.UUID(cid))
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.NoContent(c)
}

// Helper functions

func (h *CheckinHandler) getUserID(c *gin.Context) uuid.UUID {
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

func (h *CheckinHandler) getUserRole(c *gin.Context) string {
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

func (h *CheckinHandler) toCheckInResponse(output *checkin.CheckInOutput) generated.CheckInResponse {
	resp := generated.CheckInResponse{
		Id:            openapi_types.UUID(output.ID),
		EventId:       openapi_types.UUID(output.EventID),
		ParticipantId: openapi_types.UUID(output.ParticipantID),
		CheckinMethod: generated.CheckInMethod(output.Method),
		CheckedInAt:   output.CheckedInAt,
		Message:       "Check-in successful",
		Participant: struct {
			Email openapi_types.Email `json:"email"`
			Name  string              `json:"name"`
		}{
			Name:  output.ParticipantName,
			Email: openapi_types.Email(output.ParticipantEmail),
		},
	}

	// Set CheckedInBy if present (for manual check-ins)
	if output.CheckedInBy != nil {
		resp.CheckedInBy.Id = openapi_types.UUID(*output.CheckedInBy)
		// TODO: Get user name from user repository
		resp.CheckedInBy.Name = "Staff"
	}

	return resp
}

// setCheckinMethodFields sets QR code or participant ID based on check-in method
func (h *CheckinHandler) setCheckinMethodFields(
	input *checkin.CheckInInput,
	req *generated.CheckInRequest,
) error {
	if req.Method == generated.Qrcode {
		if req.QrCode == nil || *req.QrCode == "" {
			return apperrors.BadRequest("qr_code is required for QR code check-in")
		}
		input.QRCode = req.QrCode
		return nil
	}

	if req.Method == generated.Manual {
		if req.ParticipantId == nil {
			return apperrors.BadRequest("participant_id is required for manual check-in")
		}
		participantID := uuid.UUID(*req.ParticipantId)
		input.ParticipantID = &participantID
		return nil
	}

	return nil
}

// buildListCheckInsInput builds the list input from parameters
func (h *CheckinHandler) buildListCheckInsInput(
	eventID generated.EventIDParam,
	params generated.ListCheckInsParams,
) checkin.ListCheckInsInput {
	input := checkin.ListCheckInsInput{
		EventID: uuid.UUID(eventID),
		Page:    1,
		PerPage: defaultCheckinPerPage,
		Sort:    "checked_in_at",
		Order:   "desc",
	}

	if params.Page != nil {
		input.Page = int(*params.Page)
	}
	if params.PerPage != nil {
		input.PerPage = int(*params.PerPage)
	}
	if params.Sort != nil {
		input.Sort = string(*params.Sort)
	}
	if params.Order != nil {
		input.Order = string(*params.Order)
	}

	return input
}

// buildCheckInListResponse builds the check-in list response
func (h *CheckinHandler) buildCheckInListResponse(
	output *checkin.ListCheckInsOutput,
	input checkin.ListCheckInsInput,
) generated.CheckInListResponse {
	checkinItems := h.convertCheckInsToItems(output.CheckIns)

	return generated.CheckInListResponse{
		Checkins: checkinItems,
		Pagination: generated.PaginationMeta{
			Page:       input.Page,
			PerPage:    input.PerPage,
			Total:      int(output.TotalCount),
			TotalPages: h.calculateTotalPages(output.TotalCount, input.PerPage),
		},
	}
}

// convertCheckInsToItems converts check-in outputs to response items
func (h *CheckinHandler) convertCheckInsToItems(checkIns []*checkin.CheckInOutput) []struct {
	CheckedInAt time.Time `json:"checked_in_at"`
	CheckedInBy struct {
		Id   openapi_types.UUID `json:"id"`
		Name string             `json:"name"`
	} `json:"checked_in_by"`
	CheckinMethod generated.CheckInMethod `json:"checkin_method"`
	DeviceInfo    *map[string]any         `json:"device_info,omitempty"`
	EventId       openapi_types.UUID      `json:"event_id"`
	Id            openapi_types.UUID      `json:"id"`
	Participant   struct {
		Email      openapi_types.Email `json:"email"`
		EmployeeId *string             `json:"employee_id"`
		Name       string              `json:"name"`
	} `json:"participant"`
	ParticipantId openapi_types.UUID `json:"participant_id"`
} {
	items := make([]struct {
		CheckedInAt time.Time `json:"checked_in_at"`
		CheckedInBy struct {
			Id   openapi_types.UUID `json:"id"`
			Name string             `json:"name"`
		} `json:"checked_in_by"`
		CheckinMethod generated.CheckInMethod `json:"checkin_method"`
		DeviceInfo    *map[string]any         `json:"device_info,omitempty"`
		EventId       openapi_types.UUID      `json:"event_id"`
		Id            openapi_types.UUID      `json:"id"`
		Participant   struct {
			Email      openapi_types.Email `json:"email"`
			EmployeeId *string             `json:"employee_id"`
			Name       string              `json:"name"`
		} `json:"participant"`
		ParticipantId openapi_types.UUID `json:"participant_id"`
	}, len(checkIns))

	for i, ci := range checkIns {
		items[i].Id = openapi_types.UUID(ci.ID)
		items[i].EventId = openapi_types.UUID(ci.EventID)
		items[i].ParticipantId = openapi_types.UUID(ci.ParticipantID)
		items[i].Participant.Name = ci.ParticipantName
		items[i].Participant.Email = openapi_types.Email(ci.ParticipantEmail)
		items[i].Participant.EmployeeId = ci.ParticipantEmployeeID
		items[i].CheckedInAt = ci.CheckedInAt
		items[i].CheckinMethod = generated.CheckInMethod(ci.Method)

		// Set CheckedInBy if present (for manual check-ins)
		if ci.CheckedInBy != nil {
			items[i].CheckedInBy.Id = openapi_types.UUID(*ci.CheckedInBy)
			items[i].CheckedInBy.Name = "Staff"
		}
	}

	return items
}

// calculateTotalPages calculates total pages for pagination
func (h *CheckinHandler) calculateTotalPages(totalCount int64, perPage int) int {
	return int((totalCount + int64(perPage) - 1) / int64(perPage))
}

// buildCheckInStatusResponse builds the check-in status response
func (h *CheckinHandler) buildCheckInStatusResponse(
	output *checkin.CheckInStatusOutput,
) generated.CheckInStatusResponse {
	resp := generated.CheckInStatusResponse{
		ParticipantId:   openapi_types.UUID(output.ParticipantID),
		ParticipantName: output.ParticipantName,
		EventId:         openapi_types.UUID(output.EventID),
		EventName:       output.EventName,
		CheckedIn:       output.IsCheckedIn,
		Checkin:         nil,
	}

	if output.ParticipantEmail != "" {
		email := openapi_types.Email(output.ParticipantEmail)
		resp.ParticipantEmail = &email
	}

	if output.CheckIn != nil {
		checkinDetail := struct {
			CheckedInAt time.Time `json:"checked_in_at"`
			CheckedInBy struct {
				Id   openapi_types.UUID `json:"id"`
				Name string             `json:"name"`
			} `json:"checked_in_by"`
			CheckinMethod generated.CheckInMethod `json:"checkin_method"`
			DeviceInfo    *map[string]any         `json:"device_info,omitempty"`
			Id            openapi_types.UUID      `json:"id"`
		}{
			Id:            openapi_types.UUID(output.CheckIn.ID),
			CheckedInAt:   output.CheckIn.CheckedInAt,
			CheckinMethod: generated.CheckInMethod(output.CheckIn.Method),
		}

		// Set CheckedInBy if present (for manual check-ins)
		if output.CheckIn.CheckedInBy != nil {
			checkinDetail.CheckedInBy.Id = openapi_types.UUID(*output.CheckIn.CheckedInBy)
			checkinDetail.CheckedInBy.Name = "Staff"
		}

		resp.Checkin = &checkinDetail
	}

	return resp
}
