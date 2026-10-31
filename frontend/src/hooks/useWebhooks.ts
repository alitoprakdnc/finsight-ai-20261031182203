import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '../utils/api';
import {
  WebhookSubscription,
  CreateWebhookPayload,
  UpdateWebhookPayload,
} from '../types/webhook';

/**
 * Fetch all webhook subscriptions for the current tenant.
 */
const fetchWebhooks = async (): Promise<WebhookSubscription[]> => {
  const { data } = await api.get<{ data: WebhookSubscription[] }>('/api/v1/webhooks');
  // The API follows the standard response envelope { status, data, error }
  // Adjust the destructuring if your backend returns a different shape.
  return data;
};

/**
 * Hook exposing CRUD operations for webhook subscriptions.
 *
 * It uses TanStack Query for caching, background refetching and optimistic UI updates.
 * All mutations automatically invalidate the `webhooks` query to keep the list fresh.
 */
export const useWebhooks = () => {
  const queryClient = useQueryClient();

  // ── LIST ───────────────────────────────────────────────────────────────────────
  const {
    data: webhooks,
    isLoading,
    isError,
    error,
  } = useQuery<WebhookSubscription[], Error>(['webhooks'], fetchWebhooks, {
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 2,
  });

  // ── CREATE ─────────────────────────────────────────────────────────────────────
  const createMutation = useMutation<
    WebhookSubscription,
    Error,
    CreateWebhookPayload
  >(
    (payload) =>
      api
        .post<{ data: WebhookSubscription }>('/api/v1/webhooks', payload)
        .then((res) => res.data),
    {
      onSuccess: () => {
        // Refresh the list after a successful creation.
        queryClient.invalidateQueries(['webhooks']);
      },
    }
  );

  // ── UPDATE ─────────────────────────────────────────────────────────────────────
  const updateMutation = useMutation<
    WebhookSubscription,
    Error,
    { id: string } & UpdateWebhookPayload
  >(
    ({ id, ...payload }) =>
      api
        .put<{ data: WebhookSubscription }>(`/api/v1/webhooks/${id}`, payload)
        .then((res) => res.data),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['webhooks']);
      },
    }
  );

  // ── DELETE ─────────────────────────────────────────────────────────────────────
  const deleteMutation = useMutation<void, Error, string>(
    (id) => api.delete(`/api/v1/webhooks/${id}`).then(() => undefined),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['webhooks']);
      },
    }
  );

  return {
    // Data
    webhooks,
    isLoading,
    isError,
    error,

    // Mutations (expose async functions for callers)
    createWebhook: createMutation.mutateAsync,
    updateWebhook: updateMutation.mutateAsync,
    deleteWebhook: deleteMutation.mutateAsync,

    // Mutation status objects (useful for UI feedback)
    createStatus: createMutation,
    updateStatus: updateMutation,
    deleteStatus: deleteMutation,
  };
};

export default useWebhooks;