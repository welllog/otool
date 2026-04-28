import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

import path from 'path';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
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
