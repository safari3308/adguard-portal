import React from 'react';
import { Languages } from 'lucide-react';
import { locales, Locale } from '../i18n';

interface LanguageSwitcherProps {
  currentLocale: Locale;
  onLocaleChange: (locale: Locale) => void;
}

export const LanguageSwitcher: React.FC<LanguageSwitcherProps> = ({
  currentLocale,
  onLocaleChange,
}) => {
  return (
    <div className="flex items-center gap-1.5 text-xs">
      <Languages className="w-3.5 h-3.5 text-slate-400" />
      <select
        value={currentLocale}
        onChange={(e) => onLocaleChange(e.target.value as Locale)}
        aria-label="Select language"
        className="bg-transparent text-slate-600 dark:text-slate-300 font-medium cursor-pointer focus:outline-none hover:text-slate-900 dark:hover:text-white"
      >
        {Object.entries(locales).map(([code, label]) => (
          <option key={code} value={code} className="bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100">
            {label}
          </option>
        ))}
      </select>
    </div>
  );
};
