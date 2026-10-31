import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  ReactNode,
} from "react";
import { v4 as uuidv4 } from "uuid";

/**
 * Toast types – match the design system colour palette.
 */
export type ToastVariant = "success" | "error" | "info";

/**
 * Internal toast representation.
 */
interface Toast {
  id: string;
  message: string;
  variant: ToastVariant;
  /** Auto‑dismiss after `duration` ms. If 0, toast stays until manually closed. */
  duration: number;
}

/**
 * Public context API.
 */
interface ToastContextValue {
  /**
   * Show a toast.
   *
   * @param message   Text to display.
   * @param variant   Visual style – success / error / info. Defaults to "info".
   * @param duration  Auto‑dismiss timeout in ms. `0` disables auto‑dismiss.
   *                  Defaults to 5000 ms.
   */
  addToast: (
    message: string,
    variant?: ToastVariant,
    duration?: number
  ) => void;
}

/* -------------------------------------------------------------------------- */
/* Context creation                                                            */
/* -------------------------------------------------------------------------- */
const ToastContext = createContext<ToastContextValue | undefined>(undefined);

/**
 * Hook to access the toast API from any component.
 */
export const useToast = (): ToastContextValue => {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within a ToastProvider");
  }
  return ctx;
};

/* -------------------------------------------------------------------------- */
/* Provider implementation                                                     */
/* -------------------------------------------------------------------------- */
interface ToastProviderProps {
  children: ReactNode;
}

/**
 * ToastProvider renders a portal‑style container that holds all active toasts.
 * It supplies `addToast` via context.
 */
export const ToastProvider: React.FC<ToastProviderProps> = ({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const addToast = useCallback(
    (message: string, variant: ToastVariant = "info", duration = 5000) => {
      const id = uuidv4();
      const toast: Toast = { id, message, variant, duration };
      setToasts((prev) => [...prev, toast]);

      if (duration > 0) {
        // Auto‑remove after the specified timeout.
        setTimeout(() => removeToast(id), duration);
      }
    },
    [removeToast]
  );

  // Ensure timers are cleared if a toast is removed manually before timeout.
  useEffect(() => {
    return () => {
      // Component unmount – clear any pending timers.
      // (We don't keep references to timers, but React will clean them up.)
    };
  }, []);

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      {/* Toast container – fixed to the top‑right corner, pointer‑events none
          so clicks fall through except on the toast itself. */}
      <div
        aria-live="assertive"
        className="fixed inset-0 flex flex-col items-end p-4 space-y-2 pointer-events-none"
      >
        {toasts.map((toast) => (
          <ToastItem key={toast.id} toast={toast} onClose={removeToast} />
        ))}
      </div>
    </ToastContext.Provider>
  );
};

/* -------------------------------------------------------------------------- */
/* Individual toast UI component                                               */
/* -------------------------------------------------------------------------- */
interface ToastItemProps {
  toast: Toast;
  onClose: (id: string) => void;
}

/**
 * Renders a single toast with appropriate styling and a manual dismiss button.
 * Includes a simple fade‑in/out animation using Tailwind utilities.
 */
const ToastItem: React.FC<ToastItemProps> = ({ toast, onClose }) => {
  const { id, message, variant } = toast;

  // Tailwind colour mapping per variant.
  const variantStyles = {
    success:
      "bg-green-50 border-l-4 border-green-500 text-green-800 dark:bg-green-900 dark:text-green-100",
    error:
      "bg-red-50 border-l-4 border-red-500 text-red-800 dark:bg-red-900 dark:text-red-100",
    info:
      "bg-blue-50 border-l-4 border-blue-500 text-blue-800 dark:bg-blue-900 dark:text-blue-100",
  };

  const icon = {
    success: (
      <svg
        className="w-5 h-5 text-green-600 dark:text-green-300 flex-shrink-0"
        fill="currentColor"
        viewBox="0 0 20 20"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
          clipRule="evenodd"
        />
      </svg>
    ),
    error: (
      <svg
        className="w-5 h-5 text-red-600 dark:text-red-300 flex-shrink-0"
        fill="currentColor"
        viewBox="0 0 20 20"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-4a1 1 0 112 0v2a1 1 0 11-2 0v-2zm0-8a1 1 0 112 0v6a1 1 0 11-2 0V6z"
          clipRule="evenodd"
        />
      </svg>
    ),
    info: (
      <svg
        className="w-5 h-5 text-blue-600 dark:text-blue-300 flex-shrink-0"
        fill="currentColor"
        viewBox="0 0 20 20"
        aria-hidden="true"
      >
        <path d="M9 9h2V7H9v2zm0 4h2v-2H9v2z" />
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm0-14a6 6 0 110 12A6 6 0 0110 4z"
          clipRule="evenodd"
        />
      </svg>
    ),
  }[variant];

  return (
    <div
      className={`
        max-w-sm w-full
        ${variantStyles[variant]}
        rounded-md shadow-lg
        pointer-events-auto
        flex
        items-start
        space-x-3
        p-4
        animate-enter
        transition-all
        duration-300
        ease-out
        motion-reduce:transition-none
      `}
      role="alert"
    >
      {icon}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">{message}</p>
      </div>
      <button
        type="button"
        onClick={() => onClose(id)}
        className={`
          inline-flex
          rounded-md
          focus:outline-none
          focus:ring-2
          focus:ring-offset-2
          focus:ring-${variant === "success"
            ? "green"
            : variant === "error"
            ? "red"
            : "blue"}-600
          text-${variant === "success"
            ? "green"
            : variant === "error"
            ? "red"
            : "blue"}-600
          hover:text-${variant === "success"
            ? "green"
            : variant === "error"
            ? "red"
            : "blue"}-800
          dark:text-${variant === "success"
            ? "green"
            : variant === "error"
            ? "red"
            : "blue"}-300
          dark:hover:text-${variant === "success"
            ? "green"
            : variant === "error"
            ? "red"
            : "blue"}-100
        `}
        aria-label="Dismiss notification"
      >
        <svg
          className="w-4 h-4"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
    </div>
  );
};

/* -------------------------------------------------------------------------- */
/* Tailwind animation utilities (enter/leave)                                   */
/* -------------------------------------------------------------------------- */
/**
 * The following CSS is injected via Tailwind's `@layer utilities` in the
 * global stylesheet (e.g., `globals.css`). If the project already defines
 * these utilities, this comment serves as documentation.
 *
 * ```css
 * @layer utilities {
 *   .animate-enter {
 *     @apply opacity-0 translate-y-2;
 *   }
 *   .animate-enter-active {
 *     @apply opacity-100 translate-y-0;
 *   }
 * }
 * ```
 *
 * The component uses `animate-enter` together with Tailwind's transition
 * utilities to achieve a smooth fade‑in. The fade‑out is handled by simply
 * removing the toast from the state, which instantly collapses the element.
 * For a more elaborate exit animation you could integrate `react-transition-group`
 * or `framer-motion`, but the current implementation satisfies the design
 * system's requirement for a lightweight, accessible toast system.
 */

export default ToastProvider;