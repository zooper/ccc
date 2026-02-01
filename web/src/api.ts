import type { StatusResponse, RegisterResponse, DashboardResponse, HealthResponse, AdminEndpoint, AdminAddRequest, AdminMetrics, EventsResponse, AdminSettings } from './types';

const API_BASE = '/api';

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('Authentication required');
    }
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

function authHeader(password: string): Record<string, string> {
  // Basic auth with username "admin" and provided password
  const credentials = btoa(`admin:${password}`);
  return { Authorization: `Basic ${credentials}` };
}

export async function getHealth(): Promise<HealthResponse> {
  return fetchJSON<HealthResponse>(`${API_BASE}/health`);
}

export async function getStatus(): Promise<StatusResponse> {
  return fetchJSON<StatusResponse>(`${API_BASE}/status`);
}

export async function register(): Promise<RegisterResponse> {
  return fetchJSON<RegisterResponse>(`${API_BASE}/register`, {
    method: 'POST',
  });
}

export async function getDashboard(): Promise<DashboardResponse> {
  return fetchJSON<DashboardResponse>(`${API_BASE}/dashboard`);
}

export async function getEvents(): Promise<EventsResponse> {
  return fetchJSON<EventsResponse>(`${API_BASE}/events`);
}

// Admin API (requires password)
export async function adminListEndpoints(password: string): Promise<AdminEndpoint[]> {
  return fetchJSON<AdminEndpoint[]>(`${API_BASE}/admin/endpoints`, {
    headers: authHeader(password),
  });
}

export async function adminAddEndpoint(password: string, data: AdminAddRequest): Promise<AdminEndpoint> {
  return fetchJSON<AdminEndpoint>(`${API_BASE}/admin/endpoints`, {
    method: 'POST',
    headers: authHeader(password),
    body: JSON.stringify(data),
  });
}

export async function adminDeleteEndpoint(password: string, id: string): Promise<void> {
  await fetchJSON<{ message: string }>(`${API_BASE}/admin/endpoints/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    headers: authHeader(password),
  });
}

export async function adminGetMetrics(password: string): Promise<AdminMetrics> {
  return fetchJSON<AdminMetrics>(`${API_BASE}/admin/metrics`, {
    headers: authHeader(password),
  });
}

export async function adminGetSettings(password: string): Promise<AdminSettings> {
  return fetchJSON<AdminSettings>(`${API_BASE}/admin/settings`, {
    headers: authHeader(password),
  });
}

export async function adminUpdateSettings(password: string, settings: AdminSettings): Promise<AdminSettings> {
  return fetchJSON<AdminSettings>(`${API_BASE}/admin/settings`, {
    method: 'PUT',
    headers: authHeader(password),
    body: JSON.stringify(settings),
  });
}
