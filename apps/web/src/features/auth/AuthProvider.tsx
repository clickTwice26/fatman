"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  logoutSession,
  refreshSession,
  type AuthUser,
} from "@/lib/api";

interface AuthContextValue {
  user: AuthUser | null;
  accessToken: string | null;
  loading: boolean;
  setSession: (accessToken: string, user: AuthUser) => void;
  setUser: (user: AuthUser) => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUserState] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    refreshSession()
      .then((res) => {
        if (!active) return;
        setAccessToken(res.access_token);
        setUserState(res.user);
      })
      .catch(() => {
        if (!active) return;
        setAccessToken(null);
        setUserState(null);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const setSession = useCallback((token: string, nextUser: AuthUser) => {
    setAccessToken(token);
    setUserState(nextUser);
  }, []);

  const setUser = useCallback((nextUser: AuthUser) => {
    setUserState(nextUser);
  }, []);

  const logout = useCallback(async () => {
    const token = accessToken;
    setAccessToken(null);
    setUserState(null);
    if (token) {
      await logoutSession(token).catch(() => undefined);
    }
  }, [accessToken]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      accessToken,
      loading,
      setSession,
      setUser,
      logout,
    }),
    [user, accessToken, loading, setSession, setUser, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
