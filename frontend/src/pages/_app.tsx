import type { AppProps } from 'next/app';
import { ReactNode, useEffect } from 'react';
import { ThemeProvider } from 'next-themes';
import Layout from '../components/Layout';
import { ToastProvider } from '../components/ToastProvider';
import '../styles/globals.css';

/**
 * Simple error boundary to avoid a complete UI crash.
 * It logs the error to the console (or a remote logger) and
 * displays a friendly fallback UI.
 */
class ErrorBoundary extends React.Component<{ children: ReactNode }, { hasError: boolean }> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(_: Error) {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // In production you would forward this to an external monitoring service.
    // eslint-disable-next-line no-console
    console.error('Uncaught error in React tree:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
          <div className="max-w-md text-center">
            <h1 className="mb-4 text-2xl font-bold text-gray-900 dark:text-gray-100">
              Something went wrong
            </h1>
            <p className="text-gray-600 dark:text-gray-400">
              An unexpected error occurred. Please refresh the page or contact support if the
              problem persists.
            </p>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

/**
 * Custom Next.js App component.
 *
 * - Injects Tailwind global styles.
 * - Provides dark‑mode support via `next-themes`.
 * - Wraps every page with the shared `Layout`.
 * - Supplies a global toast system.
 * - Adds a top‑level error boundary for resilience.
 *
 * Individual pages can export a `getLayout` function to override the default layout:
 *
 *   const Page = () => { … };
 *   Page.getLayout = (page) => <CustomLayout>{page}</CustomLayout>;
 *
 * If `getLayout` is not defined, the standard `Layout` component is used.
 */
function MyApp({ Component, pageProps }: AppProps) {
  // Ensure the theme is applied as early as possible to avoid flash of unstyled content.
  useEffect(() => {
    const html = document.documentElement;
    if (!html.classList.contains('light') && !html.classList.contains('dark')) {
      // Default to system preference; next-themes will sync later.
      html.classList.add('light');
    }
  }, []);

  const getLayout =
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (Component as any).getLayout ||
    ((page: ReactNode) => <Layout>{page}</Layout>);

  return (
    <ErrorBoundary>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
        <ToastProvider>{getLayout(<Component {...pageProps} />)}</ToastProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}

export default MyApp;