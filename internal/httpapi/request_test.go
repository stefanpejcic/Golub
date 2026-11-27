// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "testing"

func TestSupportedLocaleArabic(t *testing.T) {
	for in, want := range map[string]string{"ar": "ar", "ar-EG": "ar", "ar_SA": "ar", "sr-Cyrl": "", "fr": ""} {
		if got := supportedLocale(in); got != want {
			t.Errorf("%s: %q", in, got)
		}
	}
}
