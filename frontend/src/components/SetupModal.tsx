import React, { useState } from 'react';
import { X, Copy, Check, Download, Smartphone, Monitor, Globe } from 'lucide-react';
import { Trans } from '@lingui/react/macro';
import type { Device } from '../types';
import { getMobileConfigUrl } from '../services/api';

interface SetupModalProps {
  device: Device;
  dohDomain: string;
  onClose: () => void;
}

export const SetupModal: React.FC<SetupModalProps> = ({ device, dohDomain, onClose }) => {
  const [copied, setCopied] = useState(false);
  const [mobileConfigUrl, setMobileConfigUrl] = useState<string>('');
  const dohUrl = `https://${dohDomain}/dns-query/${device.client_token}`;

  React.useEffect(() => {
    getMobileConfigUrl(device.id).then(setMobileConfigUrl);
  }, [device.id]);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(dohUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm animate-fade-in">
      <div className="bg-white dark:bg-slate-900 rounded-2xl shadow-2xl max-w-xl w-full border border-slate-200 dark:border-slate-800 overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <div>
            <h3 className="text-base font-semibold text-slate-900 dark:text-white">
              <Trans>Connect</Trans> “{device.name}”
            </h3>
            <p className="text-xs text-slate-500">
              <Trans>Device Token:</Trans>{' '}
              <code className="text-emerald-600 dark:text-emerald-400 font-mono">{device.client_token}</code>
            </p>
          </div>
          <button
            onClick={onClose}
            aria-label="Close modal"
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 p-1 rounded-lg"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6 max-h-[80vh] overflow-y-auto">
          {/* DoH URL Box */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-2">
              <Trans>Your Personalized DoH URL</Trans>
            </label>
            <div className="flex items-center gap-2">
              <input
                type="text"
                readOnly
                value={dohUrl}
                aria-label="DoH URL"
                className="w-full text-xs font-mono bg-slate-100 dark:bg-slate-800/80 border border-slate-300 dark:border-slate-700 rounded-lg px-3 py-2 text-slate-800 dark:text-slate-200 select-all focus:outline-none"
              />
              <button
                onClick={copyToClipboard}
                className="flex items-center gap-1.5 px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-medium rounded-lg transition shrink-0"
              >
                {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                <span>{copied ? <Trans>Copied</Trans> : <Trans>Copy</Trans>}</span>
              </button>
            </div>
          </div>

          {/* Quick Setup Options */}
          <div className="space-y-4">
            {/* Apple iOS & macOS */}
            <div className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/40">
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-center gap-2.5">
                  <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                    <Smartphone className="w-5 h-5" />
                  </div>
                  <div>
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-white">
                      <Trans>Apple iOS & macOS (1-Click Profile)</Trans>
                    </h4>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                      <Trans>Installs native encrypted DNS without installing any third-party apps.</Trans>
                    </p>
                  </div>
                </div>
                <a
                  href={mobileConfigUrl || '#'}
                  download
                  onClick={async (e) => {
                    e.preventDefault();
                    const url = await getMobileConfigUrl(device.id);
                    setMobileConfigUrl(url);
                    window.location.href = url;
                  }}
                  className="flex items-center gap-1.5 px-3 py-2 bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-slate-200 text-white dark:text-slate-900 text-xs font-medium rounded-lg transition shrink-0"
                >
                  <Download className="w-3.5 h-3.5" />
                  <span><Trans>Download .mobileconfig</Trans></span>
                </a>
              </div>
            </div>

            {/* Android */}
            <div className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/40">
              <div className="flex items-center gap-2.5 mb-2">
                <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                  <Monitor className="w-5 h-5" />
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-white">
                    <Trans>Android (Private DNS)</Trans>
                  </h4>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    <Trans>Settings → Network & internet → Private DNS → Custom hostname:</Trans>
                  </p>
                </div>
              </div>
              <div className="bg-slate-200 dark:bg-slate-800 px-3 py-1.5 rounded-lg font-mono text-xs text-slate-800 dark:text-slate-200">
                {device.client_token}.{dohDomain}
              </div>
            </div>

            {/* Web Browser */}
            <div className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/40">
              <div className="flex items-center gap-2.5 mb-2">
                <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                  <Globe className="w-5 h-5" />
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-white">
                    <Trans>Chrome, Firefox & Edge (Secure DNS)</Trans>
                  </h4>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    <Trans>Browser Settings → Privacy & Security → Use Secure DNS → Custom provider: paste the personalized DoH URL above.</Trans>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-3 bg-slate-50 dark:bg-slate-800/60 border-t border-slate-200 dark:border-slate-800 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white transition"
          >
            <Trans>Done</Trans>
          </button>
        </div>
      </div>
    </div>
  );
};
