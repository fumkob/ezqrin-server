package handler

import (
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

// QRCodeHandler handles QR code distribution endpoints.
type QRCodeHandler struct {
	participantUsecase participant.Usecase
	logger             *logger.Logger
}

// NewQRCodeHandler creates a new QRCodeHandler.
func NewQRCodeHandler(uc participant.Usecase, logger *logger.Logger) *QRCodeHandler {
	return &QRCodeHandler{participantUsecase: uc, logger: logger}
}

// SendEventQRCodes handles POST /events/{id}/qrcodes/send.
func (h *QRCodeHandler) SendEventQRCodes(c *gin.Context, id generated.EventIDParam) {
	var req generated.SendQRCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	userID, _ := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == string(entity.RoleAdmin)

	// Convert participant_ids from OpenAPI UUIDs to domain UUIDs.
	var participantIDs []uuid.UUID
	if req.ParticipantIds != nil {
		participantIDs = make([]uuid.UUID, 0, len(*req.ParticipantIds))
		for _, pid := range *req.ParticipantIds {
			participantIDs = append(participantIDs, uuid.UUID(pid))
		}
	}

	sendToAll := false
	if req.SendToAll != nil {
		sendToAll = *req.SendToAll
	}

	template := "default"
	if req.EmailTemplate != nil {
		template = string(*req.EmailTemplate)
	}

	input := participant.SendQRCodesInput{
		EventID:        uuid.UUID(id),
		ParticipantIDs: participantIDs,
		SendToAll:      sendToAll,
		EmailTemplate:  template,
	}

	result, err := h.participantUsecase.SendQRCodes(c.Request.Context(), userID, isAdmin, input)
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Return 207 Multi-Status when some sends succeeded and some failed.
	statusCode := http.StatusOK
	if result.FailedCount > 0 && result.SentCount > 0 {
		statusCode = http.StatusMultiStatus
	}

	response.Data(c, statusCode, generated.SendQRCodesResponse{
		SentCount:   result.SentCount,
		FailedCount: result.FailedCount,
		Total:       result.Total,
		Failures:    toGeneratedFailures(result.Failures),
	})
}

// toGeneratedFailures converts usecase failures to generated API failures.
func toGeneratedFailures(fs []participant.SendQRCodeFailure) []generated.SendQRCodeFailure {
	out := make([]generated.SendQRCodeFailure, 0, len(fs))
	for _, f := range fs {
		pid := openapi_types.UUID(f.ParticipantID)
		email := openapi_types.Email(f.Email)
		out = append(out, generated.SendQRCodeFailure{
			ParticipantId: pid,
			Email:         email,
			Reason:        f.Reason,
		})
	}
	return out
}
