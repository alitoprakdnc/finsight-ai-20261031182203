import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import {
  useQuery,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import api from '../../utils/api';
import Layout from '../../components/Layout';
import LoadingSkeleton from '../../components/LoadingSkeleton';
import PermissionGate from '../../components/PermissionGate';
import ModalConfirm from '../../components/ModalConfirm';
import { Card } from '../../components/Card';
import { Table } from '../../components/Table';
import { Badge } from '../../components/Badge';
import { useToast } from '../../components/ToastProvider';
import { format } from 'date-fns';
import { ArrowPathIcon, PencilIcon } from '@heroicons/react/24/outline';

type DocumentStatus = 'uploaded' | 'processing' | 'completed' | 'failed' | 'approved' | 'rejected';

interface StatusHistoryEntry {
  status: DocumentStatus;
  timestamp: string; // ISO string
  user: string;
}

interface Document {
  id: string;
  filename: string;
  uploadedAt: string;
  status: DocumentStatus;
  previewUrl: string; // presigned URL
  extractedData: Record<string, any>;
  statusHistory: StatusHistoryEntry[];
}

/**
 * Fetch a single document by ID.
 */
const fetchDocument = async (id: string): Promise<Document> => {
  const { data } = await api.get(`/api/v1/documents/${id}`);
  return data;
};

/**
 * Update extracted data for a document.
 */
const updateDocument = async ({
  id,
  extractedData,
}: {
  id: string;
  extractedData: Record<string, any>;
}) => {
  const { data } = await api.patch(`/api/v1/documents/${id}`, {
    extractedData,
  });
  return data;
};

export default function DocumentDetailPage() {
  const router = useRouter();
  const { id } = router.query as { id: string };
  const toast = useToast();
  const queryClient = useQueryClient();

  const [isEditOpen, setEditOpen] = useState(false);
  const [editedJson, setEditedJson] = useState<string>('');

  // Load document
  const {
    data: document,
    isLoading,
    isError,
    error,
  } = useQuery<Document>(['document', id], () => fetchDocument(id), {
    enabled: !!id,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  // Mutation for saving edits
  const {
    mutateAsync: saveEdits,
    isLoading: isSaving,
  } = useMutation(updateDocument, {
    onSuccess: () => {
      toast.success('Document data updated successfully');
      setEditOpen(false);
      queryClient.invalidateQueries(['document', id]);
    },
    onError: (err: any) => {
      toast.error(
        err?.response?.data?.message ||
          'Failed to update document. Please try again.'
      );
    },
  });

  // Open edit modal with current JSON
  const openEditModal = () => {
    setEditedJson(JSON.stringify(document?.extractedData ?? {}, null, 2));
    setEditOpen(true);
  };

  // Handle save
  const handleSave = async () => {
    try {
      const parsed = JSON.parse(editedJson);
      await saveEdits({ id, extractedData: parsed });
    } catch (e) {
      toast.error('Invalid JSON. Please correct before saving.');
    }
  };

  // Re‑process document (push a new task)
  const {
    mutateAsync: reprocess,
    isLoading: isReprocessing,
  } = useMutation(
    async () => {
      await api.post(`/api/v1/documents/${id}/reprocess`);
    },
    {
      onSuccess: () => {
        toast.success('Re‑processing started');
        queryClient.invalidateQueries(['document', id]);
      },
      onError: (err: any) => {
        toast.error(
          err?.response?.data?.message ||
            'Failed to start re‑processing. Please try again.'
        );
      },
    }
  );

  // UI helpers
  const renderPreview = () => {
    if (!document?.previewUrl) return null;
    const lower = document.filename.toLowerCase();
    if (lower.endsWith('.pdf')) {
      return (
        <iframe
          src={document.previewUrl}
          className="w-full h-96 border rounded"
          title="Document preview"
        />
      );
    }
    // Assume image for other extensions
    return (
      <img
        src={document.previewUrl}
        alt={document.filename}
        className="max-w-full max-h-96 object-contain rounded border"
      />
    );
  };

  const renderStatusBadge = (status: DocumentStatus) => {
    const colors: Record<DocumentStatus, string> = {
      uploaded: 'bg-gray-200 text-gray-800',
      processing: 'bg-blue-100 text-blue-800',
      completed: 'bg-green-100 text-green-800',
      failed: 'bg-red-100 text-red-800',
      approved: 'bg-indigo-100 text-indigo-800',
      rejected: 'bg-yellow-100 text-yellow-800',
    };
    return (
      <span
        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[status]}`}
      >
        {status}
      </span>
    );
  };

  if (isLoading) {
    return (
      <Layout>
        <LoadingSkeleton className="h-96 w-full" />
      </Layout>
    );
  }

  if (isError || !document) {
    return (
      <Layout>
        <div className="p-8 text-center">
          <p className="text-red-600">
            {error?.response?.data?.message ||
              'Unable to load document. Please try again later.'}
          </p>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
          Document: {document.filename}
        </h1>
        <div className="flex items-center space-x-2">
          <PermissionGate roles={['Admin', 'Accountant']}>
            <button
              onClick={openEditModal}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
            >
              <PencilIcon className="h-5 w-5 mr-1" aria-hidden="true" />
              Edit Data
            </button>
          </PermissionGate>
          <button
            onClick={() => reprocess()}
            disabled={isReprocessing}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-primary-700 bg-primary-100 hover:bg-primary-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
          >
            <ArrowPathIcon
              className="h-5 w-5 mr-1"
              aria-hidden="true"
            />
            {isReprocessing ? 'Re‑processing...' : 'Re‑process'}
          </button>
        </div>
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Preview Card */}
        <Card className="p-4">
          <h2 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-3">
            Preview
          </h2>
          <div className="flex justify-center">{renderPreview()}</div>
        </Card>

        {/* Status & Timeline Card */}
        <Card className="p-4">
          <h2 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-3">
            Status
          </h2>
          <div className="flex items-center space-x-2 mb-4">
            {renderStatusBadge(document.status)}
            <span className="text-sm text-gray-500">
              Updated {format(new Date(document.uploadedAt), 'PPpp')}
            </span>
          </div>

          <h3 className="text-md font-medium text-gray-700 dark:text-gray-300 mb-2">
            History
          </h3>
          <ul className="space-y-2">
            {document.statusHistory
              .slice()
              .reverse()
              .map((h, idx) => (
                <li
                  key={idx}
                  className="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400"
                >
                  <div className="flex items-center space-x-2">
                    {renderStatusBadge(h.status)}
                    <span>{h.user}</span>
                  </div>
                  <span>{format(new Date(h.timestamp), 'PPpp')}</span>
                </li>
              ))}
          </ul>
        </Card>

        {/* Extracted Data Card */}
        <Card className="p-4 lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-gray-800 dark:text-gray-200">
              Extracted Data
            </h2>
            <PermissionGate roles={['Admin', 'Accountant']}>
              <button
                onClick={openEditModal}
                className="inline-flex items-center px-3 py-1.5 border border-transparent text-sm font-medium rounded-md shadow-sm text-primary-700 bg-primary-100 hover:bg-primary-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
              >
                <PencilIcon className="h-4 w-4 mr-1" aria-hidden="true" />
                Edit
              </button>
            </PermissionGate>
          </div>

          {Object.keys(document.extractedData).length === 0 ? (
            <p className="text-gray-500">No data extracted yet.</p>
          ) : (
            <Table>
              <thead>
                <tr className="bg-gray-50 dark:bg-gray-700">
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Field
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Value
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800">
                {Object.entries(document.extractedData).map(
                  ([key, value], idx) => (
                    <tr
                      key={idx}
                      className={idx % 2 === 0 ? '' : 'bg-gray-50 dark:bg-gray-700'}
                    >
                      <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                        {key}
                      </td>
                      <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300">
                        {typeof value === 'object'
                          ? JSON.stringify(value, null, 2)
                          : String(value)}
                      </td>
                    </tr>
                  )
                )}
              </tbody>
            </Table>
          )}
        </Card>
      </div>

      {/* Edit Modal */}
      <ModalConfirm
        isOpen={isEditOpen}
        title="Edit Extracted Data"
        confirmLabel="Save"
        cancelLabel="Cancel"
        onConfirm={handleSave}
        onCancel={() => setEditOpen(false)}
        isLoading={isSaving}
        size="lg"
      >
        <div className="mt-2">
          <label htmlFor="json-editor" className="sr-only">
            JSON editor
          </label>
          <textarea
            id="json-editor"
            rows={15}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
            value={editedJson}
            onChange={(e) => setEditedJson(e.target.value)}
          />
          <p className="mt-1 text-sm text-gray-500">
            Edit the JSON representation of the extracted data. Ensure it remains
            valid JSON.
          </p>
        </div>
      </ModalConfirm>
    </Layout>
  );
}