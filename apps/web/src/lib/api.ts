export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8180";

export interface UserProfile {
  display_name: string;
  bio: string;
  avatar_url: string;
  unit_pref: "kg" | "lb";
  start_weight_grams?: number;
  current_weight_grams?: number;
  goal_weight_grams?: number;
  weight_privacy: "public" | "squads_only" | "private";
  journey_focus: string[];
  timezone: string;
}

export interface AuthUser {
  id: string;
  email: string;
  username?: string;
  onboarding_completed: boolean;
  created_at: string;
  profile?: UserProfile;
}

export interface RequestOtpResponse {
  message: string;
  expires_in_seconds: number;
  resend_after_seconds: number;
}

export interface VerifyOtpResponse {
  access_token: string;
  is_new_user: boolean;
  onboarding_completed: boolean;
  user: AuthUser;
}

export interface RefreshResponse {
  access_token: string;
  onboarding_completed: boolean;
  user: AuthUser;
}

export interface CompleteOnboardingPayload {
  username: string;
  display_name: string;
  bio: string;
  unit_pref: "kg" | "lb";
  start_weight_grams?: number;
  current_weight_grams?: number;
  goal_weight_grams?: number;
  weight_privacy: "public" | "squads_only" | "private";
  journey_focus: string[];
  timezone: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details?: unknown;

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

interface Envelope<T> {
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: unknown;
  };
}

async function apiRequest<T>(
  path: string,
  options: RequestInit & { accessToken?: string } = {},
): Promise<T> {
  const { accessToken, headers, ...rest } = options;
  const reqHeaders = new Headers(headers);
  if (!reqHeaders.has("Content-Type") && rest.body) {
    reqHeaders.set("Content-Type", "application/json");
  }
  if (accessToken) {
    reqHeaders.set("Authorization", `Bearer ${accessToken}`);
  }

  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...rest,
    headers: reqHeaders,
    credentials: "include",
  });

  const payload = (await res.json().catch(() => ({}))) as Envelope<T>;
  if (!res.ok) {
    throw new ApiError(
      res.status,
      payload.error?.code ?? "REQUEST_FAILED",
      payload.error?.message ?? "Something went wrong. Please try again.",
      payload.error?.details,
    );
  }
  return payload.data as T;
}

export function requestEmailOtp(email: string): Promise<RequestOtpResponse> {
  return apiRequest<RequestOtpResponse>("/api/v1/auth/otp/request", {
    method: "POST",
    body: JSON.stringify({ email }),
  });
}

export function verifyEmailOtp(email: string, code: string): Promise<VerifyOtpResponse> {
  return apiRequest<VerifyOtpResponse>("/api/v1/auth/otp/verify", {
    method: "POST",
    body: JSON.stringify({ email, code }),
  });
}

export function refreshSession(): Promise<RefreshResponse> {
  return apiRequest<RefreshResponse>("/api/v1/auth/refresh", {
    method: "POST",
  });
}

export function logoutSession(accessToken: string): Promise<{ status: string }> {
  return apiRequest<{ status: string }>("/api/v1/auth/logout", {
    method: "POST",
    accessToken,
  });
}

export function fetchCurrentUser(accessToken: string): Promise<{ user: AuthUser }> {
  return apiRequest<{ user: AuthUser }>("/api/v1/auth/me", {
    method: "GET",
    accessToken,
  });
}

export function checkUsernameAvailable(
  username: string,
  accessToken: string,
): Promise<{ username: string; available: boolean }> {
  const qs = new URLSearchParams({ username });
  return apiRequest<{ username: string; available: boolean }>(
    `/api/v1/onboarding/username-available?${qs.toString()}`,
    {
      method: "GET",
      accessToken,
    },
  );
}

export function completeOnboarding(
  payload: CompleteOnboardingPayload,
  accessToken: string,
): Promise<{ onboarding_completed: boolean; user: AuthUser }> {
  return apiRequest<{ onboarding_completed: boolean; user: AuthUser }>(
    "/api/v1/onboarding/complete",
    {
      method: "POST",
      body: JSON.stringify(payload),
      accessToken,
    },
  );
}
