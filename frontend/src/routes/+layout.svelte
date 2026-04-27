<script lang="ts">
    import "../app.css";
    import { page } from '$app/state';
    import { onMount, onDestroy } from 'svelte';
    import { sineIn } from 'svelte/easing';
    import DarkMode from '$lib/DarkMode.svelte';
    import {
        Navbar, NavBrand, NavUl, NavLi,
        Drawer, Sidebar, SidebarGroup, SidebarItem, SidebarWrapper, CloseButton,
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
        { title: "文本编解码", href: "/text", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-gray-500 transition duration-75 group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-400"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m3.75 9v6m3-3H9m1.5-12H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" /></svg>` },
        { title: "文件编解码", href: "/file", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-gray-500 transition duration-75 group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-400"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m6.75 12-3-3m0 0-3 3m3-3v6m-1.5-15H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" /></svg>` },
        { title: "进制转换", href: "/conversion", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-gray-500 transition duration-75 group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-400"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>` },
        { title: "图片转换", href: "/image", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-gray-500 transition duration-75 group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-400"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" /></svg>` },
        { title: "二维码工具", href: "/qrcode", icon: `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-gray-500 transition duration-75 group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-400"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 3.75 9.375v-4.5ZM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 0 1-1.125-1.125v-4.5ZM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 13.5 9.375v-4.5Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 6.75h.75v.75h-.75v-.75ZM6.75 16.5h.75v.75h-.75v-.75ZM16.5 6.75h.75v.75h-.75v-.75ZM13.5 13.5h.75v.75h-.75v-.75ZM13.5 19.5h.75v.75h-.75v-.75ZM19.5 13.5h.75v.75h-.75v-.75ZM19.5 19.5h.75v.75h-.75v-.75ZM16.5 16.5h.75v.75h-.75v-.75Z" /></svg>` },
    ];

    let notifyClose = () => {};
    let height = $state(0);
    let transitionParams = {
        x: -320,
        duration: 200,
        easing: sineIn
    };
    let breakPoint: number = 1024;
    let width = $state(0);
    let drawerOpen = $state(false);
    let lastWidth = 0;

    $effect(() => {
        // 第一次获取到有效宽度时进行初始化
        if (lastWidth === 0 && width > 0) {
            drawerOpen = width >= breakPoint;
        } else {
            // 仅在窗口尺寸跨越断点时同步状态，平时允许手动覆盖
            const isLarge = width >= breakPoint;
            const wasLarge = lastWidth >= breakPoint;
            
            if (isLarge !== wasLarge) {
                drawerOpen = isLarge;
            }
        }
        lastWidth = width;
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
            drawerOpen = !drawerOpen;
        }
    };
    const toggleDrawer = () => {
        drawerOpen = !drawerOpen;
    };
</script>

<style>
    ::-webkit-scrollbar {
        width: 0;
        height: 0;
        display: none;
    }

    ::-webkit-scrollbar-thumb {
        background-color: transparent;
    }
</style>

<svelte:window bind:innerWidth={width}/>

<div class="flex flex-col h-screen overflow-hidden bg-slate-50 dark:bg-slate-950">
    <div class="fixed top-14 right-4 z-50 h-3/4 overflow-scroll pointer-events-none">
        <div class="pointer-events-auto">
            <ToastContainer/>
        </div>
    </div>

    <header class="flex-none z-30" style="--wails-draggable:drag">
        <Navbar class="px-0 py-0 w-full border-b border-gray-200 dark:border-gray-800 bg-white/80 backdrop-blur-md dark:bg-gray-900/80 sticky top-0 shadow-sm dark:shadow-none">
            <span class="flex items-center lg:ml-64 ml-14">
                <button onclick={toggleDrawer} class="dark:text-white mr-4 transition-colors hover:text-primary-600 dark:hover:text-primary-400" aria-label="Toggle menu">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24" fill="currentColor"><path
                            d="M21 17.9995V19.9995H3V17.9995H21ZM17.4038 3.90332L22 8.49951L17.4038 13.0957L15.9896 11.6815L19.1716 8.49951L15.9896 5.31753L17.4038 3.90332ZM12 10.9995V12.9995H3V10.9995H12ZM12 3.99951V5.99951H3V3.99951H12Z"></path></svg>
                </button>
                <NavBrand href="/">
                    <span class="ml-0 whitespace-nowrap text-xl font-bold dark:text-white tracking-tight">
                        {title}
                    </span>
                </NavBrand>
            </span>
            <NavUl
                    classes={{ active: "text-primary-700 dark:text-primary-500", nonActive: "text-gray-700 hover:text-primary-700 dark:text-white dark:hover:text-primary-700" }}
                    activeUrl={currentPath}
            >
                {#each topMenus as item (item.href)}
                    <NavLi href={item.href}>{item.title}</NavLi>
                {/each}
            </NavUl>
            <DarkMode/>
        </Navbar>
    </header>

    <div class="flex-1 flex overflow-hidden relative bg-slate-50 dark:bg-slate-950">
        <!-- Sidebar Implementation -->
        <aside 
            class="bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 z-20 transition-all duration-300 shadow-sm
            {width < breakPoint ? 'fixed inset-y-0 left-0 h-full' : 'relative h-full'}
            {drawerOpen ? 'w-64' : 'w-0 overflow-hidden border-none'}"
        >
            <div class="h-full flex flex-col w-64">
                <nav class="flex-1 overflow-y-auto py-6 px-4 space-y-1">
                    {#each sideMenus as sm}
                        <a 
                            href={sm.href} 
                            onclick={toggleSide}
                            class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all duration-200 group
                            {currentPath === sm.href 
                                ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' 
                                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800/60 hover:text-gray-900 dark:hover:text-gray-200'}"
                        >
                            <span class="flex-shrink-0 transition-colors {currentPath === sm.href ? 'text-primary-600 dark:text-primary-400' : 'text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-300'}">
                                {@html sm.icon}
                            </span>
                            {sm.title}
                        </a>
                    {/each}
                </nav>
            </div>
        </aside>

        <!-- Overlay for mobile -->
        {#if drawerOpen && width < breakPoint}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div 
                class="fixed inset-0 bg-black/20 backdrop-blur-sm z-10 lg:hidden"
                onclick={() => (drawerOpen = false)}
            ></div>
        {/if}

        <main class="flex-1 overflow-y-auto px-6 py-10 transition-all duration-300">
            <div class="max-w-5xl mx-auto">
                {@render children()}
            </div>
        </main>
    </div>
</div>
