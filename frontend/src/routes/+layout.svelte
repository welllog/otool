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
    }

    let { children } = $props();

    let currentPath = $derived(page.url.pathname);

    let title = "OTOOL";
    let topMenus: menu[] = [];
    let sideMenus: menu[] = [
        { title: "文本编解码", href: "/text" },
        { title: "文件编解码", href: "/file" },
        { title: "进制转换", href: "/conversion" },
        { title: "图片转换", href: "/image" },
        { title: "二维码工具", href: "/qrcode" },
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
    let outsideclose = $state(true);

    $effect(() => {
        if (width >= breakPoint) {
            drawerOpen = true;
            outsideclose = false;
        } else {
            drawerOpen = false;
            outsideclose = true;
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
            drawerOpen = !drawerOpen;
        }
    };
    const toggleDrawer = () => {
        drawerOpen = true;
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

<svelte:window bind:innerWidth={width} bind:innerHeight={height}/>
<div class="fixed top-14 right-4 z-30 h-3/4 overflow-scroll">
    <ToastContainer/>
</div>
<header style="--wails-draggable:drag">
    <Navbar class="px-0 py-0 w-full z-10 border-b bg-white dark:bg-gray-800">
        <span class="flex items-center lg:ml-64 ml-14">
            <button onclick={toggleDrawer} class="dark:fill-white mr-4 lg:hidden" aria-label="Toggle menu">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24"><path
                        d="M21 17.9995V19.9995H3V17.9995H21ZM17.4038 3.90332L22 8.49951L17.4038 13.0957L15.9896 11.6815L19.1716 8.49951L15.9896 5.31753L17.4038 3.90332ZM12 10.9995V12.9995H3V10.9995H12ZM12 3.99951V5.99951H3V3.99951H12Z"></path></svg>
            </button>
            <NavBrand href="/">
                <span class="ml-0 whitespace-nowrap text-xl font-semibold dark:text-white">
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

<Drawer
        modal={false}
        {transitionParams}
        bind:open={drawerOpen}
        {outsideclose}
        class="w-64 overflow-y-auto z-20 p-4 bg-white dark:bg-gray-800 border-r dark:border-gray-800"
        id="sidebar"
>
    <div class="flex items-center" style="--wails-draggable:drag">
        <CloseButton onclick={() => (drawerOpen = false)} class="dark:text-white lg:invisible"/>
    </div>
    <Sidebar
            class="w-54"
            activeUrl={currentPath}
            activeClass="flex items-center p-2 text-base font-normal text-primary-900 bg-primary-200 dark:bg-primary-700 rounded-lg dark:text-white hover:bg-primary-100 dark:hover:bg-gray-700"
            nonActiveClass="flex items-center p-2 text-base font-normal text-green-900 rounded-lg dark:text-white hover:bg-green-100 dark:hover:bg-green-700"
    >
        <SidebarWrapper class="py-4 px-3 rounded dark:bg-gray-800">
            <SidebarGroup>
                {#each sideMenus as sm}
                    <SidebarItem label={sm.title} href={sm.href} onclick={toggleSide}/>
                {/each}
            </SidebarGroup>
        </SidebarWrapper>
    </Sidebar>
</Drawer>

<div class="flex px-4 mx-auto bg-white dark:bg-slate-900 overflow-scroll" style="height: {height - 44}px">
    <main class="lg:ml-72 w-full mx-auto">
        {@render children()}
    </main>
</div>
