// SPDX-License-Identifier: GPL-3.0-or-later
// Runs before first paint so the page never flashes the wrong theme.
(function () {
  var m = document.cookie.match(/(?:^|;\s*)golub_theme=(light|dark|system)(?:;|$)/);
  document.documentElement.dataset.theme = m ? m[1] : 'system';
})();
