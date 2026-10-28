import { getAccessToken } from './auth';
import type { AppConfig, Device, BlockedService, User } from '../types';

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  const token = await getAccessToken();
  const headers = new Headers(options.headers || {});
  
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  headers.set('Content-Type', 'application/json');

  const res = await fetch(url, { ...options, headers });
  if (res.status === 401) {
    // If unauthorized, session is expired
    throw new Error('Unauthorized');
  }
  return res;
}

export async function fetchPublicConfig(): Promise<AppConfig> {
  const res = await fetch('/api/config');
  if (!res.ok) {
    throw new Error('Failed to fetch public configuration');
  }
  return res.json();
}

export async function fetchMe(): Promise<User> {
  const res = await fetchWithAuth('/api/me');
  if (!res.ok) {
    throw new Error('Failed to fetch user profile');
  }
  return res.json();
}

export async function fetchDevices(): Promise<Device[]> {
  const res = await fetchWithAuth('/api/devices');
  if (!res.ok) {
    throw new Error('Failed to load devices');
  }
  return res.json();
}

export async function createDevice(name: string): Promise<Device> {
  const res = await fetchWithAuth('/api/devices', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || 'Failed to create device');
  }
  return res.json();
}

export async function updateDevice(id: string, updates: Partial<Device>): Promise<Device> {
  const res = await fetchWithAuth(`/api/devices/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(updates),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || 'Failed to update device');
  }
  return res.json();
}

export async function deleteDevice(id: string): Promise<void> {
  const res = await fetchWithAuth(`/api/devices/${id}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    throw new Error('Failed to delete device');
  }
}

export async function fetchServices(): Promise<BlockedService[]> {
  const res = await fetchWithAuth('/api/services');
  if (!res.ok) {
    return [];
  }
  return res.json();
}

export async function getMobileConfigUrl(deviceId: string): Promise<string> {
  const token = await getAccessToken();
  if (token) {
    return `/api/devices/${deviceId}/mobileconfig?token=${encodeURIComponent(token)}`;
  }
  return `/api/devices/${deviceId}/mobileconfig`;
}
