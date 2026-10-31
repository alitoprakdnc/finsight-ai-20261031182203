// frontend/src/types/api.d.ts

/**
 * Core API response envelope used across the backend.
 * All successful responses include a `status` field set to "success"
 * and a `data` payload. Errors are returned with `status: "error"`
 * and an `error` message.
 */
export interface ApiResponse<T = unknown> {
  /** Indicates request outcome: "success" or "error". */
  status: 'success' | 'error';
  /** Payload for successful requests. Omitted when `status` is "error". */
  data?: T;
  /** Human‑readable error message when `status` is "error". */
  error?: string;
}

/**
 * Pagination metadata for cursor‑based pagination.
 */
export interface PaginationMeta {
  /** Cursor pointing to the next page; omitted if there is no further data. */
  nextCursor?: string;
  /** Total number of items matching the query (optional, may be omitted for performance). */
  total?: number;
}

/**
 * Generic paginated response.
 */
export interface PaginatedResponse<T> extends ApiResponse<{
  items: T[];
  pagination: PaginationMeta;
}> {}

/* -------------------------------------------------------------------------- */
/* Authentication                                                            */
/* -------------------------------------------------------------------------- */

/**
 * Payload sent to `/api/v1/auth/login` (or OIDC token exchange endpoint).
 */
export interface LoginRequest {
  /** User's email address. */
  email: string;
  /** Plain‑text password. */
  password: string;
}

/**
 * JWT token pair returned after a successful login.
 */
export interface TokenResponse {
  /** Access token (JWT) used for authorisation. */
  accessToken: string;
  /** Refresh token used to obtain a new access token. */
  refreshToken: string;
  /** Expiration time in seconds from issuance. */
  expiresIn: number;
  /** Token type, typically "Bearer". */
  tokenType: 'Bearer';
}

/* -------------------------------------------------------------------------- */
/* Document handling                                                         */
/* -------------------------------------------------------------------------- */

/**
 * Enumerates the possible lifecycle states of a document.
 */
export type DocumentStatus =
  | 'uploaded'   // file received, waiting for processing
  | 'processing' // OCR / LLM job in progress
  | 'completed' // extraction succeeded
  | 'failed'     // processing failed
  | 'approved'  // manually approved by an accountant
  | 'rejected'; // manually rejected or flagged

/**
 * Minimal metadata stored for each uploaded document.
 */
export interface DocumentMeta {
  /** UUID of the document record. */
  id: string;
  /** Identifier of the tenant that owns the document. */
  tenantId: string;
  /** Original filename supplied by the user. */
  filename: string;
  /** MIME type of the uploaded file. */
  contentType: string;
  /** Size in bytes. */
  size: number;
  /** ISO‑8601 timestamp when the upload completed. */
  uploadedAt: string;
  /** Current processing status. */
  status: DocumentStatus;
  /** Object storage key (e.g., `tenant/<id>/documents/<uuid>.pdf`). */
  storageKey: string;
  /** Optional extracted data; present when status === "completed". */
  extractedData?: ExtractedData;
}

/**
 * Structured data produced by the OCR + LLM pipeline.
 */
export interface ExtractedData {
  /** Resolved vendor name (e.g., "Acme Corp"). */
  vendor: string;
  /** Invoice or receipt number, if detected. */
  invoiceNumber?: string;
  /** Document date (ISO‑8601). */
  date?: string;
  /** Total monetary amount. */
  totalAmount?: number;
  /** ISO‑4217 currency code (e.g., "USD"). */
  currency?: string;
  /** Calculated tax amount, if applicable. */
  tax?: number;
  /** Raw OCR text before LLM parsing (useful for debugging). */
  rawText?: string;
  /** Detailed line items extracted from the document. */
  lineItems: LineItem[];
}

/**
 * Individual line‑item extracted from a financial document.
 */
export interface LineItem {
  /** Description of the product / service. */
  description: string;
  /** Quantity purchased. */
  quantity: number;
  /** Unit price (same currency as `totalAmount`). */
  unitPrice: number;
  /** Calculated line total (quantity * unitPrice). */
  amount: number;
  /** Optional categorisation (e.g., "Office Supplies"). */
  category?: string;
}

/**
 * Request body for generating a pre‑signed upload URL.
 * The backend validates the filename and size before returning a URL.
 */
export interface PresignUploadRequest {
  /** Original filename, including extension. */
  filename: string;
  /** MIME type of the file. */
  contentType: string;
  /** Size in bytes; used for quota checks. */
  size: number;
}

/**
 * Response containing a single‑use, time‑limited URL for direct upload to MinIO.
 * The client should perform a PUT request to `url` with the file bytes.
 */
export interface PresignUploadResponse {
  /** Fully‑qualified URL to which the file should be uploaded. */
  url: string;
  /** Expiration timestamp (ISO‑8601). */
  expiresAt: string;
}

/**
 * Payload returned after a document is successfully created (metadata only).
 */
export type DocumentCreateResponse = ApiResponse<DocumentMeta>;

/**
 * Detailed view of a single document, including audit timestamps.
 */
export interface DocumentDetail extends DocumentMeta {
  /** Timestamp of the last status change. */
  updatedAt: string;
  /** Optional notes added by accountants. */
  notes?: string;
}

/**
 * Response for fetching a single document.
 */
export type DocumentDetailResponse = ApiResponse<DocumentDetail>;

/**
 * Query parameters for listing documents with cursor‑based pagination.
 */
export interface ListDocumentsParams {
  /** Optional status filter (comma‑separated list). */
  status?: DocumentStatus | DocumentStatus[];
  /** Optional search term applied to filename or vendor. */
  q?: string;
  /** Cursor returned from the previous page. */
  cursor?: string;
  /** Number of items to return (max 100). */
  limit?: number;
}

/**
 * Response for `/api/v1/documents` list endpoint.
 */
export type ListDocumentsResponse = PaginatedResponse<DocumentMeta>;

/* -------------------------------------------------------------------------- */
/* Webhook management                                                         */
/* -------------------------------------------------------------------------- */

/**
 * Subscription definition for outbound webhook notifications.
 */
export interface WebhookSubscription {
  /** UUID of the subscription record. */
  id: string;
  /** URL to which events will be POSTed. */
  url: string;
  /** Secret used to compute HMAC signatures (stored hashed on the server). */
  secret: string;
  /** Array of event names the subscriber is interested in. */
  events: string[];
  /** Creation timestamp. */
  createdAt: string;
  /** Last update timestamp. */
  updatedAt: string;
}

/**
 * Request payload for creating a new webhook subscription.
 */
export interface CreateWebhookRequest {
  /** Destination URL for the webhook. */
  url: string;
  /** Optional secret; if omitted the server will generate a random one. */
  secret?: string;
  /** List of events to subscribe to (e.g., ["document.completed"]). */
  events: string[];
}

/**
 * Request payload for updating an existing webhook subscription.
 * All fields are optional; only provided fields are patched.
 */
export interface UpdateWebhookRequest {
  url?: string;
  secret?: string;
  events?: string[];
}

/**
 * Response wrapper for a single webhook subscription.
 */
export type WebhookSubscriptionResponse = ApiResponse<WebhookSubscription>;

/**
 * Paginated list response for webhook subscriptions.
 */
export type ListWebhooksResponse = PaginatedResponse<WebhookSubscription>;

/**
 * Payload sent by FinSight AI to a subscriber's endpoint.
 * The request body is JSON and includes an `X-Signature` header
 * containing a constant‑time HMAC of the raw body using the shared secret.
 */
export interface WebhookEventPayload {
  /** Name of the event (e.g., "document.completed"). */
  event: string;
  /** Identifier of the affected document. */
  documentId: string;
  /** Tenant identifier the document belongs to. */
  tenantId: string;
  /** Timestamp when the event was generated (ISO‑8601). */
  timestamp: string;
  /** Optional event‑specific data (e.g., extractedData). */
  data?: Record<string, unknown>;
}

/* -------------------------------------------------------------------------- */
/* Miscellaneous utilities                                                    */
/* -------------------------------------------------------------------------- */

/**
 * Helper type for values that may be null.
 */
export type Nullable<T> = T | null;

/**
 * Generic error shape returned by the API when `status` is "error".
 */
export interface ApiError {
  /** Short error code (e.g., "validation_error", "unauthorized"). */
  code: string;
  /** Human‑readable message describing the problem. */
  message: string;
  /** Optional field‑level validation details. */
  details?: Record<string, string[]>;
}

/**
 * Union of all possible API response shapes used throughout the frontend.
 * This is useful for generic error handling in Axios interceptors.
 */
export type AnyApiResponse =
  | ApiResponse
  | DocumentCreateResponse
  | DocumentDetailResponse
  | ListDocumentsResponse
  | WebhookSubscriptionResponse
  | ListWebhooksResponse
  | ApiResponse<TokenResponse>;

/* -------------------------------------------------------------------------- */
/* Export a consolidated namespace for convenient imports                     */
/* -------------------------------------------------------------------------- */

export namespace FinSightAPI {
  export type {
    ApiResponse,
    PaginatedResponse,
    PaginationMeta,
    LoginRequest,
    TokenResponse,
    DocumentStatus,
    DocumentMeta,
    DocumentDetail,
    ExtractedData,
    LineItem,
    PresignUploadRequest,
    PresignUploadResponse,
    DocumentCreateResponse,
    DocumentDetailResponse,
    ListDocumentsParams,
    ListDocumentsResponse,
    WebhookSubscription,
    CreateWebhookRequest,
    UpdateWebhookRequest,
    WebhookSubscriptionResponse,
    ListWebhooksResponse,
    WebhookEventPayload,
    ApiError,
    Nullable,
    AnyApiResponse,
  };
}