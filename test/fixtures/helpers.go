package fixtures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const eventStartOffsetHours = 24

// Helper provides HTTP test helper methods for making API calls.
type Helper struct {
	Router *gin.Engine
}

// NewHelper creates a new test helper with the given router.
func NewHelper(router *gin.Engine) *Helper {
	return &Helper{Router: router}
}

// RegisterUser creates a new user via POST /api/v1/auth/register.
// Returns the auth response (with access token).
// Panics on failure to surface errors clearly in tests.
func (h *Helper) RegisterUser(email, password, name, role string) *generated.AuthResponse {
	reqBody := generated.RegisterRequest{
		Email:    openapi_types.Email(email),
		Password: password,
		Name:     name,
		Role:     generated.UserRole(role),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		panic(fmt.Sprintf("RegisterUser failed: status=%d body=%s", w.Code, w.Body.String()))
	}
	var resp generated.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp
}

// Login authenticates a user via POST /api/v1/auth/login.
func (h *Helper) Login(email, password string) *generated.AuthResponse {
	reqBody := generated.LoginRequest{
		Email:    openapi_types.Email(email),
		Password: password,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		panic(fmt.Sprintf("Login failed: status=%d body=%s", w.Code, w.Body.String()))
	}
	var resp generated.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp
}

// CreateEvent creates an event via POST /api/v1/events.
func (h *Helper) CreateEvent(token, name string) *generated.Event {
	reqBody := generated.CreateEventRequest{
		Name:      name,
		StartDate: time.Now().Add(eventStartOffsetHours * time.Hour),
		Status:    generated.EventStatusPublished,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		panic(fmt.Sprintf("CreateEvent failed: status=%d body=%s", w.Code, w.Body.String()))
	}
	var resp generated.Event
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp
}

// CreateParticipant adds a participant to an event via POST /api/v1/events/:id/participants.
func (h *Helper) CreateParticipant(token, eventID, name, email string) *generated.Participant {
	reqBody := map[string]string{"name": name, "email": email}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+eventID+"/participants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		panic(fmt.Sprintf("CreateParticipant failed: status=%d body=%s", w.Code, w.Body.String()))
	}
	var resp generated.Participant
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp
}

// CheckInByQR performs a QR-code check-in via POST /api/v1/events/:id/checkin.
func (h *Helper) CheckInByQR(token, eventID, qrCode string) (*generated.CheckInResponse, int) {
	reqBody := map[string]string{"method": "qrcode", "qr_code": qrCode}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+eventID+"/checkin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return nil, w.Code
	}
	var resp generated.CheckInResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp, w.Code
}

// CheckInByManual performs a manual check-in by participant ID.
func (h *Helper) CheckInByManual(token, eventID, participantID string) (*generated.CheckInResponse, int) {
	reqBody := map[string]string{"method": "manual", "participant_id": participantID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+eventID+"/checkin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return nil, w.Code
	}
	var resp generated.CheckInResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp, w.Code
}

// GetEventStats retrieves event statistics via GET /api/v1/events/:id/stats.
func (h *Helper) GetEventStats(token, eventID string) (*generated.EventStatsResponse, int) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+eventID+"/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return nil, w.Code
	}
	var resp generated.EventStatsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp, w.Code
}

// DoRequest is a low-level helper for making arbitrary requests and returning the raw recorder.
func (h *Helper) DoRequest(method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)
	return w
}
