import React, { ReactNode, HTMLAttributes, useMemo } from "react";

type SkeletonVariant = "text" | "avatar" | "card";

interface LoadingSkeletonProps extends HTMLAttributes<HTMLDivElement> {
  /** When true the skeleton UI is shown, otherwise the children are rendered */
  isLoading: boolean;
  /** Content to render once loading is complete */
  children: ReactNode;
  /** Optional custom skeleton to render instead of the default variant */
  skeleton?: ReactNode;
  /** Choose a preset skeleton layout */
  variant?: SkeletonVariant;
  /** Number of lines for the text variant (default: 3) */
  lines?: number;
}

/**
 * Default skeleton implementations that follow the project's design system:
 * - Uses Tailwind's `animate-pulse` for the loading animation.
 * - Uses light gray in light mode and dark gray in dark mode.
 * - Rounded corners and appropriate spacing.
 * - Accessible via `aria-busy` and `aria-live`.
 */
const DefaultSkeleton = ({
  variant = "text",
  lines = 3,
}: {
  variant: SkeletonVariant;
  lines?: number;
}) => {
  const lineSkeleton = (key: number) => (
    <div
      key={key}
      className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full"
    />
  );

  switch (variant) {
    case "avatar":
      return (
        <div className="flex items-center space-x-4">
          <div className="w-12 h-12 rounded-full bg-gray-200 dark:bg-gray-700 animate-pulse" />
          <div className="flex-1 space-y-2 py-1">
            {Array.from({ length: 2 }).map((_, i) => (
              <div
                key={i}
                className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"
              />
            ))}
          </div>
        </div>
      );
    case "card":
      return (
        <div className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm animate-pulse">
          <div className="flex items-center space-x-4 mb-4">
            <div className="w-10 h-10 rounded-full bg-gray-200 dark:bg-gray-700" />
            <div className="flex-1 space-y-2">
              {Array.from({ length: 2 }).map((_, i) => (
                <div
                  key={i}
                  className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"
                />
              ))}
            </div>
          </div>
          <div className="space-y-2">
            {Array.from({ length: lines }).map((_, i) => (
              <div
                key={i}
                className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full"
              />
            ))}
          </div>
        </div>
      );
    case "text":
    default:
      return (
        <div className="space-y-2 animate-pulse">
          {Array.from({ length: lines }).map((_, i) => (
            <div
              key={i}
              className={i === lines - 1 ? "h-4 w-3/4" : "h-4 w-full"}
              className="h-4 bg-gray-200 dark:bg-gray-700 rounded"
            />
          ))}
        </div>
      );
  }
};

/**
 * LoadingSkeleton component
 *
 * Renders a skeleton UI while data is being fetched. Once `isLoading` becomes false,
 * the wrapped `children` are displayed. The component respects the project's
 * design system (Tailwind, color palette, animation) and is fully accessible.
 */
export const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  isLoading,
  children,
  skeleton,
  variant = "text",
  lines = 3,
  className = "",
  ...rest
}) => {
  const defaultSkeleton = useMemo(
    () => <DefaultSkeleton variant={variant} lines={lines} />,
    [variant, lines]
  );

  if (!isLoading) {
    return <>{children}</>;
  }

  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className={`animate-pulse ${className}`}
      {...rest}
    >
      {skeleton ?? defaultSkeleton}
    </div>
  );
};

export default LoadingSkeleton;