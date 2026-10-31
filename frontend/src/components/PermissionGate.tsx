import React, { FC, ReactNode } from "react";
import { useAuth } from "@/hooks/useAuth";

/**
 * PermissionGate
 *
 * Renders its children only when the authenticated user possesses at least one
 * of the roles specified in `allowedRoles`. While the authentication state is
 * loading, a subtle skeleton is shown to avoid layout shift. If the user does not
 * have the required role, an optional `fallback` UI is rendered (defaults to
 * nothing). The component is fully typed and follows the project's Tailwind‑based
 * design system.
 *
 * Example:
 * ```tsx
 * <PermissionGate allowedRoles={["admin", "accountant"]}>
 *   <SensitiveComponent />
 * </PermissionGate>
 * ```
 */
interface PermissionGateProps {
  /** Roles that are allowed to view the children. */
  allowedRoles: readonly string[];
  /** Content to render when the user has the required role(s). */
  children: ReactNode;
  /** Optional UI rendered when the user lacks permission. */
  fallback?: ReactNode;
}

/**
 * Helper to determine if the user has any of the required roles.
 */
const hasRequiredRole = (
  userRoles: readonly string[] | undefined,
  allowedRoles: readonly string[]
): boolean => {
  if (!userRoles) return false;
  return userRoles.some((role) => allowedRoles.includes(role));
};

export const PermissionGate: FC<PermissionGateProps> = ({
  allowedRoles,
  children,
  fallback = null,
}) => {
  const { user, loading, error } = useAuth();

  // While auth state is being resolved, show a lightweight loading placeholder.
  if (loading) {
    return (
      <div
        className="flex items-center space-x-2 text-sm text-gray-500 animate-pulse"
        aria-live="polite"
        aria-busy="true"
      >
        <svg
          className="w-4 h-4 text-gray-400"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8v8H4z"
          />
        </svg>
        <span>Loading permissions…</span>
      </div>
    );
  }

  // If there was an error fetching auth info, render a non‑intrusive error message.
  if (error) {
    return (
      <div
        className="text-sm text-red-600"
        role="alert"
        aria-live="assertive"
      >
        Unable to verify permissions.
      </div>
    );
  }

  const userRoles = user?.roles as readonly string[] | undefined;
  const isAllowed = hasRequiredRole(userRoles, allowedRoles);

  if (isAllowed) {
    return <>{children}</>;
  }

  // Render fallback UI (or nothing) when the user lacks the required role.
  return <>{fallback}</>;
};

PermissionGate.displayName = "PermissionGate";

export default PermissionGate;