import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		headers: {
			'Cache-Control': 'no-store, max-age=0',
			Pragma: 'no-cache',
			Expires: '0'
		}
	}
});
