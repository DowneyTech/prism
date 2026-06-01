"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

interface User {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
}

interface AuthState {
  user: User | null;
  setAuth: (token: string, user: User) => void;
  clearAuth: () => void;
  isAuthenticated: () => boolean;
}

function setTokenCookie(token: string) {
  document.cookie = `token=${encodeURIComponent(token)}; path=/; max-age=${7 * 24 * 3600}; SameSite=Lax`;
}

function clearTokenCookie() {
  document.cookie = "token=; path=/; max-age=0; SameSite=Lax";
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      setAuth: (token, user) => {
        setTokenCookie(token);
        set({ user });
      },
      clearAuth: () => {
        clearTokenCookie();
        set({ user: null });
      },
      isAuthenticated: () => {
        if (get().user !== null) return true;
        if (typeof document === "undefined") return false;
        return /(?:^|;\s*)token=/.test(document.cookie);
      },
    }),
    {
      name: "prism-auth",
      partialize: (state) => ({ user: state.user }),
    }
  )
);
