// SPDX-License-Identifier: GPL-3.0-or-later

import { mount } from 'svelte';
import './theme/tokens.css';
import './app.css';
import App from './App.svelte';
import { brand } from './lib/brand.svelte';

void brand.load();

mount(App, { target: document.getElementById('app')! });
