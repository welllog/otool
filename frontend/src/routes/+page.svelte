<script>
  import { Version, OpenURL } from "wjs/go/internal/App";
  import { onMount } from "svelte";

  let version = $state('');

  function openUrl(/** @type {MouseEvent} */ e) {
      e.preventDefault();
      const href = /** @type {HTMLAnchorElement} */ (e.currentTarget).href;
      OpenURL(href);
  }

  onMount(() => {
      Version().then((v) => {
          version = v;
      });
  });
</script>

<div class="mt-20 w-full h-full text-gray-800 dark:text-gray-200 flex flex-col items-center justify-center p-4">
    <div class="bg-white dark:bg-gray-800 p-12 rounded-[2rem] shadow-2xl shadow-primary-500/5 border border-gray-100 dark:border-gray-700 flex flex-col items-center max-w-2xl text-center relative overflow-hidden w-full">
        <!-- Decorative bg -->
        <div class="absolute -top-24 -right-24 w-64 h-64 bg-primary-500/10 rounded-full blur-3xl pointer-events-none"></div>
        <div class="absolute -bottom-24 -left-24 w-64 h-64 bg-blue-500/10 rounded-full blur-3xl pointer-events-none"></div>
        
        <div class="relative z-10 flex flex-col items-center">
            <div class="w-20 h-20 mb-6 bg-gradient-to-br from-primary-500 to-blue-600 rounded-2xl flex items-center justify-center shadow-lg shadow-primary-500/30 transform rotate-3">
                <svg class="w-10 h-10 text-white transform -rotate-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
            </div>

            <h1 class="text-5xl font-extrabold mb-3 bg-gradient-to-r from-primary-600 to-blue-500 bg-clip-text text-transparent tracking-tight">OTOOL</h1>
            <div class="inline-block px-4 py-1 mb-8 bg-gray-100 dark:bg-gray-700/50 rounded-full text-sm font-semibold text-gray-600 dark:text-gray-300 border border-gray-200 dark:border-gray-600">
                Version {version || 'Unknown'}
            </div>
            
            <p class="text-lg text-gray-500 dark:text-gray-400 mb-10 max-w-md leading-relaxed">
                一款现代化的开发者实用工具箱。如果您觉得这个项目对您有帮助，欢迎给我们在 GitHub 上点个 Star ⭐。
            </p>
            
            <a onclick={openUrl} href="https://github.com/welllog/otool" class="group flex items-center gap-3 px-8 py-4 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-full font-semibold hover:bg-gray-800 dark:hover:bg-gray-100 transition-all hover:scale-105 shadow-xl shadow-gray-900/20 dark:shadow-white/20">
                <svg class="w-6 h-6 fill-current" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>
                前往 GitHub
                <svg class="w-5 h-5 ml-1 transform group-hover:translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                </svg>
            </a>
        </div>
    </div>
</div>
