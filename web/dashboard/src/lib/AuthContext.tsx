import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { api } from "./api";

interface AuthState {
  status: "loading" | "needs-setup" | "logged-out" | "logged-in";
  email?: string;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthState["status"]>("loading");
  const [email, setEmail] = useState<string | undefined>();

  const refresh = async () => {
    const setupStatus = await api.setupStatus();
    if (setupStatus.needsSetup) {
      setStatus("needs-setup");
      return;
    }
    try {
      const me = await api.me();
      setEmail(me.email);
      setStatus("logged-in");
    } catch {
      setStatus("logged-out");
    }
  };

  const logout = async () => {
    await api.logout();
    setEmail(undefined);
    setStatus("logged-out");
  };

  useEffect(() => {
    refresh();
  }, []);

  return <AuthContext.Provider value={{ status, email, refresh, logout }}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
