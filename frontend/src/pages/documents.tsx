import React, { useEffect, useState, ChangeEvent } from 'react';
import Layout from '@/components/Layout';
import DocumentCard from '@/components/DocumentCard';
import UploadDropzone from '@/components/UploadDropzone';
import PermissionGate from '@/components/PermissionGate';
import { useDocuments } from '@/hooks/useDocuments';
import { useToast } from '@/components/ToastProvider';
import { FiSearch, FiTrash2, FiCheckCircle } from 'react-icons/fi';
import { debounce } from 'lodash';
import { DocumentStatus } from '@/types/document';
import Head from 'next/head';

const statusOptions: { label: string; value: DocumentStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Pending', value: 'pending' },
  { label: 'Processing', value: 'processing' },
  { label: 'Processed', value: 'processed' },
  { label: 'Approved', value: 'approved' },
  { label: 'Rejected', value: 'rejected' },
];

const DocumentsPage: React.FC = () => {
  const {
    documents,
    total,
    page,
    pageSize,
    isLoading,
    isError,
    error,
    setPage,
    setPageSize,
    setSearch,
    setFilters,
    selectedIds,
    toggleSelect,
    selectAll,
    clearSelection,
    bulkDelete,
    bulkApprove,
    refetch,
  } = useDocuments();

  const toast = useToast();

  const [searchInput, setSearchInput] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [dateFrom, setDateFrom] = useState<string>('');
  const [dateTo, setDateTo] = useState<string>('');

  // Debounced search to avoid spamming API
  const debouncedSearch = React.useMemo(
    () =>
      debounce((value: string) => {
        setSearch(value);
        setPage(1);
      }, 400),
    [setSearch, setPage],
  );

  useEffect(() => {
    debouncedSearch(searchInput);
    return debouncedSearch.cancel;
  }, [searchInput, debouncedSearch]);

  // Apply filters whenever any filter changes
  useEffect(() => {
    const filters: Record<string, string> = {};
    if (statusFilter) filters.status = statusFilter;
    if (dateFrom) filters.dateFrom = dateFrom;
    if (dateTo) filters.dateTo = dateTo;
    setFilters(filters);
    setPage(1);
  }, [statusFilter, dateFrom, dateTo, setFilters, setPage]);

  // Error handling
  useEffect(() => {
    if (isError && error) {
      toast.error(`Failed to load documents: ${error.message}`);
    }
  }, [isError, error, toast]);

  const handleSelectAllChange = (e: ChangeEvent<HTMLInputElement>) => {
    if (e.target.checked) {
      selectAll();
    } else {
      clearSelection();
    }
  };

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return;
    try {
      await bulkDelete(Array.from(selectedIds));
      toast.success('Selected documents have been deleted.');
      clearSelection();
      refetch();
    } catch (err: any) {
      toast.error(`Delete failed: ${err.message}`);
    }
  };

  const handleBulkApprove = async () => {
    if (selectedIds.size === 0) return;
    try {
      await bulkApprove(Array.from(selectedIds));
      toast.success('Selected documents have been approved.');
      clearSelection();
      refetch();
    } catch (err: any) {
      toast.error(`Approve failed: ${err.message}`);
    }
  };

  const totalPages = Math.ceil(total / pageSize);
  const startIdx = (page - 1) * pageSize + 1;
  const endIdx = Math.min(page * pageSize, total);

  return (
    <>
      <Head>
        <title>Documents – FinSight AI</title>
      </Head>
      <Layout>
        <div className="flex flex-col gap-6 p-4 md:p-8">
          {/* Header */}
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
              Documents
            </h1>
            <UploadDropzone />
          </div>

          {/* Toolbar */}
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 bg-white dark:bg-gray-800 p-4 rounded-lg shadow-sm">
            {/* Search */}
            <div className="relative w-full md:w-64">
              <input
                type="text"
                placeholder="Search by filename, vendor..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                aria-label="Search documents"
              />
              <FiSearch className="absolute left-3 top-2.5 text-gray-400 dark:text-gray-500" />
            </div>

            {/* Filters */}
            <div className="flex flex-wrap items-center gap-3">
              {/* Status */}
              <select
                className="border border-gray-300 dark:border-gray-600 rounded-md py-2 px-3 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                aria-label="Filter by status"
              >
                {statusOptions.map((opt) => (
                  <option key={opt.label} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>

              {/* Date range */}
              <input
                type="date"
                className="border border-gray-300 dark:border-gray-600 rounded-md py-2 px-3 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                aria-label="Filter from date"
              />
              <span className="text-gray-500 dark:text-gray-400">–</span>
              <input
                type="date"
                className="border border-gray-300 dark:border-gray-600 rounded-md py-2 px-3 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                aria-label="Filter to date"
              />
            </div>

            {/* Bulk actions */}
            <PermissionGate roles={['admin', 'accountant']}>
              <div className="flex items-center gap-2">
                <label className="inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="form-checkbox h-4 w-4 text-primary-600 border-gray-300 rounded"
                    checked={selectedIds.size === documents.length && documents.length > 0}
                    onChange={handleSelectAllChange}
                    aria-label="Select all documents"
                  />
                  <span className="ml-2 text-sm text-gray-700 dark:text-gray-200">
                    Select All
                  </span>
                </label>
                <button
                  type="button"
                  onClick={handleBulkApprove}
                  disabled={selectedIds.size === 0}
                  className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium ${
                    selectedIds.size === 0
                      ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                      : 'bg-primary-600 text-white hover:bg-primary-700'
                  } focus:outline-none focus:ring-2 focus:ring-primary-500`}
                >
                  <FiCheckCircle />
                  Approve
                </button>
                <button
                  type="button"
                  onClick={handleBulkDelete}
                  disabled={selectedIds.size === 0}
                  className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium ${
                    selectedIds.size === 0
                      ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                      : 'bg-red-600 text-white hover:bg-red-700'
                  } focus:outline-none focus:ring-2 focus:ring-red-500`}
                >
                  <FiTrash2 />
                  Delete
                </button>
              </div>
            </PermissionGate>
          </div>

          {/* Document list */}
          {isLoading ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {Array.from({ length: pageSize }).map((_, idx) => (
                <div key={idx} className="p-4 bg-white dark:bg-gray-800 rounded-lg shadow animate-pulse">
                  <div className="h-40 bg-gray-200 dark:bg-gray-700 rounded mb-4" />
                  <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4 mb-2" />
                  <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2" />
                </div>
              ))}
            </div>
          ) : documents.length === 0 ? (
            <div className="text-center py-12 text-gray-500 dark:text-gray-400">
              No documents found. Try adjusting your filters or upload new documents.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {documents.map((doc) => (
                <DocumentCard
                  key={doc.id}
                  document={doc}
                  selected={selectedIds.has(doc.id)}
                  onSelect={() => toggleSelect(doc.id)}
                />
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <nav
              className="flex items-center justify-between py-4"
              aria-label="Pagination"
            >
              <div className="text-sm text-gray-700 dark:text-gray-300">
                Showing <span className="font-medium">{startIdx}</span> to{' '}
                <span className="font-medium">{endIdx}</span> of{' '}
                <span className="font-medium">{total}</span> documents
              </div>
              <div className="flex space-x-2">
                <button
                  onClick={() => setPage(page - 1)}
                  disabled={page === 1}
                  className={`px-3 py-1 rounded-md text-sm font-medium focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                    page === 1
                      ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                      : 'bg-primary-600 text-white hover:bg-primary-700'
                  }`}
                >
                  Previous
                </button>
                {Array.from({ length: totalPages }).map((_, i) => {
                  const pageNum = i + 1;
                  return (
                    <button
                      key={pageNum}
                      onClick={() => setPage(pageNum)}
                      aria-current={pageNum === page ? 'page' : undefined}
                      className={`px-3 py-1 rounded-md text-sm font-medium focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                        pageNum === page
                          ? 'bg-primary-700 text-white'
                          : 'bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600'
                      }`}
                    >
                      {pageNum}
                    </button>
                  );
                })}
                <button
                  onClick={() => setPage(page + 1)}
                  disabled={page === totalPages}
                  className={`px-3 py-1 rounded-md text-sm font-medium focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                    page === totalPages
                      ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                      : 'bg-primary-600 text-white hover:bg-primary-700'
                  }`}
                >
                  Next
                </button>
              </div>
            </nav>
          )}
        </div>
      </Layout>
    </>
  );
};

export default DocumentsPage;