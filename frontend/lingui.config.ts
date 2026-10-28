import { formatter } from '@lingui/format-po';

export default {
  locales: ['en', 'vi', 'es', 'fr', 'de', 'zh', 'ja', 'ru'],
  sourceLocale: 'en',
  catalogs: [
    {
      path: 'src/locales/{locale}/messages',
      include: ['src'],
    },
  ],
  format: formatter({ origins: false }),
};
