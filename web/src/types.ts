export interface ISPStatus {
  name: string;
  asn?: number;
  total: number;
  up: number;
  down: number;
  last_updated: string;
}

export interface StatusResponse {
  isp: string;
  registered: boolean;
  can_register: boolean;
  endpoint_id: string | null;
  isp_status?: ISPStatus;
}

export interface RegisterResponse {
  endpoint_id: string;
  isp: string;
  message: string;
}

export interface DashboardResponse {
  isps: ISPStatus[];
  likely_outage: boolean;
  last_updated: string;
}

export interface HealthResponse {
  status: string;
  version: string;
}

export interface AdminEndpoint {
  id: string;
  ipv4: string;
  isp: string;
  status: string;
  created_at: string;
  last_seen: string;
  last_ok?: string;
  monitored_hop?: string;
  hop_number?: number;
  use_hop: boolean;
}

export interface AdminAddRequest {
  ipv4: string;
  isp?: string;
}

export interface ISPMetrics {
  name: string;
  total: number;
  up: number;
  down: number;
  unknown: number;
  uptime_pct: number;
  likely_outage: boolean;
}

export interface UptimePoint {
  timestamp: string;
  uptime_pct: number;
  up: number;
  down: number;
}

export interface Event {
  id: number;
  timestamp: string;
  event_type: string;  // "down", "up", "outage", "recovery"
  isp?: string;
  endpoint_id?: string;
  message: string;
}

export interface EventsResponse {
  events: Event[];
}

export interface AdminMetrics {
  total_endpoints: number;
  endpoints_up: number;
  endpoints_down: number;
  endpoints_unknown: number;
  overall_uptime_pct: number;
  isp_stats: ISPMetrics[];
  last_ping_time: string;
  ping_interval: string;
  next_ping_time: string;
  total_ping_cycles: number;
  direct_monitored: number;
  hop_monitored: number;
  shared_hops: number;
  server_start_time: string;
  server_uptime: string;
  version: string;
  database_size_bytes: number;
  database_path: string;
  uptime_history: UptimePoint[];
}
