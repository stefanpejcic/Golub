// SPDX-License-Identifier: GPL-3.0-or-later

import type { Translate } from './i18n';

export function errorMessage(t: Translate, code: string): string {
  const key = `errors.${code}`;
  const msg = t(key);
  return msg === key ? t('errors.unknown') : msg;
}
