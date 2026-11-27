// SPDX-License-Identifier: GPL-3.0-or-later

import { mount } from 'svelte';
import './theme/tokens.css';
import './app.css';
import App from './App.svelte';
import { brand } from './lib/brand.svelte';
import { ready } from './lib/locale.svelte';

void brand.load();

// the language chunk is small and loads in parallel with the rest of the page
ready.then(() => mount(App, { target: document.getElementById('app')! }));
