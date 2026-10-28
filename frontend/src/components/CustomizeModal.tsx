import React, { useState } from 'react';
import { X, ShieldAlert, ShieldCheck, Plus, Trash2, Sliders } from 'lucide-react';
import { Trans } from '@lingui/react/macro';
import type { Device, BlockedService } from '../types';

interface CustomizeModalProps {
  device: Device;
  availableServices: BlockedService[];
  onSave: (updates: Partial<Device>) => Promise<void>;
  onClose: () => void;
}

export const CustomizeModal: React.FC<CustomizeModalProps> = ({
  device,
  availableServices,
  onSave,
  onClose,
}) => {
  const [blockedServices, setBlockedServices] = useState<string[]>(device.blocked_services || []);
  const [blockedDomains, setBlockedDomains] = useState<string[]>(device.custom_blocked_domains || []);
  const [allowedDomains, setAllowedDomains] = useState<string[]>(device.custom_allowed_domains || []);

  const [newBlockedInput, setNewBlockedInput] = useState('');
  const [newAllowedInput, setNewAllowedInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggleService = (id: string) => {
    if (blockedServices.includes(id)) {
      setBlockedServices(blockedServices.filter((s) => s !== id));
    } else {
      setBlockedServices([...blockedServices, id]);
    }
  };

  const addBlockedDomain = () => {
    const domain = newBlockedInput.trim().toLowerCase();
    if (!domain) return;
    if (blockedDomains.length >= 50) {
      setError('Maximum 50 custom blocked domains allowed');
      return;
    }
    if (!blockedDomains.includes(domain)) {
      setBlockedDomains([...blockedDomains, domain]);
    }
    setNewBlockedInput('');
  };

  const addAllowedDomain = () => {
    const domain = newAllowedInput.trim().toLowerCase();
    if (!domain) return;
    if (allowedDomains.length >= 50) {
      setError('Maximum 50 custom allowed domains allowed');
      return;
    }
    if (!allowedDomains.includes(domain)) {
      setAllowedDomains([...allowedDomains, domain]);
    }
    setNewAllowedInput('');
  };

  const removeBlockedDomain = (domain: string) => {
    setBlockedDomains(blockedDomains.filter((d) => d !== domain));
  };

  const removeAllowedDomain = (domain: string) => {
    setAllowedDomains(allowedDomains.filter((d) => d !== domain));
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await onSave({
        blocked_services: blockedServices,
        custom_blocked_domains: blockedDomains,
        custom_allowed_domains: allowedDomains,
      });
      onClose();
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to update customization');
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm animate-fade-in">
      <div className="bg-white dark:bg-slate-900 rounded-2xl shadow-2xl max-w-2xl w-full border border-slate-200 dark:border-slate-800 overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
              <Sliders className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-slate-900 dark:text-white">
                <Trans>Customize Filtering</Trans> - {device.name}
              </h3>
              <p className="text-xs text-slate-500"><Trans>Tune blocked services and domain overrides</Trans></p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 p-1 rounded-lg"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content Body */}
        <div className="p-6 space-y-6 overflow-y-auto">
          {error && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 text-xs">
              {error}
            </div>
          )}

          {/* Blocked Services Section */}
          <div>
            <h4 className="text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-2">
              <Trans>Block Pre-configured Services</Trans>
            </h4>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
              <Trans>One-click blocking evaluated directly by AdGuard Home with zero performance loss.</Trans>
            </p>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
              {availableServices.map((svc) => {
                const isBlocked = blockedServices.includes(svc.id);
                return (
                  <button
                    key={svc.id}
                    type="button"
                    onClick={() => toggleService(svc.id)}
                    className={`flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium border transition ${
                      isBlocked
                        ? 'bg-red-50 dark:bg-red-950/30 border-red-300 dark:border-red-800 text-red-700 dark:text-red-300'
                        : 'bg-slate-50 dark:bg-slate-800/40 border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-300 hover:border-slate-300'
                    }`}
                  >
                    <span>{svc.name}</span>
                    <span
                      className={`w-2 h-2 rounded-full ${
                        isBlocked ? 'bg-red-500' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    />
                  </button>
                );
              })}
            </div>
          </div>

          <hr className="border-slate-200 dark:border-slate-800" />

          {/* Custom Blocked Domains */}
          <div>
            <div className="flex items-center gap-2 mb-1">
              <ShieldAlert className="w-4 h-4 text-red-500" />
              <h4 className="text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider">
                <Trans>Custom Blocked Domains</Trans> ({blockedDomains.length}/50)
              </h4>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
              <Trans>Blocks specific domains exclusively on this device (e.g. distracting-site.com).</Trans>
            </p>
            <div className="flex gap-2 mb-3">
              <input
                type="text"
                placeholder="domain.com"
                value={newBlockedInput}
                onChange={(e) => setNewBlockedInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && addBlockedDomain()}
                aria-label="Blocked domain"
                className="flex-1 text-xs px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-emerald-500"
              />
              <button
                type="button"
                onClick={addBlockedDomain}
                className="flex items-center gap-1 px-3 py-2 bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 text-xs font-medium rounded-lg hover:opacity-90"
              >
                <Plus className="w-3.5 h-3.5" />
                <Trans>Add</Trans>
              </button>
            </div>
            {blockedDomains.length > 0 && (
              <div className="flex flex-wrap gap-1.5 max-h-32 overflow-y-auto p-2 rounded-lg bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800">
                {blockedDomains.map((d) => (
                  <span
                    key={d}
                    className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-mono bg-red-100 dark:bg-red-950/50 text-red-700 dark:text-red-300 border border-red-200 dark:border-red-900"
                  >
                    {d}
                    <button
                      onClick={() => removeBlockedDomain(d)}
                      aria-label={`Remove ${d}`}
                      className="text-red-400 hover:text-red-700 ml-1"
                    >
                      <Trash2 className="w-3 h-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>

          <hr className="border-slate-200 dark:border-slate-800" />

          {/* Custom Allowed Domains (Whitelist override) */}
          <div>
            <div className="flex items-center gap-2 mb-1">
              <ShieldCheck className="w-4 h-4 text-emerald-500" />
              <h4 className="text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider">
                <Trans>Allowlist Overrides</Trans> ({allowedDomains.length}/50)
              </h4>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
              <Trans>Unblocks domains that might be blocked by global AdGuard filter lists.</Trans>
            </p>
            <div className="flex gap-2 mb-3">
              <input
                type="text"
                placeholder="trusted-site.com"
                value={newAllowedInput}
                onChange={(e) => setNewAllowedInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && addAllowedDomain()}
                aria-label="Allowed domain"
                className="flex-1 text-xs px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-emerald-500"
              />
              <button
                type="button"
                onClick={addAllowedDomain}
                className="flex items-center gap-1 px-3 py-2 bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 text-xs font-medium rounded-lg hover:opacity-90"
              >
                <Plus className="w-3.5 h-3.5" />
                <Trans>Add</Trans>
              </button>
            </div>
            {allowedDomains.length > 0 && (
              <div className="flex flex-wrap gap-1.5 max-h-32 overflow-y-auto p-2 rounded-lg bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800">
                {allowedDomains.map((d) => (
                  <span
                    key={d}
                    className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-mono bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-900"
                  >
                    {d}
                    <button
                      onClick={() => removeAllowedDomain(d)}
                      aria-label={`Remove ${d}`}
                      className="text-emerald-400 hover:text-emerald-700 ml-1"
                    >
                      <Trash2 className="w-3 h-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 bg-slate-50 dark:bg-slate-800/60 border-t border-slate-200 dark:border-slate-800 flex items-center justify-end gap-3 shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-xs font-semibold text-slate-600 dark:text-slate-400 hover:text-slate-800"
          >
            <Trans>Cancel</Trans>
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-xs font-semibold rounded-lg shadow-sm transition"
          >
            {saving ? <Trans>Applying...</Trans> : <Trans>Save Settings</Trans>}
          </button>
        </div>
      </div>
    </div>
  );
};
