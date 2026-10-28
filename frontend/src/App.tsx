import { useState, useEffect } from 'react';
import { I18nProvider } from '@lingui/react';
import { i18n } from '@lingui/core';
import { Trans } from '@lingui/react/macro';
import { Plus, ShieldAlert, Laptop, Lock, ArrowRight, Sparkles } from 'lucide-react';
import type { Device, BlockedService, User, AppConfig } from './types';
import { dynamicActivate, getSavedLocale, Locale } from './i18n';
import {
  fetchPublicConfig,
  fetchMe,
  fetchDevices,
  createDevice,
  updateDevice,
  deleteDevice,
  fetchServices,
} from './services/api';
import {
  initAuth,
  handleAuthCallback,
  loginWithOIDC,
  logout,
  setDevToken,
} from './services/auth';
import { Navbar } from './components/Navbar';
import { DeviceCard } from './components/DeviceCard';
import { SetupModal } from './components/SetupModal';
import { CustomizeModal } from './components/CustomizeModal';
import { AddDeviceModal } from './components/AddDeviceModal';
import { LanguageSwitcher } from './components/LanguageSwitcher';
import { BrandLogo } from './components/BrandLogo';

function GithubIcon({ className = 'w-4 h-4' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
      />
    </svg>
  );
}

export function App() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [services, setServices] = useState<BlockedService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [locale, setLocale] = useState<Locale>(getSavedLocale());

  // Modals
  const [showAddModal, setShowAddModal] = useState(false);
  const [setupDevice, setSetupDevice] = useState<Device | null>(null);
  const [customizeDevice, setCustomizeDevice] = useState<Device | null>(null);

  // Initialize Language & Config
  useEffect(() => {
    async function init() {
      try {
        setLoading(true);
        await dynamicActivate(locale);

        const cfg = await fetchPublicConfig();
        setConfig(cfg);

        await initAuth(cfg);
        await handleAuthCallback();

        // Check if logged in via OIDC or Dev token
        try {
          const profile = await fetchMe();
          setUser(profile);
          const [deviceList, svcList] = await Promise.all([
            fetchDevices(),
            fetchServices(),
          ]);
          setDevices(deviceList);
          setServices(svcList);
        } catch {
          // Not logged in yet
          setUser(null);
        }
      } catch (err: unknown) {
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('Failed to load application configuration');
        }
      } finally {
        setLoading(false);
      }
    }
    init();
  }, []);

  const handleLocaleChange = async (newLocale: Locale) => {
    await dynamicActivate(newLocale);
    setLocale(newLocale);
  };

  const handleDevLogin = async (email: string) => {
    setDevToken(`dev-user-${email.split('@')[0]}`);
    try {
      setLoading(true);
      const profile = await fetchMe();
      setUser(profile);
      const [deviceList, svcList] = await Promise.all([fetchDevices(), fetchServices()]);
      setDevices(deviceList);
      setServices(svcList);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    await logout();
    setUser(null);
    setDevices([]);
    setSetupDevice(null);
    setCustomizeDevice(null);
  };

  const handleAddDevice = async (name: string) => {
    const newDevice = await createDevice(name);
    setDevices([newDevice, ...devices]);
    setSetupDevice(newDevice);
  };

  const handleUpdateDevice = async (id: string, updates: Partial<Device>) => {
    const updated = await updateDevice(id, updates);
    setDevices(devices.map((d) => (d.id === id ? updated : d)));
  };

  const handleDeleteDevice = async (id: string) => {
    const confirmMsg =
      locale === 'vi'
        ? 'Bạn có chắc chắn muốn xoá thiết bị này? Quyền truy cập DoH sẽ bị thu hồi.'
        : 'Are you sure you want to delete this device? This will revoke its DoH access.';
    if (!window.confirm(confirmMsg)) {
      return;
    }
    await deleteDevice(id);
    setDevices(devices.filter((d) => d.id !== id));
  };

  if (loading) {
    return (
      <I18nProvider i18n={i18n}>
        <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950">
          <div className="flex flex-col items-center gap-3">
            <div className="w-8 h-8 border-3 border-emerald-500 border-t-transparent rounded-full animate-spin" />
            <p className="text-xs font-medium text-slate-500"><Trans>Connecting to AdGuard Portal...</Trans></p>
          </div>
        </div>
      </I18nProvider>
    );
  }

  // Not logged in: Show Login Screen
  if (!user) {
    const providerName = config?.oidc_provider_name || 'Single Sign-On (SSO)';

    return (
      <I18nProvider i18n={i18n}>
        <div className="min-h-screen flex flex-col justify-between bg-gradient-to-b from-slate-50 via-white to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950 text-slate-900 dark:text-slate-100">
          <div className="flex justify-end p-4 sm:p-6 max-w-6xl w-full mx-auto">
            <LanguageSwitcher currentLocale={locale} onLocaleChange={handleLocaleChange} />
          </div>

          <div className="max-w-md w-full mx-auto p-6 sm:p-8 text-center">
            <div className="inline-flex p-3 rounded-2xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 mb-6 shadow-sm">
              <BrandLogo className="w-12 h-12" />
            </div>

            <h2 className="text-2xl font-bold tracking-tight mb-2"><Trans>AdGuard Portal</Trans></h2>
            <p className="text-sm text-slate-600 dark:text-slate-400 mb-8">
              <Trans>Zero-trust encrypted DNS. Only authorized accounts authenticated via your identity provider can register devices and query DoH.</Trans>
            </p>

            {error && (
              <div className="mb-6 p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 text-xs text-left">
                {error}
              </div>
            )}

            <div className="space-y-3">
              <button
                onClick={() => loginWithOIDC()}
                className="w-full flex items-center justify-center gap-2 px-5 py-3 rounded-xl bg-slate-900 hover:bg-slate-800 dark:bg-white dark:hover:bg-slate-100 text-white dark:text-slate-900 font-semibold text-sm shadow-md transition"
              >
                <Lock className="w-4 h-4" />
                <span><Trans>Sign in with</Trans> {providerName}</span>
                <ArrowRight className="w-4 h-4 ml-1" />
              </button>

              {/* Dev mode login helper */}
              {(config?.dev_mode || !config?.oidc_issuer) && (
                <div className="mt-8 pt-6 border-t border-slate-200 dark:border-slate-800">
                  <div className="flex items-center justify-center gap-1.5 text-xs text-amber-600 dark:text-amber-400 font-medium mb-3">
                    <Sparkles className="w-3.5 h-3.5" />
                    <span><Trans>Dev Mode Enabled</Trans></span>
                  </div>
                  <button
                    onClick={() => handleDevLogin('admin@example.com')}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-semibold text-slate-700 dark:text-slate-300 transition"
                  >
                    <Trans>Quick Login as Developer</Trans>
                  </button>
                </div>
              )}
            </div>
          </div>

          <footer className="max-w-6xl w-full mx-auto px-4 py-6 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-400">
            <div>
              <span>AdGuard Portal &bull; Encrypted DNS over HTTPS</span>
            </div>
            <a
              href="https://github.com/safari3308/adguard-portal"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 hover:text-slate-600 dark:hover:text-slate-200 transition"
            >
              <GithubIcon className="w-4 h-4" />
              <span><Trans>Source Code on GitHub</Trans></span>
            </a>
          </footer>
        </div>
      </I18nProvider>
    );
  }

  const maxDevices = config?.max_devices_per_user || 10;
  const isLimitReached = devices.length >= maxDevices;

  return (
    <I18nProvider i18n={i18n}>
      <div className="min-h-screen bg-slate-50 dark:bg-slate-950 flex flex-col justify-between">
        <div>
          <Navbar
            user={user}
            currentLocale={locale}
            onLocaleChange={handleLocaleChange}
            onLogout={handleLogout}
          />

          <main className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            {/* Header Dashboard Banner */}
            <div className="mb-8 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white"><Trans>Your Devices</Trans></h2>
                <p className="text-xs text-slate-500 mt-1">
                  <Trans>Each device receives an isolated, authenticated DoH endpoint with custom filtering.</Trans>
                </p>
              </div>

              <button
                onClick={() => setShowAddModal(true)}
                disabled={isLimitReached}
                className="flex items-center gap-2 px-4 py-2.5 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-xs font-semibold rounded-xl shadow-sm transition"
              >
                <Plus className="w-4 h-4" />
                <span><Trans>Add Device</Trans> ({devices.length}/{maxDevices})</span>
              </button>
            </div>

            {/* Device Limit Alert if reached */}
            {isLimitReached && (
              <div className="mb-6 p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-700 dark:text-amber-400 text-xs flex items-center gap-2.5">
                <ShieldAlert className="w-4 h-4 shrink-0" />
                <span>
                  <Trans>You have reached the maximum device quota</Trans> ({maxDevices}). <Trans>Delete an inactive device to add another.</Trans>
                </span>
              </div>
            )}

            {/* Devices Grid */}
            {devices.length === 0 ? (
              <div className="text-center py-16 px-4 border-2 border-dashed border-slate-200 dark:border-slate-800 rounded-3xl bg-white dark:bg-slate-900/50">
                <div className="inline-flex p-3 rounded-2xl bg-slate-100 dark:bg-slate-800 text-slate-400 mb-3">
                  <Laptop className="w-8 h-8" />
                </div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">
                  <Trans>No devices configured yet</Trans>
                </h3>
                <p className="text-xs text-slate-500 max-w-sm mx-auto mb-5">
                  <Trans>Register your phone, tablet, laptop, or home browser to obtain your secure DoH URL.</Trans>
                </p>
                <button
                  onClick={() => setShowAddModal(true)}
                  className="inline-flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold rounded-xl transition"
                >
                  <Plus className="w-4 h-4" />
                  <span><Trans>Register Your First Device</Trans></span>
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                {devices.map((device) => (
                  <DeviceCard
                    key={device.id}
                    device={device}
                    onUpdate={handleUpdateDevice}
                    onDelete={handleDeleteDevice}
                    onOpenSetup={(d) => setSetupDevice(d)}
                    onOpenCustomize={(d) => setCustomizeDevice(d)}
                  />
                ))}
              </div>
            )}
          </main>
        </div>

        <footer className="max-w-6xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-slate-400 border-t border-slate-100 dark:border-slate-800/80 mt-12">
          <div>
            <Trans>AdGuard DoH Hostname:</Trans>{' '}
            <code className="font-mono text-emerald-600 dark:text-emerald-400 font-semibold">{config?.doh_domain}</code>
          </div>
          <a
            href="https://github.com/safari3308/adguard-portal"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 hover:text-slate-600 dark:hover:text-slate-200 transition"
          >
            <GithubIcon className="w-4 h-4" />
            <span><Trans>Source Code on GitHub</Trans></span>
          </a>
        </footer>

        {/* Modals */}
        {showAddModal && (
          <AddDeviceModal
            onAdd={handleAddDevice}
            onClose={() => setShowAddModal(false)}
          />
        )}

        {setupDevice && config && (
          <SetupModal
            device={setupDevice}
            dohDomain={config.doh_domain}
            onClose={() => setSetupDevice(null)}
          />
        )}

        {customizeDevice && (
          <CustomizeModal
            device={customizeDevice}
            availableServices={services}
            onSave={async (updates) => {
              await handleUpdateDevice(customizeDevice.id, updates);
              setCustomizeDevice(null);
            }}
            onClose={() => setCustomizeDevice(null)}
          />
        )}
      </div>
    </I18nProvider>
  );
}
