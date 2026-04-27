<script lang="ts" module>
    import Toast from './Toast.svelte';

    let defaultDuration = 2000;

    interface toastMeta {
        symbol: symbol;
        type: string;
        message: string;
        duration: number;
    }

    let meta: toastMeta[] = $state([]);

    export function toast(type: string, message: string, duration = 0) {
        let symbol = Symbol();
        meta.unshift({ symbol, type, message, duration })

        if (duration === 0) {
            duration = defaultDuration;
        }

        if (duration > 0) {
            setTimeout(() => {
                closeToast(symbol);
            }, duration);
        }
    }

    function closeToast(symbol: symbol) {
        meta = meta.filter(t => t.symbol !== symbol);
    }
</script>

<script>
    import { fly, fade } from 'svelte/transition';
</script>

{#each meta as t (t.symbol)}
    <div in:fly={{ x: 200, duration: 1000 }} out:fade class="m-1">
        <Toast type={t.type} message={t.message} />
    </div>
{/each}
