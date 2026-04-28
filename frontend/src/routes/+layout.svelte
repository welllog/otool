<script lang="ts">
    import "../app.css";
    import { page } from '$app/state';
    import { onMount, onDestroy } from 'svelte';
    import { fly, fade } from 'svelte/transition';
    import { cubicOut } from 'svelte/easing';
    import DarkMode from '$lib/DarkMode.svelte';
    import {
        Navbar, NavBrand, NavUl, NavLi,
    } from "flowbite-svelte";
    import ToastContainer from '$lib/ToastContainer.svelte';
    import { toast } from '$lib/ToastContainer.svelte';
    import { EventsOn } from 'wjs/runtime/runtime';

    interface menu {
        title: string;
        href: string;
        icon?: string;
    }

    let { children } = $props();

    let currentPath = $derived(page.url.pathname);

    let title = "OTOOL";
    let topMenus: menu[] = [];
    let sideMenus: menu[] = [
        { title: "文本编解码", href: "/text", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m3.75 9v6m3-3H9m1.5-12H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" /></svg>` },
        { title: "文件编解码", href: "/file", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m6.75 12-3-3m0 0-3 3m3-3v6m-1.5-15H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" /></svg>` },
        { title: "进制转换", href: "/conversion", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>` },
        { title: "图片转换", href: "/image", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" /></svg>` },
        { title: "二维码工具", href: "/qrcode", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 3.75 9.375v-4.5ZM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 0 1-1.125-1.125v-4.5ZM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 13.5 9.375v-4.5Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 6.75h.75v.75h-.75v-.75ZM6.75 16.5h.75v.75h-.75v-.75ZM16.5 6.75h.75v.75h-.75v-.75ZM13.5 13.5h.75v.75h-.75v-.75ZM13.5 19.5h.75v.75h-.75v-.75ZM19.5 13.5h.75v.75h-.75v-.75ZM19.5 19.5h.75v.75h-.75v-.75ZM16.5 16.5h.75v.75h-.75v-.75Z" /></svg>` },
    ];

    let notifyClose = () => {};
    let breakPoint: number = 1024;
    let width = $state(0);
    let drawerOpen = $state(true);
    let lastWidth = 0;

    $effect(() => {
        if (width > 0) {
            if (lastWidth === 0) {
                drawerOpen = width >= breakPoint;
            } else if ((width >= breakPoint) !== (lastWidth >= breakPoint)) {
                drawerOpen = width >= breakPoint;
            }
            lastWidth = width;
        }
    });

    onMount(() => {
        notifyClose = EventsOn('notify', (event: any) => {
            toast(event.type, event.info);
        });
    });
    onDestroy(() => {
        notifyClose();
    });

    const toggleSide = () => {
        if (width < breakPoint) {
            drawerOpen = false;
        }
    };
    const toggleDrawer = () => {
        drawerOpen = !drawerOpen;
    };
</script>



<svelte:window bind:innerWidth={width}/>

<div class="flex flex-col h-screen overflow-hidden bg-[#fafafa] dark:bg-[#050505]">
    <!-- Toast Layer -->
    <div class="fixed top-6 right-6 z-[100] w-80 pointer-events-none space-y-4">
        <ToastContainer/>
    </div>

    <!-- Header -->
    <header class="flex-none z-40" style="--wails-draggable:drag">
        <Navbar class="px-4 py-3 w-full border-b border-gray-200/50 dark:border-gray-800/50 bg-white/70 backdrop-blur-xl dark:bg-gray-900/70 sticky top-0 shadow-sm dark:shadow-none">
            <div class="flex items-center gap-4">
                <button 
                    onclick={toggleDrawer} 
                    class="p-2 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-all active:scale-95 lg:hidden" 
                    aria-label="Toggle menu"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="22" height="22" fill="currentColor"><path
                            d="M3 4H21V6H3V4ZM3 11H21V13H3V11ZM3 18H21V20H3V18Z"></path></svg>
                </button>
                
                <NavBrand href="/" class="flex items-center gap-2 group">
                    <div class="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center shadow-lg shadow-primary-500/20 group-hover:scale-110 transition-transform">
                        <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                        </svg>
                    </div>
                    <span class="text-xl font-bold dark:text-white tracking-tight leading-none pt-1">
                        {title}
                    </span>
                </NavBrand>
            </div>
            
            <div class="flex items-center gap-4">
                <NavUl
                        classes={{ active: "text-primary-600 dark:text-primary-400 font-semibold", nonActive: "text-gray-600 hover:text-primary-600 dark:text-gray-300 dark:hover:text-primary-400 transition-colors" }}
                        activeUrl={currentPath}
                >
                    {#each topMenus as item (item.href)}
                        <NavLi href={item.href}>{item.title}</NavLi>
                    {/each}
                </NavUl>
                <div class="h-6 w-[1px] bg-gray-200 dark:bg-gray-800 mx-2"></div>
                <DarkMode/>
            </div>
        </Navbar>
    </header>

    <div class="flex-1 flex overflow-hidden relative">
        <!-- Sidebar Implementation -->
        {#if drawerOpen || width >= breakPoint}
            <aside 
                class="bg-white/80 dark:bg-gray-900/80 backdrop-blur-xl border-r border-gray-200/50 dark:border-gray-800/50 z-30 transition-all duration-300 ease-in-out
                {width < breakPoint ? 'fixed inset-y-0 left-0 shadow-2xl' : 'relative'}
                {drawerOpen ? 'w-64' : 'w-0 overflow-hidden border-none'}"
                in:fly={{ x: -20, duration: 300, easing: cubicOut }}
                out:fade={{ duration: 200 }}
            >
                <div class="h-full flex flex-col w-64 pt-4">
                    <nav class="flex-1 overflow-y-auto px-4 space-y-1.5">
                        <div class="px-4 py-2 mb-2">
                            <span class="text-[10px] font-bold text-gray-400 dark:text-gray-500 uppercase tracking-[2px]">工具箱</span>
                        </div>
                        {#each sideMenus as sm}
                            <a 
                                href={sm.href} 
                                onclick={toggleSide}
                                class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all duration-200 group
                                {currentPath === sm.href 
                                    ? 'sidebar-item-active text-white shadow-lg shadow-primary-500/20' 
                                    : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800/50 hover:text-gray-900 dark:hover:text-gray-200'}"
                            >
                                <span class="flex-shrink-0 transition-colors {currentPath === sm.href ? 'text-white' : 'text-gray-400 group-hover:text-primary-500 dark:group-hover:text-primary-400'}">
                                    {@html sm.icon}
                                </span>
                                {sm.title}
                            </a>
                        {/each}
                    </nav>
                    
                    <div class="p-6 mt-auto border-t border-gray-100 dark:border-gray-800/50">
                        <div class="bg-gray-50 dark:bg-gray-800/50 rounded-2xl p-4 text-center">
                            <p class="text-[11px] text-gray-400 dark:text-gray-500 font-medium tracking-widest uppercase">Powering by OTOOL</p>
                        </div>
                    </div>
                </div>
            </aside>
        {/if}

        <!-- Overlay for mobile -->
        {#if drawerOpen && width < breakPoint}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div 
                class="fixed inset-0 bg-black/40 sidebar-mask z-20 lg:hidden"
                onclick={() => (drawerOpen = false)}
                in:fade={{ duration: 200 }}
                out:fade={{ duration: 200 }}
            ></div>
        {/if}

        <!-- Main Content with Grid Background -->
        <main class="flex-1 overflow-y-auto relative transition-all duration-300">
            <!-- Subtle Grid Background -->
            <div class="absolute inset-0 z-0 pointer-events-none opacity-[0.03] dark:opacity-[0.06]" 
                 style="background-image: radial-gradient(#000 0.8px, transparent 0.8px); background-size: 32px 32px;">
            </div>

            <div class="relative z-10 max-w-6xl mx-auto px-6 py-10">
                {@render children()}
            </div>
        </main>
    </div>
</div>

<style>
    :global(::-webkit-scrollbar) {
        width: 6px;
        height: 6px;
    }
    :global(::-webkit-scrollbar-track) {
        background: transparent;
    }
    :global(::-webkit-scrollbar-thumb) {
        background: rgba(156, 163, 175, 0.3);
        border-radius: 10px;
    }
    :global(::-webkit-scrollbar-thumb:hover) {
        background: rgba(156, 163, 175, 0.5);
    }
    
    :global(.sidebar-mask) {
        backdrop-blur: 4px;
    }

    :global(.sidebar-item-active) {
        background: #2563eb; /* primary-600 - Unified Blue Brand Color */
        position: relative;
        overflow: hidden;
    }
    
    :global(.sidebar-item-active::before) {
        content: '';
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        width: 4px;
        background: rgba(255, 255, 255, 0.5);
        filter: blur(3px);
        z-index: 10;
    }

    :global(.sidebar-item-active::after) {
        content: '';
        position: absolute;
        inset: 0;
        background: linear-gradient(120deg, rgba(255,255,255,0.15), transparent);
        pointer-events: none;
    }

    /* Dark mode icon enhancement */
    :global(.dark svg) {
        filter: drop-shadow(0 0 1px rgba(255, 255, 255, 0.05));
    }
</style>
