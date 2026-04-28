<script>
    let {
        type = 'primary',
        message = 'This is a toast message',
    } = $props();

    const styles = {
        success: 'border-green-500/20 text-green-700 dark:text-green-300 bg-green-500/10',
        danger: 'border-red-500/20 text-red-700 dark:text-red-300 bg-red-500/10',
        warning: 'border-orange-500/20 text-orange-700 dark:text-orange-300 bg-orange-500/10',
        info: 'border-blue-500/20 text-blue-700 dark:text-blue-300 bg-blue-500/10',
        primary: 'border-primary-500/20 text-primary-700 dark:text-primary-300 bg-primary-500/10',
        normal: 'border-gray-500/20 text-gray-700 dark:text-gray-300 bg-gray-500/10',
    };

    /** @type {string} */
    let currentStyle = $derived(styles[type] || styles.primary);
</script>

<div class="flex items-center gap-4 px-5 py-4 rounded-2xl border backdrop-blur-xl shadow-2xl {currentStyle} pointer-events-auto">
    <div class="flex-shrink-0">
        {#if type === 'success'}
            <svg class="w-6 h-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        {:else if type === 'danger'}
            <svg class="w-6 h-6 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        {:else if type === 'warning'}
            <svg class="w-6 h-6 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
        {:else}
            <svg class="w-6 h-6 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        {/if}
    </div>
    <div class="flex-1 text-sm font-semibold tracking-tight">
        {message}
    </div>
</div>
