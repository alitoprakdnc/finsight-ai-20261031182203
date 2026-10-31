import { format as formatDateFns } from 'date-fns';
import { DocumentStatus } from '@/types/document';

/**
 * Formats a numeric value as a localized currency string.
 *
 * @param value   - The amount to format. Can be a number or a numeric string.
 * @param locale  - BCP 47 locale identifier (default: 'en-US').
 * @param currency- ISO 4217 currency code (default: 'USD').
 * @returns A formatted currency string or a placeholder (“-”) when the value is invalid.
 */
export function formatCurrency(
  value: number | string,
  locale = 'en-US',
  currency = 'USD',
): string {
  if (value === null || value === undefined) {
    return '-';
  }

  const numericValue =
    typeof value === 'string' ? parseFloat(value.replace(/[^0-9.-]+/g, '')) : value;

  if (Number.isNaN(numericValue)) {
    return '-';
  }

  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(numericValue);
  } catch {
    // Fallback to a simple representation if Intl fails for any reason.
    return numericValue.toFixed(2);
  }
}

/**
 * Formats a Date, ISO string or timestamp into a human‑readable string.
 *
 * @param date        - Date object, ISO‑8601 string, or timestamp.
 * @param dateFormat  - date‑fns format string (default: 'PPpp' → e.g. Jan 1, 2024, 12:00 PM).
 * @returns Formatted date string or a placeholder (“-”) when the input is invalid.
 */
export function formatDate(
  date: Date | string | number,
  dateFormat = 'PPpp',
): string {
  if (date === null || date === undefined) {
    return '-';
  }

  let parsedDate: Date;

  if (date instanceof Date) {
    parsedDate = date;
  } else {
    parsedDate = new Date(date);
  }

  if (Number.isNaN(parsedDate.getTime())) {
    return '-';
  }

  try {
    return formatDateFns(parsedDate, dateFormat);
  } catch {
    // In case the format string is invalid.
    return parsedDate.toLocaleString();
  }
}

/**
 * Returns UI‑friendly badge information for a document status.
 *
 * The returned object contains:
 *  - `label`    : Human readable text.
 *  - `bgClass`  : Tailwind background colour class.
 *  - `textClass`: Tailwind text colour class.
 *
 * These classes are aligned with the project's design system palette.
 *
 * @param status - Document status enum value.
 */
export function getStatusBadge(
  status: DocumentStatus,
): { label: string; bgClass: string; textClass: string } {
  switch (status) {
    case 'pending':
      return {
        label: 'Pending',
        bgClass: 'bg-yellow-100',
        textClass: 'text-yellow-800',
      };
    case 'processing':
      return {
        label: 'Processing',
        bgClass: 'bg-blue-100',
        textClass: 'text-blue-800',
      };
    case 'completed':
      return {
        label: 'Completed',
        bgClass: 'bg-green-100',
        textClass: 'text-green-800',
      };
    case 'approved':
      return {
        label: 'Approved',
        bgClass: 'bg-indigo-100',
        textClass: 'text-indigo-800',
      };
    case 'rejected':
      return {
        label: 'Rejected',
        bgClass: 'bg-red-100',
        textClass: 'text-red-800',
      };
    case 'failed':
      return {
        label: 'Failed',
        bgClass: 'bg-rose-100',
        textClass: 'text-rose-800',
      };
    default:
      return {
        label: 'Unknown',
        bgClass: 'bg-gray-100',
        textClass: 'text-gray-800',
      };
  }
}