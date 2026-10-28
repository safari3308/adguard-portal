import React from 'react';
import { LogOut, User as UserIcon } from 'lucide-react';
import { Trans } from '@lingui/react/macro';
import type { User } from '../types';
import { LanguageSwitcher } from './LanguageSwitcher';
import { Locale } from '../i18n';
import { BrandLogo } from './BrandLogo';

interface NavbarProps {
  user: User | null;
  currentLocale: Locale;
  onLocaleChange: (locale: Locale) => void;
  onLogout: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  user,
  currentLocale,
  onLocaleChange,
  onLogout,
}) => {
  return (
    <header className="bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 sticky top-0 z-30">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
            <BrandLogo className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight text-slate-900 dark:text-white leading-tight">
              <Trans>AdGuard Portal</Trans>
            </h1>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              <Trans>Personal DNS-over-HTTPS</Trans>
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <LanguageSwitcher currentLocale={currentLocale} onLocaleChange={onLocaleChange} />

          {user && (
            <div className="flex items-center gap-3 border-l border-slate-200 dark:border-slate-800 pl-4">
              <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-xs font-medium text-slate-700 dark:text-slate-300">
                <UserIcon className="w-3.5 h-3.5 text-slate-400" />
                <span>{user.name || user.email}</span>
              </div>
              <button
                onClick={onLogout}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-600 dark:text-slate-400 hover:text-red-600 dark:hover:text-red-400 transition"
                title="Sign out"
              >
                <LogOut className="w-4 h-4" />
                <span className="hidden sm:inline"><Trans>Logout</Trans></span>
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
