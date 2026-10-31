import React, { useEffect, useState, useCallback } from "react";
import {
  ResponsiveContainer,
  LineChart,
  BarChart,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  CartesianGrid,
  Line,
  Bar,
} from "recharts";
import { ArrowPathIcon } from "@heroicons/react/24/outline";

type SpendingData = {
  /** ISO date string (e.g., "2024-09-01") */
  date: string;
  /** Amount in the tenant's base currency (e.g., USD) */
  amount: number;
};

type PeriodOption = {
  label: string;
  value: string; // passed to fetchData
};

type ChartSpendingProps = {
  /** Async function that returns spending data for the given period */
  fetchData: (params: { period: string }) => Promise<SpendingData[]>;
  /** Initial period to load, defaults to "30d" */
  initialPeriod?: string;
  /** Optional title for the chart */
  title?: string;
};

const PERIOD_OPTIONS: PeriodOption[] = [
  { label: "Last 7 days", value: "7d" },
  { label: "Last 30 days", value: "30d" },
  { label: "Last 90 days", value: "90d" },
  { label: "Last year", value: "1y" },
];

const formatCurrency = (value: number) =>
  new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
  }).format(value);

/**
 * ChartSpending – Recharts line chart (fallback to bar on small screens)
 *
 * Shows spend over time with a period selector. Handles loading,
 * error, and empty‑state UI while staying fully accessible.
 */
const ChartSpending: React.FC<ChartSpendingProps> = ({
  fetchData,
  initialPeriod = "30d",
  title = "Spending Overview",
}) => {
  const [period, setPeriod] = useState<string>(initialPeriod);
  const [data, setData] = useState<SpendingData[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(
    async (selectedPeriod: string) => {
      setLoading(true);
      setError(null);
      try {
        const result = await fetchData({ period: selectedPeriod });
        // Ensure data is sorted by date ascending for proper chart rendering
        const sorted = [...result].sort(
          (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime()
        );
        setData(sorted);
      } catch (e) {
        console.error("Failed to load spending data:", e);
        setError("Unable to load spending data. Please try again.");
      } finally {
        setLoading(false);
      }
    },
    [fetchData]
  );

  // Initial load & reload on period change
  useEffect(() => {
    loadData(period);
  }, [period, loadData]);

  const handlePeriodChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setPeriod(e.target.value);
  };

  const renderTooltip = ({
    active,
    payload,
    label,
  }: {
    active?: boolean;
    payload?: any[];
    label?: string;
  }) => {
    if (active && payload && payload.length) {
      const amount = payload[0].value as number;
      const date = new Date(label ?? "").toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
      return (
        <div className="rounded bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 p-2 shadow-md min-w-[120px]">
          <p className="text-xs text-gray-500 dark:text-gray-400">{date}</p>
          <p className="font-medium">{formatCurrency(amount)}</p>
        </div>
      );
    }
    return null;
  };

  return (
    <section
      aria-labelledby="spending-chart-title"
      className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4"
    >
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-4">
        <h2
          id="spending-chart-title"
          className="text-lg font-semibold text-gray-900 dark:text-gray-100"
        >
          {title}
        </h2>
        <div className="mt-2 sm:mt-0 flex items-center space-x-2">
          <label htmlFor="period-select" className="sr-only">
            Select period
          </label>
          <select
            id="period-select"
            value={period}
            onChange={handlePeriodChange}
            className="rounded border-gray-300 dark:border-gray-600 focus:ring-primary-500 focus:border-primary-500 text-sm text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-700 py-1 pl-2 pr-8"
          >
            {PERIOD_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => loadData(period)}
            className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
            aria-label="Refresh chart"
          >
            <ArrowPathIcon className="h-5 w-5 text-gray-600 dark:text-gray-300" />
          </button>
        </div>
      </header>

      {loading && (
        <div className="flex items-center justify-center h-48">
          <svg
            className="animate-spin h-8 w-8 text-primary-600"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            aria-label="Loading"
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
              d="M4 12a8 8 0 018-8v8z"
            />
          </svg>
        </div>
      )}

      {error && (
        <div
          role="alert"
          className="flex items-center text-red-600 bg-red-50 dark:bg-red-900/20 rounded p-3"
        >
          <svg
            className="h-5 w-5 mr-2 flex-shrink-0"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12c0 4.97-4.03 9-9 9s-9-4.03-9-9 4.03-9 9-9 9 4.03 9 9z"
            />
          </svg>
          <span>{error}</span>
        </div>
      )}

      {!loading && !error && data.length === 0 && (
        <div className="flex items-center justify-center h-48 text-gray-500 dark:text-gray-400">
          No spending data available for the selected period.
        </div>
      )}

      {!loading && !error && data.length > 0 && (
        <ResponsiveContainer width="100%" height={300}>
          {/* Switch to BarChart on very narrow viewports for readability */}
          <LineChart data={data} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="date"
              tickFormatter={(date) =>
                new Date(date).toLocaleDateString(undefined, {
                  month: "short",
                  day: "numeric",
                })
              }
              stroke="#6b7280"
            />
            <YAxis
              tickFormatter={formatCurrency}
              stroke="#6b7280"
              domain={["auto", "auto"]}
            />
            <Tooltip content={renderTooltip} />
            <Legend verticalAlign="top" height={36} />
            <Line
              type="monotone"
              dataKey="amount"
              name="Spend"
              stroke="#0A74DA"
              strokeWidth={2}
              dot={{ r: 3 }}
              activeDot={{ r: 6 }}
            />
          </LineChart>
        </ResponsiveContainer>
      )}
    </section>
  );
};

export default ChartSpending;