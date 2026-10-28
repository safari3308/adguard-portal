import { i18n } from '@lingui/core';

export const locales = {
  en: 'English',
  vi: 'Tiếng Việt',
  es: 'Español',
  fr: 'Français',
  de: 'Deutsch',
  zh: '简体中文',
  ja: '日本語',
  ru: 'Русский',
} as const;

export type Locale = keyof typeof locales;

export const defaultLocale: Locale = 'en';

export async function dynamicActivate(locale: Locale) {
  let messages;
  switch (locale) {
    case 'vi': {
      const mod = await import('./locales/vi/messages.po');
      messages = mod.messages;
      break;
    }
    case 'es': {
      const mod = await import('./locales/es/messages.po');
      messages = mod.messages;
      break;
    }
    case 'fr': {
      const mod = await import('./locales/fr/messages.po');
      messages = mod.messages;
      break;
    }
    case 'de': {
      const mod = await import('./locales/de/messages.po');
      messages = mod.messages;
      break;
    }
    case 'zh': {
      const mod = await import('./locales/zh/messages.po');
      messages = mod.messages;
      break;
    }
    case 'ja': {
      const mod = await import('./locales/ja/messages.po');
      messages = mod.messages;
      break;
    }
    case 'ru': {
      const mod = await import('./locales/ru/messages.po');
      messages = mod.messages;
      break;
    }
    case 'en':
    default: {
      const mod = await import('./locales/en/messages.po');
      messages = mod.messages;
      break;
    }
  }

  i18n.load(locale, messages);
  i18n.activate(locale);
  localStorage.setItem('agh_locale', locale);
}

export function getSavedLocale(): Locale {
  const saved = localStorage.getItem('agh_locale') as Locale | null;
  if (saved && saved in locales) {
    return saved;
  }
  // Try matching browser language prefix
  const browserLang = navigator.language.slice(0, 2).toLowerCase() as Locale;
  if (browserLang in locales) {
    return browserLang;
  }
  return defaultLocale;
}
