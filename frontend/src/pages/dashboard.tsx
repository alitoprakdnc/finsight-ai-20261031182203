import React, { useEffect, useState } from "react";
import Head from "next/head";
import Layout from "@/components/Layout";
import ChartSpending from "@/components/ChartSpending";
import DocumentCard from "@/components/DocumentCard";
import LoadingSkeleton from "@/components/LoadingSkeleton";
import { useDocuments } from "@/hooks/useDocuments";
import { Document } from "@/types/document";
import { toast } from "@/components/ToastProvider";

const Dashboard: React.FC = () => {
  // Recent documents (most recent 5)
  const {
    data: recentDocs,
    isLoading: recentLoading,
    isError: recentError,
    error: recentErr,
    refetch: refetchRecent,
  } = useDocuments({ limit: 5, sort: "created_at_desc" });

  // Pending approvals (status = "pending")
  const {
    data: pendingDocs,
    isLoading: pendingLoading,
    isError: pendingError,
    error: pendingErr,
    refetch: refetchPending,
  } = useDocuments({ status: "pending", limit: 5, sort: "created_at_desc" });

  // Refresh data every minute to keep UI up‑to‑date
  useEffect(() => {
    const interval = setInterval(() => {
      refetchRecent();
      refetchPending();
    }, 60_000);
    return () => clearInterval(interval);
  }, [refetchRecent, refetchPending]);

  // Show toast on error
  useEffect(() => {
    if (recentError) {
      toast.error(
        recentErr?.message ?? "Failed to load recent documents."
      );
    }
  }, [recentError, recentErr]);

  useEffect(() => {
    if (pendingError) {
      toast.error(
        pendingErr?.message ?? "Failed to load pending approvals."
      );
    }
  }, [pendingError, pendingErr]);

  const renderDocumentList = (
    docs: Document[] | undefined,
    loading: boolean,
    emptyMessage: string
  ) => {
    if (loading) {
      return (
        <div className="space-y-4">
          {[...Array(3)].map((_, i) => (
            <LoadingSkeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      );
    }

    if (!docs || docs.length === 0) {
      return (
        <p className="text-gray-500 dark:text-gray-400 italic">{emptyMessage}</p>
      );
    }

    return (
      <ul className="space-y-4">
        {docs.map((doc) => (
          <li key={doc.id}>
            <DocumentCard document={doc} />
          </li>
        ))}
      </ul>
    );
  };

  return (
    <>
      <Head>
        <title>Dashboard – FinSight AI</title>
      </Head>
      <Layout>
        <main className="container mx-auto px-4 py-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
            Dashboard
          </h1>

          {/* Top row – Spending chart */}
          <section className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4 mb-8">
            <h2 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-4">
              Spending Overview
            </h2>
            <ChartSpending />
          </section>

          {/* Bottom row – Pending approvals & Recent documents */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Pending approvals */}
            <section className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4">
              <h2 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-4">
                Pending Approvals
              </h2>
              {renderDocumentList(
                pendingDocs,
                pendingLoading,
                "No documents awaiting approval."
              )}
            </section>

            {/* Recent documents */}
            <section className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4">
              <h2 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-4">
                Recent Documents
              </h2>
              {renderDocumentList(
                recentDocs,
                recentLoading,
                "No recent documents found."
              )}
            </section>
          </div>
        </main>
      </Layout>
    </>
  );
};

export default Dashboard;