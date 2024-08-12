<script lang="ts">
    import "../app.postcss";
    import {page} from '$app/stores';
    import {onMount, onDestroy} from 'svelte';
    import {sineIn} from 'svelte/easing';
    import DarkMode from '$lib/DarkMode.svelte';
    import {
        Navbar, NavBrand, NavHamburger, NavUl, NavLi,
        Drawer, Sidebar, SidebarGroup, SidebarItem, SidebarWrapper, CloseButton,
    } from "flowbite-svelte";
    import ToastContainer, {toast} from '$lib/ToastContainer.svelte';
    import {EventsOn} from 'wjs/runtime/runtime';

    interface menu {
        title: string;
        href: string;
    }

    let currentPath: string;
    $: currentPath = $page.url.pathname;

    let title = "OTOOL";
    let topMenus: menu[] = [];
    let sideMenus: menu[] = [
        {title: "文本编解码", href: "/text"},
        {title: "文件编解码", href: "/file"},
        {title: "进制转换", href: "/conversion"},
        {title: "图片转换", href: "/image"},
        {title: "二维码工具", href: "/qrcode"},
    ];

    let notifyClose = () => {};
    let height: number;
    let transitionParams = {
        x: -320,
        duration: 200,
        easing: sineIn
    };
    let breakPoint: number = 1024;
    let width: number;
    let activateClickOutside = true;
    let drawerHidden: boolean = false;
    $: if (width >= breakPoint) {
        drawerHidden = false;
        activateClickOutside = false;
    } else {
        drawerHidden = true;
        activateClickOutside = true;
    }
    onMount(() => {
        notifyClose = EventsOn('notify', (event: any) => {
            toast(event.type, event.info);
        });

        if (width >= breakPoint) {
            drawerHidden = false;
            activateClickOutside = false;
        } else {
            drawerHidden = true;
            activateClickOutside = true;
        }
    });
    onDestroy(() => {
        notifyClose();
    });

    const toggleSide = () => {
        if (width < breakPoint) {
            drawerHidden = !drawerHidden;
        }
    };
    const toggleDrawer = () => {
        drawerHidden = false;
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
    <Navbar class="px-0 py-0 w-full z-10 border-b">
        <span class="flex items-center lg:ml-64 ml-14">
            <button on:click={toggleDrawer} class="dark:fill-white mr-4 lg:hidden">
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
                activeClass="text-primary-700 dark:text-primary-500"
                nonActiveClass="text-gray-700 hover:text-primary-700 dark:text-white dark:hover:text-primary-700"
                activeUrl={currentPath}
        >
            {#each topMenus as item (item.href)}
                <NavLi href={item.href}>{item.title}</NavLi>
            {/each}
        </NavUl>
        <DarkMode/>
        {#if topMenus.length > 0}
            <NavHamburger/>
        {/if}
    </Navbar>
</header>

<Drawer
        backdrop={false}
        {transitionParams}
        bind:hidden={drawerHidden}
        bind:activateClickOutside
        width="w-64"
        divClass="overflow-y-auto z-20 p-4 bg-white dark:bg-gray-800 border-r dark:border-gray-800"
        id="sidebar"
>
    <div class="flex items-center" style="--wails-draggable:drag">
        <CloseButton on:click={() => (drawerHidden = true)} class="dark:text-white lg:invisible"/>
    </div>
    <Sidebar
            asideClass="w-54"
            activeUrl={currentPath}
            activeClass="flex items-center p-2 text-base font-normal text-primary-900 bg-primary-200 dark:bg-primary-700 rounded-lg dark:text-white hover:bg-primary-100 dark:hover:bg-gray-700"
            nonActiveClass="flex items-center p-2 text-base font-normal text-green-900 rounded-lg dark:text-white hover:bg-green-100 dark:hover:bg-green-700"
    >
        <SidebarWrapper divClass="py-4 px-3 rounded dark:bg-gray-800">
            <SidebarGroup>
                {#each sideMenus as sm}
                    <SidebarItem label={sm.title} href={sm.href} on:click={toggleSide}/>
                {/each}
            </SidebarGroup>
        </SidebarWrapper>
    </Sidebar>
</Drawer>

<div class="flex px-4 mx-auto bg-white dark:bg-slate-900 overflow-scroll" style="height: {height - 44}px">
    <main class="lg:ml-72 w-full mx-auto">
        <slot/>
    </main>
</div>