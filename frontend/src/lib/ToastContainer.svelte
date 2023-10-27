<script lang="ts" context="module">
    import { writable } from 'svelte/store';
    import { fade, fly } from 'svelte/transition';
    import Toast from './Toast.svelte';

    interface toastMeta {
        symbol: symbol;
        type: string;
        message: string;
        duration: number;
    }

    let meta: toastMeta[] = [];
    const toasts = writable(meta);
    export function toast(type: string, message: string, duration = 0) {
        let symbol = Symbol();
        toasts.update(t => {
            t.unshift({ symbol, type, message, duration })
            return t;
        })

        if (duration > 0) {
            setTimeout(() => {
                closeToast(symbol);
            }, duration);
        }
    }

    function closeToast(symbol: Symbol) {
        toasts.update(t => t.filter(toast => toast.symbol !== symbol));
    }

</script>

{#each $toasts as toast (toast.symbol)}
    <div in:fly={{ x: 200, duration: 1000 }} out:fade class="m-1">
        <Toast type={toast.type} message={toast.message} on:close={() => {closeToast(toast.symbol)}} />
    </div>
{/each}