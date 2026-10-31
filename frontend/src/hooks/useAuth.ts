import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  ReactNode,
} from 'react';
import jwtDecode from 'jwt-decode';
import api from '@/utils/api';

interface JwtPayload {
  sub: string;
  email?: string;
  role?: string;
  exp: number;
  iat: number;
  [key: string]: any;
}

interface AuthContextProps {
  /** Decoded JWT payload of the current user or `null` if unauthenticated */
  user: JwtPayload | null;
  /** Raw access token string */
  token: string | null;
  /** True while an auth request (login/refresh) is in progress */
  loading: boolean;
  /** Convenience boolean – true when a valid token exists */
  isAuthenticated: boolean;
  /** Perform login with username/password */
  login: (username: string, password: string) => Promise<void>;
  /** Perform logout – clears local state and informs the backend */
  logout: () => Promise<void>;
}

/** Default empty context – will be overridden by the provider */
const AuthContext = createContext<AuthContextProps>({
  user: null,
  token: null,
  loading: false,
  isAuthenticated: false,
  login: async () => {},
  logout: async () => {},
});

/**
 * AuthProvider – wraps the application and supplies authentication state.
 *
 * It stores the access token in `localStorage` (client‑side) and expects the
 * refresh token to be managed by the backend via an httpOnly cookie.
 *
 * The provider:
 *   • Loads any persisted token on mount and validates its expiry.
 *   • Schedules an automatic refresh one minute before expiry.
 *   • Exposes `login`/`logout` helpers.
 *   • Installs Axios interceptors to attach the token and transparently
 *     retry a single request after a successful refresh.
 */
export const AuthProvider = ({
  children,
}: {
  children: ReactNode;
}): JSX.Element => {
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<JwtPayload | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const refreshTimeoutRef = React.useRef<NodeJS.Timeout | null>(null);

  /** Persist token and derived user payload */
  const setSession = useCallback((newToken: string | null) => {
    if (newToken) {
      localStorage.setItem('access_token', newToken);
      const decoded = jwtDecode<JwtPayload>(newToken);
      setUser(decoded);
      setToken(newToken);
    } else {
      localStorage.removeItem('access_token');
      setUser(null);
      setToken(null);
    }
  }, []);

  /** Schedule a token refresh one minute before expiry */
  const scheduleRefresh = useCallback((exp: number) => {
    if (refreshTimeoutRef.current) {
      clearTimeout(refreshTimeoutRef.current);
    }
    const now = Date.now() / 1000;
    const delayMs = Math.max((exp - now - 60) * 1000, 0);
    refreshTimeoutRef.current = setTimeout(() => {
      void refreshToken();
    }, delayMs);
  }, []);

  /** Refresh the access token using the backend endpoint */
  const refreshToken = useCallback(async () => {
    try {
      const resp = await api.post<{ access_token: string }>('/auth/refresh', null, {
        withCredentials: true, // send httpOnly refresh cookie
      });
      const newToken = resp.data.access_token;
      setSession(newToken);
      const decoded = jwtDecode<JwtPayload>(newToken);
      scheduleRefresh(decoded.exp);
    } catch (err) {
      // If refresh fails we treat the user as logged out
      console.error('Token refresh failed', err);
      await logout();
    }
  }, [setSession, scheduleRefresh]);

  /** Login flow – exchange credentials for an access token */
  const login = useCallback(
    async (username: string, password: string) => {
      setLoading(true);
      try {
        const resp = await api.post<{ access_token: string }>('/auth/login', {
          username,
          password,
        });
        const newToken = resp.data.access_token;
        setSession(newToken);
        const decoded = jwtDecode<JwtPayload>(newToken);
        scheduleRefresh(decoded.exp);
      } finally {
        setLoading(false);
      }
    },
    [setSession, scheduleRefresh],
  );

  /** Logout flow – clear local state and inform the backend */
  const logout = useCallback(async () => {
    setLoading(true);
    try {
      await api.post('/auth/logout', null, { withCredentials: true });
    } catch (e) {
      // ignore backend errors on logout; we still want to clear client state
      console.warn('Logout request failed', e);
    } finally {
      if (refreshTimeoutRef.current) {
        clearTimeout(refreshTimeoutRef.current);
      }
      setSession(null);
      setLoading(false);
    }
  }, [setSession]);

  /** Initialise auth state from persisted token */
  useEffect(() => {
    const stored = localStorage.getItem('access_token');
    if (stored) {
      try {
        const decoded = jwtDecode<JwtPayload>(stored);
        const now = Date.now() / 1000;
        if (decoded.exp > now) {
          setToken(stored);
          setUser(decoded);
          scheduleRefresh(decoded.exp);
        } else {
          // Token expired – attempt a silent refresh
          void refreshToken();
        }
      } catch (e) {
        console.warn('Invalid stored token, clearing', e);
        setSession(null);
      }
    }
    setLoading(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /** Axios interceptors – attach token & retry once on 401 */
  useEffect(() => {
    const requestInterceptor = api.interceptors.request.use(
      (config) => {
        if (token) {
          config.headers = {
            ...config.headers,
            Authorization: `Bearer ${token}`,
          };
        }
        return config;
      },
      (error) => Promise.reject(error),
    );

    const responseInterceptor = api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;
          try {
            await refreshToken();
            if (token) {
              originalRequest.headers.Authorization = `Bearer ${token}`;
            }
            return api(originalRequest);
          } catch (refreshError) {
            await logout();
            return Promise.reject(refreshError);
          }
        }
        return Promise.reject(error);
      },
    );

    return () => {
      api.interceptors.request.eject(requestInterceptor);
      api.interceptors.response.eject(responseInterceptor);
    };
  }, [token, refreshToken, logout]);

  const contextValue: AuthContextProps = {
    user,
    token,
    loading,
    isAuthenticated: !!token && !!user,
    login,
    logout,
  };

  return <AuthContext.Provider value={contextValue}>{children}</AuthContext.Provider>;
};

/**
 * Hook to access authentication state and actions.
 *
 * Must be used within an `AuthProvider`.
 */
export const useAuth = (): AuthContextProps => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export default useAuth;