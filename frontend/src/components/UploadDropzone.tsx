import React, { useState, useCallback, useRef, ChangeEvent, DragEvent } from 'react';
import api from '@/utils/api';
import { useToast } from '@/components/ToastProvider';
import { Document } from '@/types/api';

type UploadDropzoneProps = {
  /** Called when a document has been successfully uploaded */
  onUploadSuccess?: (doc: Document) => void;
};

/**
 * UploadDropzone
 *
 * A drag‑and‑drop file uploader that:
 *   1. Requests a pre‑signed PUT URL from the backend.
 *   2. Uploads the file directly to MinIO using that URL.
 *   3. Emits success / error toasts.
 *   4. Returns the created document via `onUploadSuccess`.
 *
 * The component follows the project's design system (Tailwind, dark mode,
 * accessible ARIA attributes) and uses the shared `api` Axios instance
 * (which already injects JWT auth headers).
 */
const UploadDropzone: React.FC<UploadDropzoneProps> = ({ onUploadSuccess }) => {
  const [dragActive, setDragActive] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [currentFileName, setCurrentFileName] = useState<string>('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  // -------------------------------------------------------------------------
  // Helpers
  // -------------------------------------------------------------------------
  const resetState = () => {
    setUploading(false);
    setProgress(0);
    setCurrentFileName('');
  };

  const requestPresignedUrl = async (file: File) => {
    // The backend expects a JSON payload with filename & content_type.
    // It returns `{ url: string, documentId: string }`.
    const response = await api.post<{
      url: string;
      documentId: string;
    }>('/api/v1/documents/presign', {
      filename: file.name,
      content_type: file.type || 'application/octet-stream',
    });
    return response.data;
  };

  const uploadToPresignedUrl = async (
    url: string,
    file: File,
    onProgress: (percent: number) => void,
  ) => {
    await api.put(url, file, {
      headers: {
        'Content-Type': file.type || 'application/octet-stream',
      },
      // `onUploadProgress` works with Axios in the browser.
      onUploadProgress: (event) => {
        if (event.total) {
          const percent = Math.round((event.loaded * 100) / event.total);
          onProgress(percent);
        }
      },
    });
  };

  // -------------------------------------------------------------------------
  // Main upload flow
  // -------------------------------------------------------------------------
  const handleFile = useCallback(
    async (file: File) => {
      try {
        resetState();
        setUploading(true);
        setCurrentFileName(file.name);
        setProgress(0);

        // 1️⃣ Get a pre‑signed URL from the API.
        const { url, documentId } = await requestPresignedUrl(file);

        // 2️⃣ Upload the file directly to MinIO.
        await uploadToPresignedUrl(url, file, setProgress);

        // 3️⃣ Notify the UI.
        toast.success('File uploaded successfully');
        setUploading(false);
        setProgress(100);

        // 4️⃣ Propagate the created document (minimal fields).
        if (onUploadSuccess) {
          const doc: Document = {
            id: documentId,
            name: file.name,
            mime_type: file.type,
            size: file.size,
            status: 'uploaded',
            created_at: new Date().toISOString(),
            // The backend will fill the rest (extracted data, etc.).
          };
          onUploadSuccess(doc);
        }
      } catch (err: any) {
        console.error('Upload error:', err);
        toast.error(err?.response?.data?.message || 'File upload failed');
        resetState();
      }
    },
    [toast, onUploadSuccess],
  );

  // -------------------------------------------------------------------------
  // Event handlers
  // -------------------------------------------------------------------------
  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files[0]) {
      handleFile(files[0]);
    }
  };

  const handleDragEnter = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(true);
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    // Only reset when leaving the dropzone entirely.
    if (e.currentTarget.contains(e.relatedTarget as Node) === false) {
      setDragActive(false);
    }
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    const files = e.dataTransfer.files;
    if (files && files[0]) {
      handleFile(files[0]);
    }
  };

  const openFileDialog = () => {
    fileInputRef.current?.click();
  };

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------
  return (
    <div
      className={`
        relative flex flex-col items-center justify-center
        border-2 border-dashed rounded-lg p-6 cursor-pointer
        transition-colors
        ${dragActive ? 'border-primary bg-primary/5' : 'border-gray-300 bg-white'}
        dark:${dragActive ? 'border-primary-400 bg-primary-900/30' : 'border-gray-600 bg-gray-800'}
        hover:border-primary
        focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary
      `}
      role="button"
      tabIndex={0}
      aria-label="Upload financial document"
      onClick={openFileDialog}
      onKeyPress={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          openFileDialog();
        }
      }}
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      <input
        type="file"
        ref={fileInputRef}
        className="hidden"
        accept=".pdf,image/*"
        onChange={handleChange}
        disabled={uploading}
      />

      {uploading ? (
        <div className="w-full text-center">
          <p className="text-sm font-medium mb-2">
            Uploading <span className="font-semibold">{currentFileName}</span>
          </p>
          <div className="w-full bg-gray-200 rounded h-2 dark:bg-gray-700">
            <div
              className="h-2 bg-primary rounded transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
          <p className="text-xs text-gray-600 mt-1 dark:text-gray-400">{progress}%</p>
        </div>
      ) : (
        <div className="flex flex-col items-center text-center">
          {/* Heroicon: Cloud Upload */}
          <svg
            className="w-12 h-12 text-gray-400 dark:text-gray-500"
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
              d="M3 15a4 4 0 014-4h1.5a2.5 2.5 0 014.5 0H17a4 4 0 110 8H5a4 4 0 01-2-7.5"
            />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 12v6m3-3H9" />
          </svg>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">
            {dragActive ? 'Drop the file here' : 'Drag & drop or click to upload'}
          </p>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            PDF, JPG, PNG – max 10 MB
          </p>
        </div>
      )}
    </div>
  );
};

export default UploadDropzone;