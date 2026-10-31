import React, { useState, useEffect, FormEvent } from "react";
import Layout from "@/components/Layout";
import PermissionGate from "@/components/PermissionGate";
import ModalConfirm from "@/components/ModalConfirm";
import { useAuth } from "@/hooks/useAuth";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/utils/api";
import { toast } from "react-hot-toast";
import { ClipboardIcon, CheckIcon, XIcon, PlusIcon, TrashIcon, RefreshIcon } from "@heroicons/react/outline";

type Webhook = {
  id: string;
  url: string;
  secret: string;
  events: string[];
  enabled: boolean;
};

type ApiKeyResponse = {
  apiKey: string;
};

const SettingsPage: React.FC = () => {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  // ---------- Password Change ----------
  const [currentPwd, setCurrentPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [pwdLoading, setPwdLoading] = useState(false);

  const handlePasswordSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (newPwd !== confirmPwd) {
      toast.error("New password and confirmation do not match");
      return;
    }
    setPwdLoading(true);
    try {
      await api.post("/api/v1/users/me/password", {
        currentPassword: currentPwd,
        newPassword: newPwd,
      });
      toast.success("Password updated successfully");
      setCurrentPwd("");
      setNewPwd("");
      setConfirmPwd("");
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Failed to change password");
    } finally {
      setPwdLoading(false);
    }
  };

  // ---------- API Key ----------
  const {
    data: apiKeyData,
    isLoading: apiKeyLoading,
    refetch: refetchApiKey,
  } = useQuery<ApiKeyResponse>(["apiKey"], async () => {
    const res = await api.get("/api/v1/tenants/me/api-key");
    return res.data;
  });

  const [showKey, setShowKey] = useState(false);
  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  };

  // ---------- Webhook Management ----------
  const {
    data: webhooks,
    isLoading: webhooksLoading,
    refetch: refetchWebhooks,
  } = useQuery<Webhook[]>(["webhooks"], async () => {
    const res = await api.get("/api/v1/webhooks");
    return res.data;
  });

  const createWebhookMutation = useMutation(
    (payload: Omit<Webhook, "id">) => api.post("/api/v1/webhooks", payload),
    {
      onSuccess: () => {
        toast.success("Webhook created");
        queryClient.invalidateQueries(["webhooks"]);
        setShowWebhookModal(false);
      },
      onError: (err: any) => {
        toast.error(err?.response?.data?.error || "Failed to create webhook");
      },
    }
  );

  const deleteWebhookMutation = useMutation(
    (id: string) => api.delete(`/api/v1/webhooks/${id}`),
    {
      onSuccess: () => {
        toast.success("Webhook deleted");
        queryClient.invalidateQueries(["webhooks"]);
        setDeleteTarget(null);
      },
      onError: (err: any) => {
        toast.error(err?.response?.data?.error || "Failed to delete webhook");
      },
    }
  );

  // Modal state for add/edit webhook
  const [showWebhookModal, setShowWebhookModal] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<Webhook | null>(null);
  const [url, setUrl] = useState("");
  const [secret, setSecret] = useState("");
  const [events, setEvents] = useState<string>(""); // comma‑separated
  const [enabled, setEnabled] = useState(true);
  const [webhookSaving, setWebhookSaving] = useState(false);

  const openAddModal = () => {
    setEditingWebhook(null);
    setUrl("");
    setSecret("");
    setEvents("");
    setEnabled(true);
    setShowWebhookModal(true);
  };

  const openEditModal = (wh: Webhook) => {
    setEditingWebhook(wh);
    setUrl(wh.url);
    setSecret(wh.secret);
    setEvents(wh.events.join(", "));
    setEnabled(wh.enabled);
    setShowWebhookModal(true);
  };

  const handleWebhookSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setWebhookSaving(true);
    const payload = {
      url,
      secret,
      events: events.split(",").map((e) => e.trim()).filter(Boolean),
      enabled,
    };
    try {
      if (editingWebhook) {
        await api.patch(`/api/v1/webhooks/${editingWebhook.id}`, payload);
        toast.success("Webhook updated");
        queryClient.invalidateQueries(["webhooks"]);
        setShowWebhookModal(false);
      } else {
        await createWebhookMutation.mutateAsync(payload as Omit<Webhook, "id">);
      }
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Failed to save webhook");
    } finally {
      setWebhookSaving(false);
    }
  };

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<Webhook | null>(null);
  const confirmDelete = (wh: Webhook) => setDeleteTarget(wh);
  const handleDelete = async () => {
    if (deleteTarget) {
      await deleteWebhookMutation.mutateAsync(deleteTarget.id);
    }
  };

  // ---------- Render ----------
  return (
    <Layout title="Settings">
      <div className="max-w-5xl mx-auto p-4 space-y-8">
        {/* Password Change */}
        <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">
            Change Password
          </h2>
          <form onSubmit={handlePasswordSubmit} className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="currentPwd" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Current Password
              </label>
              <input
                id="currentPwd"
                type="password"
                required
                value={currentPwd}
                onChange={(e) => setCurrentPwd(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:text-gray-100"
              />
            </div>
            <div>
              <label htmlFor="newPwd" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                New Password
              </label>
              <input
                id="newPwd"
                type="password"
                required
                value={newPwd}
                onChange={(e) => setNewPwd(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:text-gray-100"
              />
            </div>
            <div>
              <label htmlFor="confirmPwd" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Confirm New Password
              </label>
              <input
                id="confirmPwd"
                type="password"
                required
                value={confirmPwd}
                onChange={(e) => setConfirmPwd(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-primary-500 focus:ring-primary-5
                00 dark:bg-gray-700 dark:text-gray-100"
              />
            </div>
            <div className="flex items-end">
              <button
                type="submit"
                disabled={pwdLoading}
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50"
              >
                {pwdLoading ? (
                  <svg className="animate-spin h-5 w-5 mr-2 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                ) : null}
                Change Password
              </button>
            </div>
          </form>
        </section>

        {/* API Key */}
        <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">
            API Key
          </h2>
          <div className="flex items-center space-x-4">
            {apiKeyLoading ? (
              <div className="animate-pulse h-6 w-48 bg-gray-200 dark:bg-gray-700 rounded"></div>
            ) : (
              <>
                <span className="font-mono text-sm text-gray-800 dark:text-gray-200 break-all">
                  {showKey ? apiKeyData?.apiKey : "••••••••••••••••••••••••••••••••"}
                </span>
                <button
                  onClick={() => setShowKey((prev) => !prev)}
                  className="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  aria-label={showKey ? "Hide API key" : "Show API key"}
                >
                  {showKey ? <EyeSlashIcon className="h-5 w-5 text-gray-600 dark:text-gray-300" /> : <EyeIcon className="h-5 w-5 text-gray-600 dark:text-gray-300" />}
                </button>
                <button
                  onClick={() => apiKeyData?.apiKey && copyToClipboard(apiKeyData.apiKey)}
                  className="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  aria-label="Copy API key"
                >
                  <ClipboardIcon className="h-5 w-5 text-gray-600 dark:text-gray-300" />
                </button>
                <button
                  onClick={() => refetchApiKey()}
                  className="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  aria-label="Refresh API key"
                >
                  <RefreshIcon className="h-5 w-5 text-gray-600 dark:text-gray-300" />
                </button>
              </>
            )}
          </div>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Keep this key secret. It grants full API access for your tenant.
          </p>
        </section>

        {/* Webhook Management */}
        <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
              Webhook Subscriptions
            </h2>
            <PermissionGate roles={["Admin"]}>
              <button
                onClick={openAddModal}
                className="inline-flex items-center px-3 py-1.5 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
              >
                <PlusIcon className="h-4 w-4 mr-1" />
                Add Webhook
              </button>
            </PermissionGate>
          </div>

          {webhooksLoading ? (
            <div className="animate-pulse h-48 bg-gray-200 dark:bg-gray-700 rounded"></div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-700">
                  <tr>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-200 uppercase tracking-wider">
                      URL
                    </th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-200 uppercase tracking-wider">
                      Events
                    </th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-200 uppercase tracking-wider">
                      Enabled
                    </th>
                    <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-200 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                  {webhooks?.map((wh) => (
                    <tr key={wh.id}>
                      <td className="px-4 py-2 text-sm text-gray-900 dark:text-gray-100 break-all">{wh.url}</td>
                      <td className="px-4 py-2 text-sm text-gray-700 dark:text-gray-300">{wh.events.join(", ")}</td>
                      <td className="px-4 py-2 text-sm">
                        {wh.enabled ? (
                          <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                            Enabled
                          </span>
                        ) : (
                          <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">
                            Disabled
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-2 text-right space-x-2">
                        <PermissionGate roles={["Admin"]}>
                          <button
                            onClick={() => openEditModal(wh)}
                            className="text-primary-600 hover:text-primary-900 focus:outline-none"
                          >
                            <svg className="h-5 w-5 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536M9 11l6-6 3.536 3.536a2.121 2.121 0 010 3L12 21H6v-6l9-9z" />
                            </svg>
                          </button>
                          <button
                            onClick={() => confirmDelete(wh)}
                            className="text-red-600 hover:text-red-900 focus:outline-none"
                          >
                            <TrashIcon className="h-5 w-5 inline" />
                          </button>
                        </PermissionGate>
                      </td>
                    </tr>
                  ))}
                  {webhooks?.length === 0 && (
                    <tr>
                      <td colSpan={4} className="px-4 py-4 text-center text-gray-500 dark:text-gray-400">
                        No webhook subscriptions configured.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* Delete Confirmation Modal */}
        {deleteTarget && (
          <ModalConfirm
            title="Delete Webhook"
            description={`Are you sure you want to delete the webhook for "${deleteTarget.url}"? This action cannot be undone.`}
            confirmLabel="Delete"
            cancelLabel="Cancel"
            onConfirm={handleDelete}
            onCancel={() => setDeleteTarget(null)}
            loading={deleteWebhookMutation.isLoading}
          />
        )}

        {/* Add / Edit Webhook Modal */}
        {showWebhookModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-lg mx-4 p-6 shadow-xl">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
                {editingWebhook ? "Edit Webhook" : "Add New Webhook"}
              </h3>
              <form onSubmit={handleWebhookSubmit} className="space-y-4">
                <div>
                  <label htmlFor="wh-url" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                    Destination URL
                  </label>
                  <input
                    id="wh-url"
                    type="url"
                    required
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:text-gray-100"
                  />
                </div>
                <div>
                  <label htmlFor="wh-secret" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                    Secret (HMAC)
                  </label>
                  <input
                    id="wh-secret"
                    type="text"
                    required
                    value={secret}
                    onChange={(e) => setSecret(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:text-gray-100"
                  />
                </div>
                <div>
                  <label htmlFor="wh-events" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                    Events (comma separated)
                  </label>
                  <input
                    id="wh-events"
                    type="text"
                    required
                    placeholder="document.created, document.approved"
                    value={events}
                    onChange={(e) => setEvents(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:text-gray-100"
                  />
                </div>
                <div className="flex items-center">
                  <input
                    id="wh-enabled"
                    type="checkbox"
                    checked={enabled}
                    onChange={(e) => setEnabled(e.target.checked)}
                    className="h-4 w-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
                  />
                  <label htmlFor="wh-enabled" className="ml-2 block text-sm text-gray-700 dark:text-gray-200">
                    Enabled
                  </label>
                </div>
                <div className="flex justify-end space-x-2 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowWebhookModal(false)}
                    className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={webhookSaving}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50"
                  >
                    {webhookSaving && (
                      <svg className="animate-spin -ml-1 mr-2 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                      </svg>
                    )}
                    {editingWebhook ? "Update" : "Create"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
};

export default SettingsPage;

// Heroicons (fallback if not globally available)
function EyeIcon(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} viewBox="0 0 20 20" fill="currentColor">
      <path d="M10 3c-5.33 0-9.5 5-9.5 5s4.17 5 9.5 5 9.5-5 9.5-5-4.17-5-9.5-5zm0 8a3 3 0 110-6 3 3 0 010 6z" />
    </svg>
  );
}
function EyeSlashIcon(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} viewBox="0 0 20 20" fill="currentColor">
      <path d="M13.875 12.825A5.5 5.5 0 0110 13.5c-3.038 0-5.5-2.462-5.5-5.5 0-.93.236-1.803.647-2.564l-1.42-1.42C2.79 5.86 2 7.33 2 9c0 5 8 9 8 9s1.58-.79 3.284-2.227l-1.409-1.409zM10 5c1.657 0 3 1.343 3 3 0 .28-.04.55-.11.81l1.45 1.45c.2-.5.31-1.04.31-1.61 0-3.038-2.462-5.5-5.5-5.5-1.28 0-2.46.44-3.41 1.18l1.45 1.45C9.45 5.04 9.72 5 10 5zM4.293 4.293a1 1 0 011.414 0L16.707 15.293a1 1 0 01-1.414 1.414L4.293 5.707a1 1 0 010-1.414z" />
    </svg>
  );
}