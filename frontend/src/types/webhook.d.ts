/**
 * Types and interfaces related to webhook subscriptions and event payloads.
 *
 * These definitions are used throughout the frontend codebase (hooks, API utils,
 * UI components) to provide strong typing and autocomplete support.
 *
 * The shapes mirror the backend models defined in `internal/models/webhook.go` and
 * the OpenAPI specification (`pkg/openapi/openapi.yaml`).
 */

export enum WebhookEventType {
  /** Emitted when a document is successfully uploaded to storage. */
  DocumentUploaded = 'document.uploaded',
  /** Emitted after OCR and LLM processing have completed. */
  DocumentProcessed = 'document.processed',
  /** Emitted when a document is approved by an accountant. */
  DocumentApproved = 'document.approved',
  /** Emitted when a document is rejected or sent back for correction. */
  DocumentRejected = 'document.rejected',
  /** Emitted when a webhook delivery fails permanently. */
  DeliveryFailed = 'webhook.delivery_failed',
}

/**
 * Core fields that are present on every webhook event payload.
 */
export interface BaseWebhookEvent {
  /** Unique identifier for the event (UUID v4). */
  eventId: string;
  /** Type of the event – must be one of {@link WebhookEventType}. */
  eventType: WebhookEventType;
  /** ISO‑8601 timestamp when the event was generated (UTC). */
  timestamp: string;
  /** Identifier of the tenant (organization) that owns the document. */
  tenantId: string;
}

/**
 * Payload for {@link WebhookEventType.DocumentUploaded}.
 */
export interface DocumentUploadedPayload extends BaseWebhookEvent {
  eventType: WebhookEventType.DocumentUploaded;
  /** Identifier of the uploaded document. */
  documentId: string;
  /** Original filename supplied by the user. */
  fileName: string;
  /** Size of the file in bytes. */
  size: number;
  /** MIME type of the uploaded file. */
  contentType: string;
  /** User identifier of the uploader. */
  uploadedBy: string;
}

/**
 * Payload for {@link WebhookEventType.DocumentProcessed}.
 */
export interface DocumentProcessedPayload extends BaseWebhookEvent {
  eventType: WebhookEventType.DocumentProcessed;
  /** Identifier of the processed document. */
  documentId: string;
  /** Result of OCR – raw extracted text. */
  rawText: string;
  /** Structured data produced by the LLM (vendor, line items, taxes, etc.). */
  extractedData: Record<string, unknown>;
  /** Processing status – typically "completed". */
  status: 'completed' | 'failed';
  /** Optional error message if processing failed. */
  errorMessage?: string;
}

/**
 * Payload for {@link WebhookEventType.DocumentApproved}.
 */
export interface DocumentApprovedPayload extends BaseWebhookEvent {
  eventType: WebhookEventType.DocumentApproved;
  /** Identifier of the approved document. */
  documentId: string;
  /** User identifier of the accountant who approved the document. */
  approvedBy: string;
  /** ISO‑8601 timestamp of the approval action. */
  approvedAt: string;
}

/**
 * Payload for {@link WebhookEventType.DocumentRejected}.
 */
export interface DocumentRejectedPayload extends BaseWebhookEvent {
  eventType: WebhookEventType.DocumentRejected;
  /** Identifier of the rejected document. */
  documentId: string;
  /** User identifier of the accountant who rejected the document. */
  rejectedBy: string;
  /** ISO‑8601 timestamp of the rejection action. */
  rejectedAt: string;
  /** Human‑readable reason for rejection. */
  reason: string;
}

/**
 * Payload for {@link WebhookEventType.DeliveryFailed}.
 *
 * This event is emitted by the backend when a webhook delivery has exceeded the
 * maximum retry count and is being marked as permanently failed.
 */
export interface DeliveryFailedPayload extends BaseWebhookEvent {
  eventType: WebhookEventType.DeliveryFailed;
  /** Identifier of the webhook subscription that failed. */
  subscriptionId: string;
  /** Last HTTP status code received from the endpoint (if any). */
  lastStatusCode?: number;
  /** Error message describing the failure. */
  errorMessage: string;
}

/**
 * Union type representing any possible webhook event payload.
 *
 * This type is useful when handling generic webhook callbacks on the client
 * (e.g., previewing recent events in the UI).
 */
export type WebhookEventPayload =
  | DocumentUploadedPayload
  | DocumentProcessedPayload
  | DocumentApprovedPayload
  | DocumentRejectedPayload
  | DeliveryFailedPayload;

/**
 * Representation of a webhook subscription stored in the backend.
 *
 * The `secret` field is **never** sent to the frontend; it is only used by the
 * backend to compute HMAC signatures. The frontend receives the rest of the
 * fields for display and management purposes.
 */
export interface WebhookSubscription {
  /** Unique identifier of the subscription (UUID v4). */
  id: string;
  /** Human‑readable name for the subscription (optional). */
  name?: string;
  /** URL that will receive POST requests for the selected events. */
  url: string;
  /** List of event types this subscription is interested in. */
  events: WebhookEventType[];
  /** Whether the subscription is currently active. */
  enabled: boolean;
  /** Timestamp of creation (ISO‑8601, UTC). */
  createdAt: string;
  /** Timestamp of the last update (ISO‑8601, UTC). */
  updatedAt: string;
  /** Number of consecutive delivery failures. */
  failureCount: number;
  /** Timestamp of the most recent delivery failure, if any. */
  lastFailedAt?: string;
}

/**
 * Payload used when creating a new webhook subscription via the API.
 *
 * The backend will generate `id`, timestamps, and failure counters.
 */
export interface CreateWebhookSubscriptionPayload {
  /** Optional friendly name for the subscription. */
  name?: string;
  /** Destination URL for webhook POSTs. Must be a valid HTTPS endpoint. */
  url: string;
  /** Events that should trigger the webhook. */
  events: WebhookEventType[];
  /** Whether the subscription should be enabled immediately. */
  enabled?: boolean;
}

/**
 * Payload used when updating an existing webhook subscription.
 *
 * All fields are optional – only the supplied ones will be patched.
 */
export interface UpdateWebhookSubscriptionPayload {
  /** Updated friendly name. */
  name?: string;
  /** Updated destination URL. */
  url?: string;
  /** Updated list of events. */
  events?: WebhookEventType[];
  /** Toggle the enabled flag. */
  enabled?: boolean;
}

/**
 * Standard API response envelope used by the backend for webhook‑related
 * endpoints.  It is exported here for convenience when typing `axios` responses.
 */
export interface ApiResponse<T> {
  /** HTTP status code (mirrors the actual response status). */
  status: number;
  /** Payload data – generic to allow any shape. */
  data: T;
  /** Optional error message when `status` indicates a failure. */
  error?: string;
}

/**
 * Helper type to extract the data shape from an {@link ApiResponse}.
 *
 * Example:
 * ```ts
 * type SubList = UnwrapApiResponse<ApiResponse<WebhookSubscription[]>>; // WebhookSubscription[]
 * ```
 */
export type UnwrapApiResponse<R> = R extends ApiResponse<infer D> ? D : never;