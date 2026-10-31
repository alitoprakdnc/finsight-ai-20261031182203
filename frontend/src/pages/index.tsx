import { NextPage } from 'next';
import Head from 'next/head';
import { useRouter } from 'next/router';
import { useEffect } from 'react';
import Layout from '@/components/Layout';
import { useAuth } from '@/hooks/useAuth';
import { ArrowRightIcon } from '@heroicons/react/24/solid';
import {
  CloudArrowUpIcon,
  DocumentMagnifyingGlassIcon,
  ShieldCheckIcon,
  Cog8ToothIcon,
  ChartBarIcon,
  UsersIcon,
} from '@heroicons/react/24/outline';

interface Feature {
  title: string;
  description: string;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
}

const features: Feature[] = [
  {
    title: 'Secure Multi‑Tenant Upload',
    description:
      'Client‑side encryption and isolated storage keep every tenant’s data private.',
    icon: CloudArrowUpIcon,
  },
  {
    title: 'AI‑Enhanced Extraction',
    description:
      'Tesseract OCR + LLM categorisation delivers structured data with vendor‑level accuracy.',
    icon: DocumentMagnifyingGlassIcon,
  },
  {
    title: 'Audit‑Ready Logs',
    description:
      'Immutable event trail stored in PostgreSQL and streamed to Redis for real‑time visibility.',
    icon: ShieldCheckIcon,
  },
  {
    title: 'Configurable Workflows',
    description:
      'Role‑based approvals, custom rules and webhook integrations adapt to any process.',
    icon: Cog8ToothIcon,
  },
  {
    title: 'Real‑Time Dashboards',
    description:
      'Spend trends, pending approvals and KPI charts keep finance teams informed.',
    icon: ChartBarIcon,
  },
  {
    title: 'Enterprise‑Grade Controls',
    description:
      'RBAC, OIDC, secret‑manager integration and comprehensive metrics meet compliance standards.',
    icon: UsersIcon,
  },
];

const Home: NextPage = () => {
  const { user, login, loading } = useAuth();
  const router = useRouter();

  // Redirect authenticated users straight to the dashboard
  useEffect(() => {
    if (!loading && user) {
      router.replace('/dashboard');
    }
  }, [user, loading, router]);

  const handleSignIn = async () => {
    try {
      await login();
    } catch (err) {
      // In a production app we would surface this via a toast
      console.error('Sign‑in failed:', err);
    }
  };

  return (
    <Layout>
      <Head>
        <title>
          FinSight AI – Automated Financial Document Processing
        </title>
        <meta
          name="description"
          content="Secure, AI‑enhanced bookkeeping automation for invoices, receipts and contracts. Eliminate manual data entry, reduce errors and stay compliant."
        />
      </Head>

      <main className="flex flex-col items-center justify-center min-h-screen px-4 py-12 bg-gray-50 dark:bg-gray-900">
        {/* Hero */}
        <section className="max-w-4xl text-center">
          <h1 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-5xl">
            Automate Your Financial Documents with AI
          </h1>
          <p className="mt-6 text-lg text-gray-600 dark:text-gray-300">
            FinSight AI securely extracts, enriches and books invoices, receipts and
            contracts so your finance team can focus on insights, not data entry.
          </p>

          <div className="mt-8 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
            <button
              type="button"
              onClick={handleSignIn}
              disabled={loading}
              aria-label="Sign in to FinSight AI"
              className="inline-flex items-center justify-center rounded-md bg-primary-600 hover:bg-primary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600 px-6 py-3 text-base font-medium text-white disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Sign In
              <ArrowRightIcon className="ml-2 h-5 w-5" aria-hidden="true" />
            </button>
          </div>
        </section>

        {/* Features */}
        <section className="mt-16 w-full max-w-5xl">
          <h2 className="text-2xl font-semibold text-gray-900 dark:text-white text-center">
            Why FinSight AI?
          </h2>
          <div className="mt-8 grid gap-8 md:grid-cols-2 lg:grid-cols-3">
            {features.map((f) => (
              <div
                key={f.title}
                className="flex flex-col items-center p-6 text-center bg-white dark:bg-gray-800 rounded-lg shadow-sm"
              >
                <f.icon className="h-12 w-12 text-primary-600" aria-hidden="true" />
                <h3 className="mt-4 text-lg font-medium text-gray-900 dark:text-white">
                  {f.title}
                </h3>
                <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">
                  {f.description}
                </p>
              </div>
            ))}
          </div>
        </section>
      </main>
    </Layout>
  );
};

export default Home;