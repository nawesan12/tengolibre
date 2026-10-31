import { defineConfig } from 'vitest/config';
import adapterNode from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	// En el VPS todo va en el bundle: el contenedor no instala node_modules.
	ssr: { noExternal: true },
	plugins: [
		sveltekit({
			compilerOptions: {
				// Runes en todo el proyecto, salvo librerías.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapterNode({ precompress: true })
		})
	],
	test: {
		expect: { requireAssertions: true },
		include: ['src/**/*.{test,spec}.ts']
	}
});
