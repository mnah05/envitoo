import { useMemo, useRef } from "react";
import { useAuth } from "@clerk/clerk-react";

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.status = status;
    this.body = body;
    this.name = "ApiError";
  }
}

type Query = Record<string, string | number | boolean | undefined | null>;

function buildUrl(path: string, query?: Query): string {
  const url = `${BASE_URL}${path}`;
  if (!query) return url;
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && v !== "") params.set(k, String(v));
  }
  const qs = params.toString();
  return qs ? `${url}?${qs}` : url;
}

/** Read a user-facing message from an error body, falling back to status text. */
export function errorBodyMessage(body: unknown, fallback: string): string {
  if (body && typeof body === "object") {
    const b = body as Record<string, unknown>;
    if (typeof b.message === "string") return b.message;
    if (typeof b.error === "string") return b.error;
  }
  if (typeof body === "string") return body;
  return fallback;
}

interface ApiMethods {
  get: <T>(path: string, query?: Query) => Promise<T>;
  post: <T>(path: string, body?: unknown) => Promise<T>;
  patch: <T>(path: string, body?: unknown) => Promise<T>;
  del: <T>(path: string, query?: Query) => Promise<T>;
}

/**
 * Build a fetcher bound to the Clerk session token. Returns a stable object
 * (memoized) so it is safe to reference inside TanStack Query dependencies.
 */
export function useApi(): ApiMethods {
  const { getToken } = useAuth();
  const getTokenRef = useRef(getToken);
  getTokenRef.current = getToken;

  return useMemo<ApiMethods>(() => {
    async function request<T>(
      method: string,
      path: string,
      opts: { query?: Query; body?: BodyInit; parse?: "json" | "empty" } = {},
    ): Promise<T> {
      const { query, body, parse = "json" } = opts;
      const token = await getTokenRef.current();
      const headers: Record<string, string> = {};
      if (token) headers.Authorization = `Bearer ${token}`;
      if (body != null) headers["Content-Type"] = "application/json";

      const res = await fetch(buildUrl(path, query), { method, headers, body });

      if (!res.ok) {
        let errorBody: unknown;
        try {
          errorBody = await res.json();
        } catch {
          try {
            errorBody = await res.text();
          } catch {
            errorBody = undefined;
          }
        }
        const message = errorBodyMessage(errorBody, res.statusText || "request failed");
        throw new ApiError(res.status, message, errorBody);
      }

      if (parse === "empty" || res.status === 204) return undefined as T;
      return (await res.json()) as T;
    }

    return {
      get: (path, query) => request("GET", path, { query }),
      post: (path, body) => request("POST", path, { body: body != null ? JSON.stringify(body) : undefined }),
      patch: (path, body) => request("PATCH", path, { body: body != null ? JSON.stringify(body) : undefined }),
      del: (path, query) => request("DELETE", path, { query }),
    };
  }, []);
}

export type Api = ApiMethods;
