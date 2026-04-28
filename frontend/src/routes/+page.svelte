<script>
    import { fly, fade } from 'svelte/transition';
    import { onMount } from 'svelte';
    import { Version, OpenURL } from "wjs/go/internal/App";

    let mounted = $state(false);
    let version = $state('');

    onMount(() => {
        mounted = true;
        Version().then(v => version = v);
    });

    const tools = [
        { 
            name: "图片转换", 
            desc: "格式转换、质量优化、Alpha通道处理", 
            icon: "M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
            href: "/image",
            color: "from-blue-500 to-cyan-400",
            badge: "PRO"
        },
        { 
            name: "文本处理", 
            desc: "AES加密、哈希计算、编码转换", 
            icon: "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z",
            href: "/text",
            color: "from-purple-500 to-indigo-400",
            badge: "FAST"
        },
        { 
            name: "文件加解密", 
            desc: "大文件AES加密、校验哈希计算", 
            icon: "M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z",
            href: "/file",
            color: "from-amber-500 to-orange-400",
            badge: "SAFE"
        },
        { 
            name: "二维码工具", 
            desc: "识别与生成、支持多种样式", 
            icon: "M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z",
            href: "/qrcode",
            color: "from-emerald-500 to-teal-400",
            badge: "NEW"
        },
        { 
            name: "进制转换", 
            desc: "实时转换多进制数值", 
            icon: "M7 20l4-16m2 16l4-16M6 9h14M4 15h14",
            href: "/conversion",
            color: "from-rose-500 to-pink-400",
            badge: "MATH"
        }
    ];

    function openUrl(/** @type {MouseEvent} */ e) {
        e.preventDefault();
        const href = /** @type {HTMLAnchorElement} */ (e.currentTarget).href;
        OpenURL(href);
    }
</script>

<div class="flex flex-col gap-12 py-10 px-6 max-w-7xl mx-auto min-h-screen">
    <!-- Hero Header -->
    <header class="text-center space-y-4 relative">
        <div class="absolute inset-x-0 -top-20 -z-10 flex justify-center">
            <div class="w-[500px] h-[500px] bg-primary-500/5 rounded-full blur-[100px] animate-pulse"></div>
        </div>

        <div in:fly={{ y: 20, duration: 800 }} class="inline-flex items-center gap-2 px-4 py-1.5 bg-white/50 dark:bg-gray-800/50 backdrop-blur-xl rounded-full border border-gray-100 dark:border-white/5 shadow-sm mb-4">
            <span class="flex h-2 w-2 rounded-full bg-primary-500 animate-ping"></span>
            <span class="text-[10px] font-black text-gray-400 dark:text-gray-500 uppercase tracking-[0.2em]">OTOOL v{version || '1.2'}</span>
        </div>

        <h1 in:fly={{ y: 20, duration: 800, delay: 100 }} class="text-6xl font-black tracking-tighter text-gray-900 dark:text-white leading-[1.1]">
            极客必备的 <br/> 
            <span class="bg-gradient-to-r from-primary-600 via-blue-500 to-indigo-600 bg-clip-text text-transparent">
                全能开发者工具箱
            </span>
        </h1>
        
        <p in:fly={{ y: 20, duration: 800, delay: 200 }} class="text-gray-500 dark:text-gray-400 max-w-xl mx-auto text-lg font-medium leading-relaxed pt-4">
            专为开发者打造的极致工具集，<br/>
            高效、极简、完全本地化的数据处理体验。
        </p>

        <div in:fade={{ delay: 400 }} class="pt-6">
            <a onclick={openUrl} href="https://github.com/welllog/otool" class="inline-flex items-center gap-2 px-8 py-3 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-2xl font-black text-xs tracking-widest uppercase hover:scale-105 transition-all shadow-xl active:scale-95">
                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>
                GitHub 项目
            </a>
        </div>
    </header>

    <!-- Grid Layout -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
        {#if mounted}
            {#each tools as tool, i}
                <a 
                    href={tool.href}
                    in:fly={{ y: 30, duration: 800, delay: i * 100 }}
                    class="group relative block"
                >
                    <!-- Shadow Glow -->
                    <div class="absolute -inset-1 bg-gradient-to-r {tool.color} rounded-[32px] blur opacity-0 group-hover:opacity-20 transition-opacity duration-500"></div>
                    
                    <div class="relative h-full bg-white/70 dark:bg-gray-800/40 backdrop-blur-2xl border border-gray-100 dark:border-white/10 rounded-[32px] p-8 transition-all duration-500 group-hover:-translate-y-2 group-hover:border-primary-500/30 overflow-hidden shadow-xl shadow-gray-200/20 dark:shadow-black/20">
                        
                        <!-- Badge -->
                        <div class="absolute top-6 right-6 px-2.5 py-1 rounded-full bg-gray-100 dark:bg-white/5 border border-gray-200/50 dark:border-white/10">
                            <span class="text-[9px] font-black tracking-[0.2em] text-gray-400 group-hover:text-primary-500 transition-colors">{tool.badge}</span>
                        </div>

                        <!-- Gradient Background Highlight -->
                        <div class="absolute -right-10 -bottom-10 w-40 h-40 bg-gradient-to-br {tool.color} opacity-[0.03] group-hover:opacity-[0.08] rounded-full transition-opacity duration-500"></div>

                        <div class="flex flex-col h-full gap-6">
                            <!-- Icon Wrapper -->
                            <div class="w-16 h-16 rounded-[24px] bg-gradient-to-br {tool.color} p-4 shadow-lg shadow-gray-400/10 transition-transform duration-700 group-hover:rotate-[10deg] group-hover:scale-110">
                                <svg class="w-full h-full text-white drop-shadow-md" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                                    <path stroke-linecap="round" stroke-linejoin="round" d={tool.icon} />
                                </svg>
                            </div>

                            <!-- Content -->
                            <div class="space-y-3">
                                <h3 class="text-2xl font-black text-gray-900 dark:text-white flex items-center gap-2">
                                    {tool.name}
                                    <svg class="w-5 h-5 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-300 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
                                    </svg>
                                </h3>
                                <p class="text-sm font-medium text-gray-400 dark:text-gray-500 leading-relaxed">
                                    {tool.desc}
                                </p>
                            </div>
                        </div>

                        <!-- Bottom Decorative Bar -->
                        <div class="absolute bottom-0 left-0 h-1.5 bg-gradient-to-r {tool.color} opacity-0 group-hover:opacity-100 transition-opacity duration-500 w-full"></div>
                    </div>
                </a>
            {/each}
        {/if}
    </div>

    <!-- Footer Status -->
    <footer class="mt-auto pt-10 border-t border-gray-100 dark:border-white/5 flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="flex items-center gap-4 text-xs font-black uppercase tracking-widest text-gray-400">
            <div class="flex items-center gap-2">
                <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse shadow-[0_0_8px_rgba(34,197,94,0.5)]"></div>
                系统状态: 正常
            </div>
            <div class="w-px h-3 bg-gray-200 dark:bg-gray-800"></div>
            <div>本地存储: 已就绪</div>
        </div>
        <div class="text-[10px] font-bold text-gray-300 dark:text-gray-600 uppercase tracking-[0.3em]">
            Developed with Advanced Agentic Coding
        </div>
    </footer>
</div>
