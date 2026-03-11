package participant

import (
	"context"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
)

// Usecase defines participant business logic operations
type Usecase interface {
	Create(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input CreateParticipantInput,
	) (*entity.Participant, error)
	BulkCreate(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input BulkCreateInput,
	) (BulkCreateOutput, error)
	GetByID(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		id uuid.UUID,
	) (*entity.Participant, error)
	List(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input ListParticipantsInput,
	) (ListParticipantsOutput, error)
	Update(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		id uuid.UUID,
		input UpdateParticipantInput,
	) (*entity.Participant, error)
	Delete(ctx context.Context, userID uuid.UUID, isAdmin bool, id uuid.UUID) error
	GetQRCode(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		id uuid.UUID,
		format string,
		size int,
	) (QRCodeOutput, error)
	ExportCSV(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		eventID uuid.UUID,
	) ([]*entity.Participant, error)
	SendQRCodes(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input SendQRCodesInput,
	) (SendQRCodesOutput, error)
}

var _ Usecase = (*participantUsecase)(nil)

type participantUsecase struct {
	participantRepo    repository.ParticipantRepository
	eventRepo          repository.EventRepository
	qrGenerator        *qrcode.Generator
	qrHMACSecret       string
	qrHostingBaseURL   string
	walletPassBaseURL  string
	emailSender        domainemail.Sender
	emailPlainTextOnly bool
	logger             *logger.Logger
}

// NewUsecase creates a new participant usecase instance
func NewUsecase(
	participantRepo repository.ParticipantRepository,
	eventRepo repository.EventRepository,
	qrGenerator *qrcode.Generator,
	qrHMACSecret string,
	qrHostingBaseURL string,
	walletPassBaseURL string,
	emailSender domainemail.Sender,
	emailPlainTextOnly bool,
	logger *logger.Logger,
) Usecase {
	return &participantUsecase{
		participantRepo:    participantRepo,
		eventRepo:          eventRepo,
		qrGenerator:        qrGenerator,
		qrHMACSecret:       qrHMACSecret,
		qrHostingBaseURL:   qrHostingBaseURL,
		walletPassBaseURL:  walletPassBaseURL,
		emailSender:        emailSender,
		emailPlainTextOnly: emailPlainTextOnly,
		logger:             logger,
	}
}

// populateDistributionURL computes and sets QRDistributionURL for a participant
// based on the QR code token and the configured hosting base URL.
func (u *participantUsecase) populateDistributionURL(p *entity.Participant) {
	p.QRDistributionURL = crypto.GenerateQRDistributionURL(u.qrHostingBaseURL, p.QRCode)
}

// populateDistributionURLs computes and sets QRDistributionURL for multiple participants.
func (u *participantUsecase) populateDistributionURLs(participants []*entity.Participant) {
	for _, p := range participants {
		u.populateDistributionURL(p)
	}
}
