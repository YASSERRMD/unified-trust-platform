const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface ApiError {
  code: string;
  message: string;
  details?: unknown;
}

export interface ApiResponse<T> {
  data: T;
  meta: { requestId: string; timestamp: string };
}

export interface CollectionResponse<T> {
  data: T[];
  pagination?: { nextCursor?: string; hasMore: boolean; total?: number };
  meta: { requestId: string; timestamp: string };
}

async function request<T>(
  path: string,
  options: RequestInit & { tenantId?: string } = {}
): Promise<T> {
  const { tenantId, ...init } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init.headers as Record<string, string>),
  };
  if (tenantId) {
    headers["X-Tenant-ID"] = tenantId;
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const err: ApiError = body?.error ?? {
      code: "UNKNOWN",
      message: `HTTP ${res.status}`,
    };
    throw err;
  }

  return res.json() as Promise<T>;
}

export const api = {
  get<T>(path: string, tenantId?: string): Promise<T> {
    return request<T>(path, { method: "GET", tenantId });
  },

  post<T>(path: string, body: unknown, tenantId?: string): Promise<T> {
    return request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      tenantId,
    });
  },

  patch<T>(path: string, body: unknown, tenantId?: string): Promise<T> {
    return request<T>(path, {
      method: "PATCH",
      body: JSON.stringify(body),
      tenantId,
    });
  },

  put<T>(path: string, body: unknown, tenantId?: string): Promise<T> {
    return request<T>(path, {
      method: "PUT",
      body: JSON.stringify(body),
      tenantId,
    });
  },

  delete(path: string, tenantId?: string): Promise<void> {
    return request<void>(path, { method: "DELETE", tenantId });
  },

  // Convenience: list endpoint returning CollectionResponse
  list<T>(path: string, tenantId?: string): Promise<CollectionResponse<T>> {
    return this.get<CollectionResponse<T>>(path, tenantId);
  },
};

// Domain-specific helpers
export const tenantApi = {
  list: (cursor?: string) =>
    api.list<Tenant>(`/api/v1/tenants${cursor ? `?cursor=${cursor}` : ""}`),
  get: (id: string) => api.get<ApiResponse<Tenant>>(`/api/v1/tenants/${id}`),
  create: (body: Partial<Tenant>) =>
    api.post<ApiResponse<Tenant>>("/api/v1/tenants", body),
};

export const userApi = {
  list: (tenantId: string, cursor?: string) =>
    api.list<User>(
      `/api/v1/users${cursor ? `?cursor=${cursor}` : ""}`,
      tenantId
    ),
  get: (id: string, tenantId: string) =>
    api.get<ApiResponse<User>>(`/api/v1/users/${id}`, tenantId),
  create: (body: Partial<User>, tenantId: string) =>
    api.post<ApiResponse<User>>("/api/v1/users", body, tenantId),
};

export const roleApi = {
  list: (tenantId: string) =>
    api.list<Role>("/api/v1/roles", tenantId),
  create: (body: { name: string; description?: string }, tenantId: string) =>
    api.post<ApiResponse<Role>>("/api/v1/roles", body, tenantId),
};

export const policyApi = {
  list: (tenantId: string) =>
    api.list<Policy>("/api/v1/policies", tenantId),
  create: (body: Partial<Policy>, tenantId: string) =>
    api.post<ApiResponse<Policy>>("/api/v1/policies", body, tenantId),
  evaluate: (body: EvalRequest, tenantId: string) =>
    api.post<EvalDecision>("/api/v1/authz/evaluate", body, tenantId),
};

export const auditApi = {
  list: (tenantId: string, params?: Record<string, string>) => {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return api.list<AuditEvent>(`/api/v1/audit/events${qs}`, tenantId);
  },
};

// Types
export interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: string;
  createdAt: string;
}

export interface User {
  id: string;
  tenantId: string;
  email: string;
  displayName: string;
  status: string;
  createdAt: string;
}

export interface Role {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  isSystem: boolean;
  createdAt: string;
}

export interface Policy {
  id: string;
  tenantId: string;
  name: string;
  effect: "permit" | "deny";
  priority: number;
  actions: string[];
  isActive: boolean;
}

export interface EvalRequest {
  subject: { userId: string; roles?: string[] };
  action: string;
  resource: { type: string; id?: string };
}

export interface EvalDecision {
  allowed: boolean;
  decision: string;
  reason: string;
  matchedPolicies: string[];
  evaluationMs: number;
}

export interface AuditEvent {
  id: string;
  tenantId: string;
  actorId?: string;
  actorEmail?: string;
  action: string;
  resource: string;
  outcome: string;
  occurredAt: string;
}
