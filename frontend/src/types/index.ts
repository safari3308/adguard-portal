export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface Device {
  id: string;
  user_id: string;
  name: string;
  client_token: string;
  filtering_enabled: boolean;
  parental_enabled: boolean;
  safe_search_enabled: boolean;
  blocked_services: string[];
  custom_blocked_domains: string[];
  custom_allowed_domains: string[];
  created_at: string;
  updated_at: string;
}

export interface BlockedService {
  id: string;
  name: string;
  icon?: string;
}

export interface AppConfig {
  doh_domain: string;
  oidc_issuer: string;
  oidc_client_id: string;
  oidc_scopes?: string;
  oidc_provider_name?: string;
  max_devices_per_user: number;
  dev_mode: boolean;
}
