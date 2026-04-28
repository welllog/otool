<script>
    import { Button, Tooltip } from 'flowbite-svelte';
    import { toast } from './ToastContainer.svelte';
    import { twMerge } from 'tailwind-merge';

    let { text = '', class: className = '', ...rest } = $props();
    let copied = $state(false);

    async function copyToClipboard() {
        if (!text) return;
        try {
            await navigator.clipboard.writeText(text);
            copied = true;
            toast('success', '已成功复制到剪贴板');
            setTimeout(() => {
                copied = false;
            }, 2000);
        } catch (err) {
            toast('danger', '复制失败');
        }
    }
</script>

<Button
    size="xs"
    color="light"
    onclick={copyToClipboard}
    class={twMerge("p-2 dark:bg-gray-800 dark:hover:bg-gray-700 border-gray-200 dark:border-gray-700", className)}
    {...rest}
>
    {#if copied}
        <svg class="w-4 h-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
    {:else}
        <svg class="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
        </svg>
    {/if}
</Button>
<Tooltip color="primary">{copied ? '已复制' : '复制到剪贴板'}</Tooltip>
