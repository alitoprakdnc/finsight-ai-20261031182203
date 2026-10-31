import { useState } from 'react';
import {
  useInfiniteQuery,
  useQueryClient,
  UseInfiniteQueryResult,
} from '@tanstack/react-query';
import api from '../utils/api';
import { Document } from '../types/document';

/**
 * Shape of the paginated response returned by the backend.
 * The API is expected to return an array of documents together with
 * optional cursor values for forward/backward navigation.
 */
interface DocumentListResponse {
  /** Documents for the current page */
  documents: Document[];
  /** Cursor to fetch the next page (if any) */
  nextCursor?: string | null;
  /** Cursor to fetch the previous page (if any) */
  prevCursor?: string | null;
  /** Total number of documents matching the query (optional) */
  total?: number;
}

/**
 * Hook for fetching, searching and paginating documents.
 *
 * It uses React‑Query's `useInfiniteQuery` to lazily load pages.
 * The hook abstracts pagination details and exposes a simple API
 * that can be consumed by UI components (tables, infinite scroll, etc.).
 *
 * @param pageSize Number of documents per request (default: 20)
 * @returns An object containing the flattened document list, loading states,
 *          pagination helpers and a controlled search term.
 */
export const useDocuments = (pageSize: number = 20) => {
  const [search, setSearch] = useState<string>('');
  const queryClient = useQueryClient();

  /**
   * Fetch a single page of documents.
   *
   * The backend expects:
   *   GET /api/v1/documents?limit={pageSize}&cursor={cursor}&search={search}
   *
   * `pageParam` is the cursor supplied by React‑Query.
   */
  const fetchDocuments = async ({
    pageParam = '',
  }: {
    pageParam?: string;
  }): Promise<DocumentListResponse> => {
    const params: Record<string, string | number> = {
      limit: pageSize,
    };
    if (pageParam) {
      params.cursor = pageParam;
    }
    if (search) {
      params.search = search;
    }

    const response = await api.get<DocumentListResponse>('/documents', {
      params,
    });
    return response.data;
  };

  const {
    data,
    error,
    fetchNextPage,
    fetchPreviousPage,
    hasNextPage,
    hasPreviousPage,
    isFetching,
    isFetchingNextPage,
    isLoading,
    isError,
    refetch,
  }: UseInfiniteQueryResult<DocumentListResponse, unknown> = useInfiniteQuery(
    ['documents', search],
    fetchDocuments,
    {
      /**
       * The cursor for the next page is taken from the `nextCursor` field.
       * If the field is missing or null we signal that there is no further page.
       */
      getNextPageParam: (lastPage) =>
        lastPage.nextCursor && lastPage.nextCursor !== '' ? lastPage.nextCursor : undefined,

      /**
       * The cursor for the previous page is taken from the `prevCursor` field.
       */
      getPreviousPageParam: (firstPage) =>
        firstPage.prevCursor && firstPage.prevCursor !== '' ? firstPage.prevCursor : undefined,

      // Keep data for a short period to avoid flicker when switching pages/search.
      staleTime: 60_000,
      keepPreviousData: true,
      retry: 2,
    }
  );

  // Flatten all loaded pages into a single array for UI consumption.
  const documents: Document[] = data
    ? data.pages.flatMap((page) => page.documents)
    : [];

  /**
   * Update the search term.
   *
   * Changing the term invalidates the query cache, which triggers a fresh
   * request starting from the first page.
   */
  const setSearchTerm = (term: string) => {
    setSearch(term);
    // Invalidate the query so that a new request with the updated term is made.
    queryClient.invalidateQueries(['documents']);
  };

  /** Load the next page if available. */
  const loadMore = () => {
    if (hasNextPage) {
      void fetchNextPage();
    }
  };

  /** Load the previous page if available. */
  const loadPrevious = () => {
    if (hasPreviousPage) {
      void fetchPreviousPage();
    }
  };

  return {
    /** Complete list of documents loaded so far (flattened). */
    documents,
    /** Current search term. */
    search,
    /** Setter for the search term. */
    setSearch: setSearchTerm,
    /** Loading state for the initial request. */
    isLoading,
    /** Loading state for any background fetch (next/prev page). */
    isFetching,
    /** Loading state for the next page request. */
    isFetchingNextPage,
    /** Error object if the request failed. */
    error,
    /** Boolean flag indicating a request error. */
    isError,
    /** Pagination helpers. */
    hasNextPage,
    hasPreviousPage,
    loadMore,
    loadPrevious,
    /** Manual refetch of the current query (useful after mutations). */
    refetch,
  };
};

export default useDocuments;