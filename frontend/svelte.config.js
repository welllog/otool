import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";
import adapter from "@sveltejs/adapter-static";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  compilerOptions: {
    runes: true,
  },
  kit: {
    adapter: adapter(),
    alias: {
      wjs: "./wailsjs",
    },
  },
  preprocess: [vitePreprocess({})],
};

export default config;
