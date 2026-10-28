package agh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/safari3308/adguard-portal/internal/domain"
)

type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	mu         sync.Mutex
}

func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type aghClientPayload struct {
	Name              string   `json:"name"`
	IDs               []string `json:"ids"`
	UseGlobalSettings bool     `json:"use_global_settings"`
	FilteringEnabled  bool     `json:"filtering_enabled"`
	ParentalEnabled   bool     `json:"parental_enabled"`
	SafeSearchEnabled bool     `json:"safesearch_enabled"`
	BlockedServices   []string `json:"blocked_services"`
	Rules             []string `json:"rules"`
}

type aghUpdatePayload struct {
	Name string           `json:"name"`
	Data aghClientPayload `json:"data"`
}

type aghDeletePayload struct {
	Name string `json:"name"`
}

type aghAccessList struct {
	AllowedClients    []string `json:"allowed_clients"`
	DisallowedClients []string `json:"disallowed_clients"`
	BlockedHosts      []string `json:"blocked_hosts"`
}

func clientName(device *domain.Device) string {
	return fmt.Sprintf("%s (%s)", device.Name, device.ClientToken)
}

func compileRules(blockedDomains, allowedDomains []string) []string {
	rules := make([]string, 0, len(blockedDomains)+len(allowedDomains))
	for _, domainName := range blockedDomains {
		domainName = strings.TrimSpace(domainName)
		if domainName != "" {
			rules = append(rules, fmt.Sprintf("||%s^", domainName))
		}
	}
	for _, domainName := range allowedDomains {
		domainName = strings.TrimSpace(domainName)
		if domainName != "" {
			rules = append(rules, fmt.Sprintf("@@||%s^", domainName))
		}
	}
	return rules
}

func (c *Client) SyncDevice(ctx context.Context, device *domain.Device) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := clientName(device)
	rules := compileRules(device.CustomBlockedDomains, device.CustomAllowedDomains)

	payload := aghClientPayload{
		Name:              name,
		IDs:               []string{device.ClientToken},
		UseGlobalSettings: false,
		FilteringEnabled:  device.FilteringEnabled,
		ParentalEnabled:   device.ParentalEnabled,
		SafeSearchEnabled: device.SafeSearchEnabled,
		BlockedServices:   device.BlockedServices,
		Rules:             rules,
	}

	// Try updating first; if it returns 400/404 or fails, try adding
	updateReq := aghUpdatePayload{
		Name: name,
		Data: payload,
	}

	status, _, err := c.doRequest(ctx, http.MethodPost, "/control/clients/update", updateReq)
	if err == nil && status == http.StatusOK {
		return c.ensureAllowedClientInternal(ctx, device.ClientToken)
	}

	// If update failed, attempt adding new client
	addStatus, respBody, err := c.doRequest(ctx, http.MethodPost, "/control/clients/add", payload)
	if err != nil {
		return fmt.Errorf("agh add client failed: %w", err)
	}
	if addStatus != http.StatusOK {
		return fmt.Errorf("agh add client returned status %d: %s", addStatus, string(respBody))
	}

	return c.ensureAllowedClientInternal(ctx, device.ClientToken)
}

func (c *Client) DeleteDevice(ctx context.Context, clientToken, deviceName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := fmt.Sprintf("%s (%s)", deviceName, clientToken)
	delPayload := aghDeletePayload{Name: name}

	_, _, _ = c.doRequest(ctx, http.MethodPost, "/control/clients/delete", delPayload)
	return c.removeAllowedClientInternal(ctx, clientToken)
}

func (c *Client) EnsureAllowedClient(ctx context.Context, clientToken string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ensureAllowedClientInternal(ctx, clientToken)
}

func (c *Client) RemoveAllowedClient(ctx context.Context, clientToken string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.removeAllowedClientInternal(ctx, clientToken)
}

func (c *Client) ensureAllowedClientInternal(ctx context.Context, clientToken string) error {
	access, err := c.getAccessList(ctx)
	if err != nil {
		return err
	}

	// Only modify if access control list is enabled (has allowed clients)
	if len(access.AllowedClients) == 0 {
		return nil
	}

	for _, id := range access.AllowedClients {
		if id == clientToken {
			return nil
		}
	}

	access.AllowedClients = append(access.AllowedClients, clientToken)
	return c.setAccessList(ctx, access)
}

func (c *Client) removeAllowedClientInternal(ctx context.Context, clientToken string) error {
	access, err := c.getAccessList(ctx)
	if err != nil {
		return err
	}

	if len(access.AllowedClients) == 0 {
		return nil
	}

	newAllowed := make([]string, 0, len(access.AllowedClients))
	for _, id := range access.AllowedClients {
		if id != clientToken {
			newAllowed = append(newAllowed, id)
		}
	}

	access.AllowedClients = newAllowed
	return c.setAccessList(ctx, access)
}

func (c *Client) getAccessList(ctx context.Context) (*aghAccessList, error) {
	status, body, err := c.doRequest(ctx, http.MethodGet, "/control/access/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch access list: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get access list returned status %d: %s", status, string(body))
	}

	var access aghAccessList
	if err := json.Unmarshal(body, &access); err != nil {
		return nil, fmt.Errorf("unmarshal access list: %w", err)
	}
	return &access, nil
}

func (c *Client) setAccessList(ctx context.Context, access *aghAccessList) error {
	status, body, err := c.doRequest(ctx, http.MethodPost, "/control/access/set", access)
	if err != nil {
		return fmt.Errorf("failed to set access list: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("set access list returned status %d: %s", status, string(body))
	}
	return nil
}

func (c *Client) GetBlockedServices(ctx context.Context) ([]domain.BlockedService, error) {
	status, body, err := c.doRequest(ctx, http.MethodGet, "/control/blocked_services/all", nil)
	if err != nil {
		return defaultBlockedServices(), nil
	}
	if status != http.StatusOK {
		return defaultBlockedServices(), nil
	}

	// Handle string array response e.g. ["tiktok", "youtube"]
	var strServices []string
	if err := json.Unmarshal(body, &strServices); err == nil && len(strServices) > 0 {
		result := make([]domain.BlockedService, len(strServices))
		for i, s := range strServices {
			result[i] = domain.BlockedService{
				ID:   s,
				Name: formatServiceName(s),
			}
		}
		return result, nil
	}

	// Handle map or struct array if newer AGH
	var mapServices struct {
		BlockedServices []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Icon string `json:"icon"`
		} `json:"blocked_services"`
	}
	if err := json.Unmarshal(body, &mapServices); err == nil && len(mapServices.BlockedServices) > 0 {
		result := make([]domain.BlockedService, len(mapServices.BlockedServices))
		for i, s := range mapServices.BlockedServices {
			result[i] = domain.BlockedService{
				ID:   s.ID,
				Name: s.Name,
				Icon: s.Icon,
			}
		}
		return result, nil
	}

	return defaultBlockedServices(), nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, payload any) (int, []byte, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var reqBody io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, bodyBytes, nil
}

func formatServiceName(raw string) string {
	parts := strings.Split(raw, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func defaultBlockedServices() []domain.BlockedService {
	services := []string{
		"tiktok", "youtube", "facebook", "instagram", "twitter",
		"reddit", "discord", "steam", "netflix", "spotify",
		"roblox", "twitch", "telegram", "whatsapp", "snapchat",
	}
	out := make([]domain.BlockedService, len(services))
	for i, s := range services {
		out[i] = domain.BlockedService{
			ID:   s,
			Name: formatServiceName(s),
		}
	}
	return out
}
