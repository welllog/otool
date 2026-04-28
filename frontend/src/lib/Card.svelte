<script>
    import { twMerge } from 'tailwind-merge';
    import { fly } from 'svelte/transition';

    let {
        children,
        title = '',
        icon = '',
        extra = null,
        class: className = '',
        animate = true,
        delay = 0,
        ...rest
    } = $props();

    const baseClass = "relative bg-white/70 dark:bg-gray-900/40 backdrop-blur-xl rounded-[24px] shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:shadow-none border border-white/40 dark:border-white/5 p-6 transition-all duration-500 hover:shadow-[0_20px_50px_rgba(0,0,0,0.1)] dark:hover:shadow-[0_20px_50px_rgba(0,0,0,0.3)] hover:-translate-y-1 hover:border-primary-500/30 group";
</script>

{#if animate}
    <div 
        class={twMerge(baseClass, className)}
        in:fly={{ y: 30, duration: 800, delay: delay, opacity: 0 }}
        {...rest}
    >
        <!-- Premium Border Glow Overlay -->
        <div class="absolute inset-0 rounded-[24px] pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity duration-500 bg-gradient-to-br from-primary-500/10 to-transparent"></div>

        {#if title || icon || extra}
            <div class="flex items-center justify-between mb-6 relative z-10">
                <div class="flex items-center gap-4">
                    {#if icon}
                        <div class="p-2.5 bg-gradient-to-br from-primary-50 to-primary-100/50 dark:from-primary-900/30 dark:to-primary-800/10 text-primary-600 dark:text-primary-400 rounded-xl shadow-sm">
                            {@html icon}
                        </div>
                    {/if}
                    {#if title}
                        <h2 class="text-xl font-extrabold text-gray-900 dark:text-white tracking-tight leading-none pt-0.5">{title}</h2>
                    {/if}
                </div>
                {#if extra}
                    <div class="flex items-center gap-2">
                        {@render extra()}
                    </div>
                {/if}
            </div>
        {/if}
        
        <div class="relative z-10">
            {@render children()}
        </div>
    </div>
{:else}
    <div 
        class={twMerge(baseClass, className)}
        {...rest}
    >
        <!-- Premium Border Glow Overlay -->
        <div class="absolute inset-0 rounded-[24px] pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity duration-500 bg-gradient-to-br from-primary-500/10 to-transparent"></div>

        {#if title || icon || extra}
            <div class="flex items-center justify-between mb-6 relative z-10">
                <div class="flex items-center gap-4">
                    {#if icon}
                        <div class="p-2.5 bg-gradient-to-br from-primary-50 to-primary-100/50 dark:from-primary-900/30 dark:to-primary-800/10 text-primary-600 dark:text-primary-400 rounded-xl shadow-sm">
                            {@html icon}
                        </div>
                    {/if}
                    {#if title}
                        <h2 class="text-xl font-extrabold text-gray-900 dark:text-white tracking-tight leading-none pt-0.5">{title}</h2>
                    {/if}
                </div>
                {#if extra}
                    <div class="flex items-center gap-2">
                        {@render extra()}
                    </div>
                {/if}
            </div>
        {/if}
        
        <div class="relative z-10">
            {@render children()}
        </div>
    </div>
{/if}
