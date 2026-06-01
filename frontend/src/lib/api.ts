const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
  }
}

function getToken(): string | null {
  if (typeof document === "undefined") return null;
  const m = document.cookie.match(/(?:^|;\s*)token=([^;]+)/);
  return m ? decodeURIComponent(m[1]) : null;
}

async function request<T>(
  path: string,
  init: RequestInit = {}
): Promise<T> {
  const token = getToken();

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init.headers as Record<string, string> ?? {}),
  };

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      message = body.message ?? message;
    } catch {
      // non-JSON error body
    }
    throw new ApiError(res.status, message);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// ── Auth ────────────────────────────────────────────────────

export interface AuthResponse {
  token: string;
  user: { id: string; email: string; name: string; avatar_url?: string };
}

export const auth = {
  signup: (name: string, email: string, password: string) =>
    request<AuthResponse>("/api/auth/signup", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    }),

  login: (email: string, password: string) =>
    request<AuthResponse>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  logout: () =>
    request<void>("/api/auth/logout", { method: "POST" }),
};

// ── Workspaces ──────────────────────────────────────────────

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  created_by: string;
  deadline_day?: number;
  deadline_hour?: number;
  created_at: string;
}

export interface WorkspaceMember {
  id: string;
  workspace_id: string;
  user_id: string;
  role: "admin" | "member" | "viewer";
  joined_at: string;
  name?: string;
  email?: string;
}

export interface WorkspaceDetail {
  id: string;
  name: string;
  slug: string;
  created_by: string;
  deadline_day?: number;
  deadline_hour?: number;
  created_at: string;
  members: WorkspaceMember[];
  my_role: string;
}

export const workspaces = {
  create: (name: string, slug?: string) =>
    request<WorkspaceDetail>("/api/workspaces", {
      method: "POST",
      body: JSON.stringify({ name, slug }),
    }),

  get: (slug: string) =>
    request<WorkspaceDetail>(`/api/workspaces/${slug}`),

  getMembers: (slug: string) =>
    request<WorkspaceMember[]>(`/api/workspaces/${slug}/members`),

  invite: (slug: string, email: string, role: "member" | "viewer") =>
    request<{ invite_url: string; email: string; expires_at: string }>(
      `/api/workspaces/${slug}/invite`,
      { method: "POST", body: JSON.stringify({ email, role }) }
    ),

  updateMemberRole: (slug: string, memberId: string, role: string) =>
    request<void>(`/api/workspaces/${slug}/members/${memberId}`, {
      method: "PUT",
      body: JSON.stringify({ role }),
    }),

  removeMember: (slug: string, memberId: string) =>
    request<void>(`/api/workspaces/${slug}/members/${memberId}`, {
      method: "DELETE",
    }),
};

// ── Reports ─────────────────────────────────────────────────

export interface WeeklyReport {
  id: string;
  workspace_id: string;
  user_id: string;
  week_start_date: string;
  done?: string;
  blockers?: string;
  next_week?: string;
  score?: number;
  submitted_at?: string;
  updated_at: string;
  user_name?: string;
  user_email?: string;
}

export interface TeamReportsResponse {
  week_start_date: string;
  reports: WeeklyReport[];
  submitted: number;
  total: number;
}

export const reports = {
  submit: (
    slug: string,
    data: {
      week_start_date?: string;
      done?: string;
      blockers?: string;
      next_week?: string;
      score?: number;
    }
  ) =>
    request<WeeklyReport>(`/api/workspaces/${slug}/reports`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getTeam: (slug: string, week?: string) =>
    request<TeamReportsResponse>(
      `/api/workspaces/${slug}/reports${week ? `?week=${encodeURIComponent(week)}` : ""}`
    ),

  getMyReports: (slug: string) =>
    request<WeeklyReport[]>(`/api/workspaces/${slug}/reports/me`),

  getWeek: (slug: string, week: string) =>
    request<TeamReportsResponse>(`/api/workspaces/${slug}/reports/${week}`),
};
