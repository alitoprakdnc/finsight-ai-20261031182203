import React, { useState, useEffect, useCallback, ReactNode } from "react";
import Link from "next/link";
import { useRouter } from "next/router";
import {
  HomeIcon,
  DocumentTextIcon,
  CogIcon,
  MoonIcon,
  SunIcon,
  Bars3Icon,
  XMarkIcon,
} from "@heroicons/react/24/outline";

type NavItem = {
  name: string;
  href: string;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
};

const NAV_ITEMS: NavItem[] = [
  { name: "Dashboard", href: "/dashboard", icon: HomeIcon },
  { name: "Documents", href: "/documents", icon: DocumentTextIcon },
  { name: "Settings", href: "/settings", icon: CogIcon },
];

type LayoutProps = {
  children: ReactNode;
};

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const router = useRouter();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [isDark, setIsDark] = useState<boolean>(false);

  // Initialise theme from localStorage or system preference
  useEffect(() => {
    const stored = localStorage.getItem("theme");
    if (stored === "dark") {
      setIsDark(true);
    } else if (stored === "light") {
      setIsDark(false);
    } else {
      // fallback to system preference
      const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      setIsDark(prefersDark);
    }
  }, []);

  // Apply theme class to <html>
  useEffect(() => {
    const root = window.document.documentElement;
    if (isDark) {
      root.classList.add("dark");
      localStorage.setItem("theme", "dark");
    } else {
      root.classList.remove("dark");
      localStorage.setItem("theme", "light");
    }
  }, [isDark]);

  const toggleTheme = useCallback(() => setIsDark((prev) => !prev), []);

  const closeMobileMenu = useCallback(() => setMobileOpen(false), []);

  const isActive = (href: string) => router.pathname === href;

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Mobile sidebar */}
      <div
        className={`fixed inset-0 z-40 md:hidden transition-opacity ${
          mobileOpen ? "opacity-100" : "opacity-0 pointer-events-none"
        }`}
        aria-hidden={!mobileOpen}
      >
        <div
          className="absolute inset-0 bg-black bg-opacity-25"
          onClick={closeMobileMenu}
        />
        <aside
          className="absolute inset-y-0 left-0 w-64 bg-white dark:bg-gray-800 shadow-lg p-4 overflow-y-auto"
          role="dialog"
          aria-modal="true"
        >
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-gray-800 dark:text-gray-200">
              FinSight AI
            </h2>
            <button
              type="button"
              className="p-2 rounded-md text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
              onClick={closeMobileMenu}
              aria-label="Close menu"
            >
              <XMarkIcon className="h-6 w-6" aria-hidden="true" />
            </button>
          </div>
          <nav>
            <ul className="space-y-2">
              {NAV_ITEMS.map((item) => (
                <li key={item.name}>
                  <Link href={item.href}>
                    <a
                      className={`flex items-center px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                        isActive(item.href)
                          ? "bg-primary-100 text-primary-600 dark:bg-primary-900 dark:text-primary-300"
                          : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                      }`}
                      onClick={closeMobileMenu}
                    >
                      <item.icon
                        className="h-5 w-5 mr-3 flex-shrink-0"
                        aria-hidden="true"
                      />
                      {item.name}
                    </a>
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        </aside>
      </div>

      {/* Desktop sidebar */}
      <aside className="hidden md:flex md:flex-col md:w-64 md:fixed md:inset-y-0 md:bg-white dark:md:bg-gray-800 md:shadow-lg">
        <div className="flex flex-col h-full p-4">
          <div className="flex items-center justify-between mb-8">
            <h1 className="text-2xl font-bold text-primary-600 dark:text-primary-400">
              FinSight AI
            </h1>
          </div>
          <nav className="flex-1">
            <ul className="space-y-1">
              {NAV_ITEMS.map((item) => (
                <li key={item.name}>
                  <Link href={item.href}>
                    <a
                      className={`flex items-center px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                        isActive(item.href)
                          ? "bg-primary-100 text-primary-600 dark:bg-primary-900 dark:text-primary-300"
                          : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                      }`}
                    >
                      <item.icon
                        className="h-5 w-5 mr-3 flex-shrink-0"
                        aria-hidden="true"
                      />
                      {item.name}
                    </a>
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
          <button
            type="button"
            onClick={toggleTheme}
            className="mt-4 flex items-center px-3 py-2 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
            aria-label="Toggle dark mode"
          >
            {isDark ? (
              <SunIcon className="h-5 w-5 mr-2" aria-hidden="true" />
            ) : (
              <MoonIcon className="h-5 w-5 mr-2" aria-hidden="true" />
            )}
            {isDark ? "Light Mode" : "Dark Mode"}
          </button>
        </div>
      </aside>

      {/* Main content area */}
      <div className="flex flex-col flex-1 md:ml-64">
        {/* Header */}
        <header className="flex items-center justify-between bg-white dark:bg-gray-800 px-4 py-3 shadow-sm md:px-6">
          <div className="flex items-center">
            {/* Mobile menu button */}
            <button
              type="button"
              className="md:hidden p-2 rounded-md text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
            >
              <Bars3Icon className="h-6 w-6" aria-hidden="true" />
            </button>
            <h2 className="ml-2 text-lg font-semibold text-gray-800 dark:text-gray-200">
              {NAV_ITEMS.find((i) => isActive(i.href))?.name || "FinSight AI"}
            </h2>
          </div>

          <div className="flex items-center space-x-4">
            {/* Theme toggle (desktop) */}
            <button
              type="button"
              onClick={toggleTheme}
              className="p-2 rounded-full text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
              aria-label="Toggle dark mode"
            >
              {isDark ? (
                <SunIcon className="h-5 w-5" aria-hidden="true" />
              ) : (
                <MoonIcon className="h-5 w-5" aria-hidden="true" />
              )}
            </button>

            {/* User avatar placeholder */}
            <div className="relative">
              <button
                type="button"
                className="flex text-sm rounded-full focus:outline-none focus:ring-2 focus:ring-primary-500"
                id="user-menu-button"
                aria-expanded="false"
                aria-haspopup="true"
              >
                <span className="sr-only">Open user menu</span>
                <img
                  className="h-8 w-8 rounded-full"
                  src="/favicon.ico"
                  alt="User avatar"
                />
              </button>
              {/* Future dropdown could be added here */}
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-4 md:p-6">{children}</main>
      </div>
    </div>
  );
};

export default Layout;