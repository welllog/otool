import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

import path from 'path';

export default defineConfig({
	plugins: [sveltekit()],
	resolve: {
		alias: {
			'wjs': path.resolve(__dirname, './wailsjs')
		}
	},
	server: {
		fs: {
			allow: ['./wailsjs']
		}
	}
});
