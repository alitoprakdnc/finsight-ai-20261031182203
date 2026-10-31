import React, { useState } from "react";
import { useRouter } from "next/router";
import { Document } from "@/src/types/document";
import { api } from "@/src/utils/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/src/components/ui/Card";
import { Badge } from "@/src/components/ui/Badge";
import { IconButton } from "@/src/components/ui/IconButton";
import { ModalConfirm } from "@/src/components/ModalConfirm";
import { useToast } from "@/src/components/ToastProvider";
import { PermissionGate } from "@/src/components/PermissionGate";
import {
  EyeIcon,
  DownloadIcon,
  TrashIcon,
  RefreshIcon,
} from "@heroicons/react/24/outline";

type DocumentCardProps = {
  /** Document data to render */
  document: Document;
  /** Optional callback after a successful delete (e.g., to close a parent modal) */
  onDeleted?: (id: string) => void;
};

/**
 * Mapping of document status to badge colour and label.
 * Colours follow the design system palette.
 */
const statusMap: Record<
  Document["status"],
  { label: string; color: string; bg: string }
> = {
  pending: { label: "Pending", color: "text-yellow-800", bg: "bg-yellow-100" },
  processing: {
    label: "Processing",
    color: "text-blue-800",
    bg: "bg-blue-100",
  },
  completed: {
    label: "Completed",
    color: "text-green-800",
    bg: "bg-green-100",
  },
  failed: { label: "Failed", color: "text-red-800", bg: "bg-red-100" },
};

export const DocumentCard: React.FC<DocumentCardProps> = ({
  document,
  onDeleted,
}) => {
  const router = useRouter();
  const toast = useToast();
  const queryClient = useQueryClient();

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [downloading, setDownloading] = useState(false);

  const deleteMutation = useMutation(
    async () => {
      await api.delete(`/api/v1/documents/${document.id}`);
    },
    {
      onSuccess: () => {
        toast.success("Document deleted");
        queryClient.invalidateQueries({ queryKey: ["documents"] });
        onDeleted?.(document.id);
      },
      onError: (err: any) => {
        console.error(err);
        toast.error("Failed to delete document");
      },
    }
  );

  const handleView = () => {
    router.push(`/documents/${document.id}`);
  };

  const handleDownload = async () => {
    setDownloading(true);
    try {
      // Expect the backend to return a pre‑signed URL
      const { data } = await api.get<{ url: string }>(
        `/api/v1/documents/${document.id}/download`
      );
      const link = document.createElement("a");
      link.href = data.url;
      link.download = document.filename ?? "document";
      document.body.appendChild(link);
      link.click();
      link.remove();
      toast.success("Download started");
    } catch (err: any) {
      console.error(err);
      toast.error("Unable to download document");
    } finally {
      setDownloading(false);
    }
  };

  const handleDelete = () => {
    setConfirmOpen(true);
  };

  const confirmDelete = async () => {
    setConfirmOpen(false);
    deleteMutation.mutate();
  };

  const statusInfo = statusMap[document.status] ?? {
    label: document.status,
    color: "text-gray-800",
    bg: "bg-gray-100",
  };

  return (
    <Card className="flex flex-col h-full">
      {/* Thumbnail */}
      <div className="relative aspect-[4/3] bg-gray-50 rounded-t-md overflow-hidden">
        {document.thumbnailUrl ? (
          <img
            src={document.thumbnailUrl}
            alt={`Preview of ${document.title}`}
            className="object-cover w-full h-full"
          />
        ) : (
          <div className="flex items-center justify-center h-full text-gray-400">
            <EyeIcon className="w-12 h-12" aria-hidden="true" />
          </div>
        )}
        {/* Status badge – positioned top‑right */}
        <Badge
          className={`absolute top-2 right-2 ${statusInfo.color} ${statusInfo.bg} text-xs font-medium`}
        >
          {statusInfo.label}
        </Badge>
      </div>

      {/* Content */}
      <div className="flex-1 p-4 flex flex-col justify-between">
        <h3 className="text-sm font-medium text-gray-900 truncate">
          {document.title}
        </h3>
        <p className="mt-1 text-xs text-gray-500">
          {new Date(document.createdAt).toLocaleDateString()}
        </p>

        {/* Action bar */}
        <div className="mt-3 flex items-center space-x-2">
          <IconButton
            aria-label="View document details"
            onClick={handleView}
            className="text-primary-600 hover:text-primary-800"
          >
            <EyeIcon className="w-5 h-5" />
          </IconButton>

          <IconButton
            aria-label="Download document"
            onClick={handleDownload}
            disabled={downloading}
            className="text-primary-600 hover:text-primary-800"
          >
            <DownloadIcon className="w-5 h-5" />
          </IconButton>

          {/* Delete only for users with the Accountant role */}
          <PermissionGate roles={["Admin", "Accountant"]}>
            <IconButton
              aria-label="Delete document"
              onClick={handleDelete}
              disabled={deleteMutation.isLoading}
              className="text-red-600 hover:text-red-800"
            >
              <TrashIcon className="w-5 h-5" />
            </IconButton>
          </PermissionGate>

          {/* Refresh status – visible to any role */}
          <IconButton
            aria-label="Refresh status"
            onClick={() => queryClient.invalidateQueries({ queryKey: ["document", document.id] })}
            className="text-gray-600 hover:text-gray-800"
          >
            <RefreshIcon className="w-5 h-5" />
          </IconButton>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <ModalConfirm
        open={confirmOpen}
        title="Delete document"
        description={`Are you sure you want to permanently delete “${document.title}”? This action cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={confirmDelete}
        onClose={() => setConfirmOpen(false)}
      />
    </Card>
  );
};

export default DocumentCard;