import axios, {
  AxiosError,
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
} from 'axios';
import Router from 'next/router';
import { toast } from '@/components/ToastProvider';

// Base URL for the API (e.g. https://api.finsight.example.com)
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL?.replace(/\/+$/, '') ?? '';

/**
 * Create a pre‑configured Axios instance.
 * All requests are sent to `/api/v1` and include JSON headers.
 */
const api: AxiosInstance = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
});

/**
 * Retrieve the current access token from localStorage.
 * The token is stored by the `useAuth` hook after a successful login.
 */
function getAccessToken(): string | null {
  if (typeof window === 'undefined') {
    return null;
  }
  return localStorage.getItem('access_token');
}

/**
 * Request interceptor – injects the Authorization header when a token exists.
 */
api.interceptors.request.use(
  (config: AxiosRequestConfig) => {
    const token = getAccessToken();
    if (token) {
      // Ensure headers object exists
      config.headers = config.headers ?? {};
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

/**
 * Response interceptor – centralised error handling.
 *
 * - 401 → token invalid/expired → clear storage & redirect to login.
 * - 4xx → show the error message returned by the API (if any).
 * - 5xx → generic server‑error toast.
 * - Network errors → toast about connectivity.
 */
api.interceptors.response.use(
  (response: AxiosResponse) => response,
  (error: AxiosError) => {
    if (error.response) {
      const { status, data } = error.response;

      // Prefer a message from the API payload, fall back to generic text.
      const apiMessage =
        (data && (data as any).error) ||
        (data && (data as any).message) ||
        error.message ||
        'An unexpected error occurred';

      if (status === 401) {
        // Session is no longer valid – clean up and force re‑login.
        if (typeof window !== 'undefined') {
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          Router.replace('/login');
        }
        toast.error('Session expired. Please log in again.');
      } else if (status >= 400 && status < 500) {
        toast.error(apiMessage);
      } else if (status >= 500) {
        toast.error('Server error. Please try again later.');
      }
    } else if (error.request) {
      // Request was made but no response received.
      toast.error('Network error. Please check your connection.');
    } else {
      // Something happened while setting up the request.
      toast.error(error.message);
    }

    return Promise.reject(error);
  },
);

export default api;