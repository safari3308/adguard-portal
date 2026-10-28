package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/safari3308/adguard-portal/internal/auth"
	"github.com/safari3308/adguard-portal/internal/domain"
	"github.com/safari3308/adguard-portal/internal/service"
)

type downloadTicket struct {
	userID    string
	deviceID  string
	expiresAt time.Time
}

type DeviceHandler struct {
	svc     *service.DeviceService
	tickets map[string]downloadTicket
	mu      sync.Mutex
}

func NewDeviceHandler(svc *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{
		svc:     svc,
		tickets: make(map[string]downloadTicket),
	}
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	devices, err := h.svc.ListDevices(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if devices == nil {
		devices = []*domain.Device{}
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	var input domain.CreateDeviceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err.Error() != "EOF" {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	device, err := h.svc.CreateDevice(r.Context(), user.ID, input)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceLimitReached) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, device)
}

func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")

	device, err := h.svc.GetDevice(r.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedDevice) {
			writeError(w, http.StatusForbidden, "unauthorized access to device")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")

	var input domain.UpdateDeviceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	device, err := h.svc.UpdateDevice(r.Context(), user.ID, id, input)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidDomainRules) || errors.Is(err, domain.ErrMaxRulesExceeded) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")

	if err := h.svc.DeleteDevice(r.Context(), user.ID, id); err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeviceHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")
	// Verify device exists and belongs to user
	_, err := h.svc.GetDevice(r.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ticket := uuid.NewString()

	h.mu.Lock()
	now := time.Now()
	// Clean up old expired tickets
	for k, v := range h.tickets {
		if now.After(v.expiresAt) {
			delete(h.tickets, k)
		}
	}
	h.tickets[ticket] = downloadTicket{
		userID:    user.ID,
		deviceID:  id,
		expiresAt: now.Add(60 * time.Second),
	}
	h.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"ticket": ticket,
	})
}

func (h *DeviceHandler) DownloadMobileConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var userID string

	ticket := r.URL.Query().Get("ticket")
	if ticket != "" {
		h.mu.Lock()
		info, exists := h.tickets[ticket]
		if exists {
			delete(h.tickets, ticket) // single-use
		}
		h.mu.Unlock()

		if !exists || time.Now().After(info.expiresAt) || info.deviceID != id {
			writeError(w, http.StatusUnauthorized, "invalid or expired download ticket")
			return
		}
		userID = info.userID
	} else {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userID = user.ID
	}

	content, err := h.svc.GenerateMobileConfig(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("adguard-doh-%s.mobileconfig", id)
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
