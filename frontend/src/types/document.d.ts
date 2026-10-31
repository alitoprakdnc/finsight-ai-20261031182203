/**
 * FinSight AI – Document domain types
 *
 * This file defines the core TypeScript types used throughout the
 * frontend application to model financial documents, the data extracted
 * from them and the processing lifecycle status.
 *
 * The definitions mirror the Go models (`internal/models/document.go`,
 * `internal/models/audit_log.go`) and the OpenAPI specification
 * (`pkg/openapi/openapi.yaml`) to ensure type‑safety between client and
 * server.
 *
 * All timestamps are ISO‑8601 strings in UTC.
 * Monetary values are represented as numbers (float) in the smallest
 * currency unit (e.g. cents) to avoid floating‑point rounding issues.
 */

 /** Enum representing the processing state of a document. */
export enum DocumentStatus {
  /** Document has been uploaded but not yet queued for processing. */
  Pending = 'PENDING',
  /** Document is currently being processed (OCR / LLM). */
  Processing = 'PROCESSING',
  /** Processing completed successfully and data has been extracted. */
  Completed = 'COMPLETED',
  /** Processing failed – see `errorMessage` for details. */
  Failed = 'FAILED',
  /** Document has been reviewed and approved by an accountant. */
  Approved = 'APPROVED',
  /** Document has been reviewed and rejected (requires re‑upload). */
  Rejected = 'REJECTED',
}

/**
 * A single line‑item extracted from an invoice/receipt.
 *
 * All monetary fields are expressed in the document's currency and are
 * stored as numbers representing the smallest unit (e.g. cents). The UI
 * formats them according to the user's locale.
 */
export interface LineItem {
  /** Human‑readable description of the item (e.g. "Office chairs"). */
  description: string;
  /** Quantity purchased. */
  quantity: number;
  /** Unit price in the smallest currency unit. */
  unitPrice: number;
  /** Total price for this line (quantity * unitPrice). */
  total: number;
  /** Category assigned by the LLM (e.g. "Supplies", "Travel"). */
  category: string;
  /** Optional tax amount for this line item. */
  taxAmount?: number;
}

/**
 * Structured data produced by the OCR + LLM pipeline.
 *
 * All fields are optional because the extraction process may not
 * retrieve every piece of information for every document type.
 */
export interface ExtractedData {
  /** Raw OCR text extracted from the image/PDF. */
  rawText?: string;
  /** Resolved vendor name (e.g. "Acme Corp"). */
  vendor?: string;
  /** Invoice or receipt number identified in the document. */
  documentNumber?: string;
  /** Date the document was issued (ISO‑8601). */
  issueDate?: string;
  /** Payment due date (ISO‑8601). */
  dueDate?: string;
  /** Total amount payable in the smallest currency unit. */
  totalAmount?: number;
  /** Currency code (ISO 4217, e.g. "USD", "EUR"). */
  currency?: string;
  /** Total tax amount in the smallest currency unit. */
  taxAmount?: number;
  /** Array of line items parsed from the document. */
  lineItems?: LineItem[];
}

/**
 * Core document model used throughout the UI.
 *
 * The `extractedData` field is populated only after the document reaches
 * the `COMPLETED` status. Errors from the processing pipeline are exposed
 * via `errorMessage`.
 */
export interface Document {
  /** UUID of the document (primary key). */
  id: string;
  /** Identifier of the tenant that owns the document. */
  tenantId: string;
  /** Original filename supplied by the user. */
  filename: string;
  /** MIME type of the uploaded file (e.g. "application/pdf"). */
  contentType: string;
  /** Size of the file in bytes. */
  size: number;
  /** Object key used in MinIO (S3‑compatible) storage. */
  storageKey: string;
  /** Current processing status. */
  status: DocumentStatus;
  /** Timestamp when the file was uploaded (ISO‑8601). */
  uploadedAt: string;
  /** Timestamp when processing finished (ISO‑8601). Present only for COMPLETED/FAILED. */
  processedAt?: string;
  /** Structured data extracted from the document. */
  extractedData?: ExtractedData;
  /** Human‑readable error message if processing failed. */
  errorMessage?: string;
}

/**
 * Paginated response shape returned by `/api/v1/documents`.
 *
 * The UI uses this shape with React‑Query for infinite scrolling and
 * cursor‑based pagination.
 */
export interface DocumentListResponse {
  /** Array of documents for the current page. */
  data: Document[];
  /** Total number of documents matching the query (for UI pagination controls). */
  total: number;
  /** Zero‑based page index. */
  page: number;
  /** Number of items per page. */
  pageSize: number;
}

/**
 * Helper type to map a DocumentStatus to a user‑friendly label.
 * Used by UI components (e.g., status badges).
 */
export const DocumentStatusLabel: Record<DocumentStatus, string> = {
  [DocumentStatus.Pending]: 'Pending',
  [DocumentStatus.Processing]: 'Processing',
  [DocumentStatus.Completed]: 'Completed',
  [DocumentStatus.Failed]: 'Failed',
  [DocumentStatus.Approved]: 'Approved',
  [DocumentStatus.Rejected]: 'Rejected',
};

/**
 * Helper type to map a DocumentStatus to a Tailwind CSS colour class.
 * This keeps the visual representation consistent across the app.
 */
export const DocumentStatusColor: Record<DocumentStatus, string> = {
  [DocumentStatus.Pending]: 'bg-gray-200 text-gray-800',
  [DocumentStatus.Processing]: 'bg-blue-200 text-blue-800',
  [DocumentStatus.Completed]: 'bg-green-200 text-green-800',
  [DocumentStatus.Failed]: 'bg-red-200 text-red-800',
  [DocumentStatus.Approved]: 'bg-indigo-200 text-indigo-800',
  [DocumentStatus.Rejected]: 'bg-yellow-200 text-yellow-800',
};