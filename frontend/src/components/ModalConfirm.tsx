import React, { useState, useEffect, Fragment } from "react";
import { Dialog, Transition } from "@headlessui/react";

export interface ModalConfirmProps {
  /** Controls the visibility of the modal */
  isOpen: boolean;
  /** Title displayed at the top of the modal */
  title: string;
  /** Optional descriptive text shown under the title */
  description?: string;
  /** Text for the confirm button (defaults to "Confirm") */
  confirmText?: string;
  /** Text for the cancel button (defaults to "Cancel") */
  cancelText?: string;
  /** Callback executed when the user confirms. May be async. */
  onConfirm: () => Promise<void> | void;
  /** Callback executed when the modal is dismissed or cancel is clicked */
  onCancel: () => void;
  /** Optional external loading flag (e.g., when parent controls async state) */
  loading?: boolean;
}

/**
 * Reusable, accessible confirmation modal.
 *
 * Design System:
 * - Primary color: #0A74DA
 * - Secondary/neutral: gray-600 / gray-300
 * - Rounded corners, focus rings, hover states
 * - Dark mode support via Tailwind's `dark:` utilities
 *
 * Usage:
 * ```tsx
 * const [open, setOpen] = useState(false);
 *
 * <ModalConfirm
 *   isOpen={open}
 *   title="Delete document"
 *   description="This action cannot be undone."
 *   confirmText="Delete"
 *   cancelText="Keep"
 *   onConfirm={handleDelete}
 *   onCancel={() => setOpen(false)}
 * />
 * ```
 */
export const ModalConfirm: React.FC<ModalConfirmProps> = ({
  isOpen,
  title,
  description,
  confirmText = "Confirm",
  cancelText = "Cancel",
  onConfirm,
  onCancel,
  loading,
}) => {
  const [internalLoading, setInternalLoading] = useState(false);

  // Reset internal loading when modal is closed
  useEffect(() => {
    if (!isOpen) {
      setInternalLoading(false);
    }
  }, [isOpen]);

  const isSubmitting = loading ?? internalLoading;

  const handleConfirm = async () => {
    if (isSubmitting) return;
    try {
      setInternalLoading(true);
      await onConfirm();
    } finally {
      setInternalLoading(false);
    }
  };

  const handleClose = () => {
    if (isSubmitting) return; // Prevent closing while submitting
    onCancel();
  };

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={handleClose}>
        {/* Background overlay */}
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-200"
          enterFrom="opacity-0"
          enterTo="opacity-60"
          leave="ease-in duration-150"
          leaveFrom="opacity-60"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black dark:bg-gray-900 bg-opacity-60" />
        </Transition.Child>

        {/* Modal panel */}
        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-200"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-150"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel
                className="w-full max-w-md transform rounded-lg bg-white dark:bg-gray-800 p-6 text-left align-middle shadow-xl transition-all"
                data-testid="modal-confirm"
              >
                <Dialog.Title
                  as="h3"
                  className="text-lg font-medium leading-6 text-gray-900 dark:text-gray-100"
                >
                  {title}
                </Dialog.Title>
                {description && (
                  <div className="mt-2">
                    <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>
                  </div>
                )}

                <div className="mt-6 flex justify-end space-x-3">
                  {/* Cancel button */}
                  <button
                    type="button"
                    className="inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary-500 disabled:opacity-50"
                    onClick={handleClose}
                    disabled={isSubmitting}
                    data-testid="modal-cancel"
                  >
                    {cancelText}
                  </button>

                  {/* Confirm button */}
                  <button
                    type="button"
                    className="inline-flex items-center rounded-md border border-transparent bg-primary-600 dark:bg-primary-500 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 dark:hover:bg-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed"
                    onClick={handleConfirm}
                    disabled={isSubmitting}
                    data-testid="modal-confirm-button"
                  >
                    {isSubmitting ? (
                      <svg
                        className="animate-spin h-5 w-5 mr-2 text-white"
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
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
                          d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
                        />
                      </svg>
                    ) : null}
                    {confirmText}
                  </button>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
};

export default ModalConfirm;