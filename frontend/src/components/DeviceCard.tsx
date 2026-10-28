import React from 'react';
import { Smartphone, Shield, Sliders, ExternalLink, Trash2, Baby, Search } from 'lucide-react';
import { Trans } from '@lingui/react/macro';
import type { Device } from '../types';

interface DeviceCardProps {
  device: Device;
  onUpdate: (id: string, updates: Partial<Device>) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onOpenSetup: (device: Device) => void;
  onOpenCustomize: (device: Device) => void;
}

export const DeviceCard: React.FC<DeviceCardProps> = ({
  device,
  onUpdate,
  onDelete,
  onOpenSetup,
  onOpenCustomize,
}) => {
  const toggleFiltering = () => {
    onUpdate(device.id, { filtering_enabled: !device.filtering_enabled });
  };

  const toggleSafeSearch = () => {
    onUpdate(device.id, { safe_search_enabled: !device.safe_search_enabled });
  };

  const toggleParental = () => {
    onUpdate(device.id, { parental_enabled: !device.parental_enabled });
  };

  const totalRules =
    (device.blocked_services?.length || 0) +
    (device.custom_blocked_domains?.length || 0) +
    (device.custom_allowed_domains?.length || 0);

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-5 shadow-sm hover:shadow-md transition duration-200 flex flex-col justify-between">
      <div>
        {/* Device Header */}
        <div className="flex items-start justify-between gap-3 mb-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300">
              <Smartphone className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-slate-900 dark:text-white leading-tight">
                {device.name}
              </h3>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className="font-mono text-[11px] text-slate-500 bg-slate-100 dark:bg-slate-800/80 px-2 py-0.5 rounded">
                  {device.client_token}
                </span>
              </div>
            </div>
          </div>

          <button
            onClick={() => onDelete(device.id)}
            className="p-1.5 text-slate-400 hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-950/20 transition"
            title="Delete Device"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>

        {/* Feature Toggles */}
        <div className="space-y-2.5 my-4 bg-slate-50 dark:bg-slate-800/40 p-3.5 rounded-xl border border-slate-100 dark:border-slate-800/80">
          {/* Ad Blocking Toggle */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
              <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
                <Trans>Ad & Tracker Blocking</Trans>
              </span>
            </div>
            <button
              type="button"
              onClick={toggleFiltering}
              aria-label="Toggle Ad & Tracker Blocking"
              className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                device.filtering_enabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
              }`}
            >
              <span
                className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  device.filtering_enabled ? 'translate-x-4' : 'translate-x-0'
                }`}
              />
            </button>
          </div>

          {/* SafeSearch Toggle */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Search className="w-4 h-4 text-sky-600 dark:text-sky-400" />
              <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
                <Trans>SafeSearch Enforced</Trans>
              </span>
            </div>
            <button
              type="button"
              onClick={toggleSafeSearch}
              aria-label="Toggle SafeSearch Enforced"
              className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                device.safe_search_enabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
              }`}
            >
              <span
                className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  device.safe_search_enabled ? 'translate-x-4' : 'translate-x-0'
                }`}
              />
            </button>
          </div>

          {/* Parental Control Toggle */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Baby className="w-4 h-4 text-amber-600 dark:text-amber-400" />
              <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
                <Trans>Parental Control</Trans>
              </span>
            </div>
            <button
              type="button"
              onClick={toggleParental}
              aria-label="Toggle Parental Control"
              className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                device.parental_enabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
              }`}
            >
              <span
                className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  device.parental_enabled ? 'translate-x-4' : 'translate-x-0'
                }`}
              />
            </button>
          </div>
        </div>
      </div>

      {/* Action Footer */}
      <div className="pt-3 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2">
        <button
          onClick={() => onOpenCustomize(device)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
        >
          <Sliders className="w-3.5 h-3.5" />
          <span><Trans>Rules</Trans> ({totalRules})</span>
        </button>

        <button
          onClick={() => onOpenSetup(device)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 hover:bg-emerald-100 dark:hover:bg-emerald-900/50 transition"
        >
          <ExternalLink className="w-3.5 h-3.5" />
          <span><Trans>Setup Guide</Trans></span>
        </button>
      </div>
    </div>
  );
};
