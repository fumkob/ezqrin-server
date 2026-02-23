package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
)

// ParticipantHandler handles participant-related endpoints.
// Implements generated.ServerInterface for OpenAPI compliance.
type ParticipantHandler struct {
	usecase participant.Usecase
	logger  *logger.Logger
}

// NewParticipantHandler creates a new ParticipantHandler
func NewParticipantHandler(usecase participant.Usecase, logger *logger.Logger) *ParticipantHandler {
	return &ParticipantHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// CreateParticipant handles participant creation (POST /events/{id}/participants).
func (h *ParticipantHandler) CreateParticipant(c *gin.Context, eventID generated.EventIDParam) {
	var req generated.CreateParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	// Convert request to usecase input
	status := entity.ParticipantStatusTentative
	if req.Status != nil {
		status = entity.ParticipantStatus(*req.Status)
	}

	input := participant.CreateParticipantInput{
		EventID:       uuid.UUID(eventID),
		Name:          req.Name,
		Email:         string(req.Email),
		QREmail:       convertEmailPtr(req.QrEmail),
		EmployeeID:    req.EmployeeId,
		Phone:         req.Phone,
		Status:        status,
		Metadata:      convertMetadataToString(req.Metadata),
		PaymentStatus: entity.PaymentStatus(ptrOrDefault(req.PaymentStatus, "unpaid")),
		PaymentAmount: req.PaymentAmount,
		PaymentDate:   req.PaymentDate,
	}

	p, err := h.usecase.Create(c.Request.Context(), userID, isAdmin, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.Data(c, http.StatusCreated, h.toGeneratedParticipant(p))
}

// BulkCreateParticipants handles bulk participant creation (POST /events/{id}/participants/bulk).
func (h *ParticipantHandler) BulkCreateParticipants(c *gin.Context, eventID generated.EventIDParam) {
	var req generated.BulkCreateParticipantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	// Convert request to usecase input
	input := h.convertBulkCreateRequest(req, eventID)

	output, err := h.usecase.BulkCreate(c.Request.Context(), userID, isAdmin, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Convert to response
	bulkResp := h.convertBulkCreateResponse(output)
	response.Data(c, http.StatusCreated, bulkResp)
}

// ListParticipants handles listing participants (GET /events/{id}/participants).
func (h *ParticipantHandler) ListParticipants(
	c *gin.Context,
	eventID generated.EventIDParam,
	params generated.ListParticipantsParams,
) {
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	input := participant.ListParticipantsInput{
		EventID: uuid.UUID(eventID),
		Page:    1,
		PerPage: defaultPerPage,
		Sort:    "created_at",
		Order:   "desc",
	}

	if params.Page != nil {
		input.Page = int(*params.Page)
	}
	if params.PerPage != nil {
		input.PerPage = int(*params.PerPage)
	}
	if params.Search != nil {
		input.Search = *params.Search
	}
	if params.Status != nil {
		status := entity.ParticipantStatus(*params.Status)
		input.Status = &status
	}
	if params.Sort != nil {
		input.Sort = *params.Sort
	}
	if params.Order != nil {
		input.Order = string(*params.Order)
	}

	output, err := h.usecase.List(c.Request.Context(), userID, isAdmin, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	participants := make([]generated.Participant, len(output.Participants))
	for i, p := range output.Participants {
		participants[i] = h.toGeneratedParticipant(p)
	}

	resp := generated.ParticipantListResponse{
		Data: participants,
		Meta: generated.PaginationMeta{
			Page:       input.Page,
			PerPage:    input.PerPage,
			Total:      int(output.TotalCount),
			TotalPages: int((output.TotalCount + int64(input.PerPage) - 1) / int64(input.PerPage)),
		},
	}

	response.Data(c, http.StatusOK, resp)
}

// GetParticipant handles getting participant details (GET /participants/{id}).
func (h *ParticipantHandler) GetParticipant(c *gin.Context, id generated.ParticipantIDParam) {
	participantID := uuid.UUID(id)
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	p, err := h.usecase.GetByID(c.Request.Context(), userID, isAdmin, participantID)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.Data(c, http.StatusOK, h.toGeneratedParticipant(p))
}

// UpdateParticipant handles participant update (PUT /participants/{id}).
func (h *ParticipantHandler) UpdateParticipant(c *gin.Context, id generated.ParticipantIDParam) {
	participantID := uuid.UUID(id)

	var req generated.UpdateParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	// Convert request to usecase input
	input := participant.UpdateParticipantInput{
		Name:          req.Name,
		Email:         convertEmailPtrToStringPtr(req.Email),
		QREmail:       convertEmailPtr(req.QrEmail),
		EmployeeID:    req.EmployeeId,
		Phone:         req.Phone,
		Metadata:      convertMetadataToString(req.Metadata),
		PaymentAmount: req.PaymentAmount,
		PaymentDate:   req.PaymentDate,
	}

	if req.Status != nil {
		status := entity.ParticipantStatus(*req.Status)
		input.Status = &status
	}
	if req.PaymentStatus != nil {
		paymentStatus := entity.PaymentStatus(*req.PaymentStatus)
		input.PaymentStatus = &paymentStatus
	}

	p, err := h.usecase.Update(c.Request.Context(), userID, isAdmin, participantID, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.Data(c, http.StatusOK, h.toGeneratedParticipant(p))
}

// DeleteParticipant handles participant deletion (DELETE /participants/{id}).
func (h *ParticipantHandler) DeleteParticipant(c *gin.Context, id generated.ParticipantIDParam) {
	participantID := uuid.UUID(id)
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	err := h.usecase.Delete(c.Request.Context(), userID, isAdmin, participantID)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	response.NoContent(c)
}

// DownloadParticipantQRCode handles QR code download (GET /participants/{id}/qrcode).
func (h *ParticipantHandler) DownloadParticipantQRCode(
	c *gin.Context,
	id generated.ParticipantIDParam,
	params generated.DownloadParticipantQRCodeParams,
) {
	participantID := uuid.UUID(id)
	userID := h.getUserID(c)
	isAdmin := h.getUserRole(c) == string(entity.RoleAdmin)

	// Parse parameters with defaults
	format := "png"
	if params.Format != nil {
		format = string(*params.Format)
	}

	size := 512
	if params.Size != nil {
		size = *params.Size
	}

	qr, err := h.usecase.GetQRCode(c.Request.Context(), userID, isAdmin, participantID, format, size)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	c.Header("Content-Type", qr.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", qr.Filename))
	c.Data(http.StatusOK, qr.ContentType, qr.Data)
}

// Helper functions

// convertBulkCreateRequest converts API request to usecase input
func (h *ParticipantHandler) convertBulkCreateRequest(
	req generated.BulkCreateParticipantsRequest,
	eventID generated.EventIDParam,
) participant.BulkCreateInput {
	participants := make([]participant.CreateParticipantInput, len(req.Participants))
	for i, p := range req.Participants {
		participants[i] = h.convertToCreateInput(p, eventID)
	}

	return participant.BulkCreateInput{
		EventID:      uuid.UUID(eventID),
		Participants: participants,
	}
}

// convertToCreateInput converts single participant request to create input
func (h *ParticipantHandler) convertToCreateInput(
	p generated.CreateParticipantRequest,
	eventID generated.EventIDParam,
) participant.CreateParticipantInput {
	status := entity.ParticipantStatusTentative
	if p.Status != nil {
		status = entity.ParticipantStatus(*p.Status)
	}

	return participant.CreateParticipantInput{
		EventID:       uuid.UUID(eventID),
		Name:          p.Name,
		Email:         string(p.Email),
		QREmail:       convertEmailPtr(p.QrEmail),
		EmployeeID:    p.EmployeeId,
		Phone:         p.Phone,
		Status:        status,
		Metadata:      convertMetadataToString(p.Metadata),
		PaymentStatus: entity.PaymentStatus(ptrOrDefault(p.PaymentStatus, "unpaid")),
		PaymentAmount: p.PaymentAmount,
		PaymentDate:   p.PaymentDate,
	}
}

// convertBulkCreateResponse converts usecase output to API response
func (h *ParticipantHandler) convertBulkCreateResponse(
	output participant.BulkCreateOutput,
) generated.BulkCreateParticipantsResponse {
	createdParticipants := make([]generated.Participant, len(output.Participants))
	for i, p := range output.Participants {
		createdParticipants[i] = h.toGeneratedParticipant(p)
	}

	bulkErrors := h.convertBulkErrors(output.Errors)

	return generated.BulkCreateParticipantsResponse{
		CreatedCount: output.CreatedCount,
		FailedCount:  output.FailedCount,
		Participants: createdParticipants,
		Errors:       &bulkErrors,
	}
}

// convertBulkErrors converts bulk create errors to API format
func (h *ParticipantHandler) convertBulkErrors(errors []participant.BulkCreateError) []struct {
	Email openapi_types.Email `json:"email"`
	Error string              `json:"error"`
	Index int                 `json:"index"`
} {
	bulkErrors := make([]struct {
		Email openapi_types.Email `json:"email"`
		Error string              `json:"error"`
		Index int                 `json:"index"`
	}, len(errors))

	for i, e := range errors {
		bulkErrors[i] = struct {
			Email openapi_types.Email `json:"email"`
			Error string              `json:"error"`
			Index int                 `json:"index"`
		}{
			Email: openapi_types.Email(e.Email),
			Error: e.Message,
			Index: e.Index,
		}
	}

	return bulkErrors
}

func (h *ParticipantHandler) getUserID(c *gin.Context) uuid.UUID {
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

func (h *ParticipantHandler) getUserRole(c *gin.Context) string {
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

func (h *ParticipantHandler) toGeneratedParticipant(p *entity.Participant) generated.Participant {
	id := openapi_types.UUID(p.ID)
	eventID := openapi_types.UUID(p.EventID)
	email := openapi_types.Email(p.Email)

	genParticipant := generated.Participant{
		Id:        &id,
		EventId:   &eventID,
		Name:      p.Name,
		Email:     email,
		Status:    generated.ParticipantStatus(p.Status),
		CreatedAt: &p.CreatedAt,
		UpdatedAt: &p.UpdatedAt,
	}

	if p.QREmail != nil {
		qrEmail := openapi_types.Email(*p.QREmail)
		genParticipant.QrEmail = &qrEmail
	}
	if p.EmployeeID != nil {
		genParticipant.EmployeeId = p.EmployeeID
	}
	if p.Phone != nil {
		genParticipant.Phone = p.Phone
	}
	if p.Metadata != nil {
		metadata := convertRawMessageToMap(p.Metadata)
		genParticipant.Metadata = &metadata
	}

	paymentStatus := generated.PaymentStatus(p.PaymentStatus)
	genParticipant.PaymentStatus = &paymentStatus
	if p.PaymentAmount != nil {
		genParticipant.PaymentAmount = p.PaymentAmount
	}
	if p.PaymentDate != nil {
		genParticipant.PaymentDate = p.PaymentDate
	}

	genParticipant.CheckedIn = &p.CheckedIn
	if p.CheckedInAt != nil {
		genParticipant.CheckedInAt = p.CheckedInAt
	}

	return genParticipant
}

// ptrOrDefault returns the value pointed to by ptr, or defaultVal if ptr is nil
func ptrOrDefault[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// convertEmailPtr converts openapi_types.Email pointer to string pointer
func convertEmailPtr(email *openapi_types.Email) *string {
	if email == nil {
		return nil
	}
	str := string(*email)
	return &str
}

// convertEmailPtrToStringPtr converts openapi_types.Email pointer to string pointer
func convertEmailPtrToStringPtr(email *openapi_types.Email) *string {
	if email == nil {
		return nil
	}
	str := string(*email)
	return &str
}

// convertMetadataToString converts metadata map to JSON string
func convertMetadataToString(metadata *map[string]any) *string {
	if metadata == nil {
		return nil
	}
	data, err := json.Marshal(*metadata)
	if err != nil {
		return nil
	}
	str := string(data)
	return &str
}

// convertRawMessageToMap converts json.RawMessage to map
func convertRawMessageToMap(raw *json.RawMessage) map[string]any {
	if raw == nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(*raw, &result); err != nil {
		return nil
	}
	return result
}
