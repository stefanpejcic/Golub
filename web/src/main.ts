// SPDX-License-Identifier: GPL-3.0-or-later

import { mount } from 'svelte';
import './theme/tokens.css';
import './app.css';
import App from './App.svelte';

mount(App, { target: document.getElementById('app')! });
