package handler

import (
	"net/http"

	"github.com/safari3308/adguard-portal/internal/domain"
)

type ConfigResponse struct {
	DOHDomain         string `json:"doh_domain"`
	OIDCIssuer        string `json:"oidc_issuer"`
	OIDCClientID      string `json:"oidc_client_id"`
	OIDCClientSecret  string `json:"oidc_client_secret,omitempty"`
	OIDCScopes        string `json:"oidc_scopes"`
	OIDCProviderName  string `json:"oidc_provider_name"`
	MaxDevicesPerUser int    `json:"max_devices_per_user"`
	DevMode           bool   `json:"dev_mode"`
}

type ServiceHandler struct {
	agh    domain.AGHClient
	config ConfigResponse
}

func NewServiceHandler(agh domain.AGHClient, config ConfigResponse) *ServiceHandler {
	return &ServiceHandler{
		agh:    agh,
		config: config,
	}
}

func (h *ServiceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	if h.agh == nil {
		writeJSON(w, http.StatusOK, []domain.BlockedService{})
		return
	}

	services, err := h.agh.GetBlockedServices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch services")
		return
	}

	writeJSON(w, http.StatusOK, services)
}

func (h *ServiceHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.config)
}
