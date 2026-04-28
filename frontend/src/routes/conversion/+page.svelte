<script>
    import Label from "$lib/Label.svelte";
    import { Radio, Select, Input } from "flowbite-svelte";
    import Card from "$lib/Card.svelte";
    import CopyButton from "$lib/CopyButton.svelte";
    import { toast } from "$lib/ToastContainer.svelte";

    let inNo = $state('0');
    let inDecimal = $state(10);
    let outDecimal = $state(16);

    // Real-time reactive conversion
    let outNo = $derived.by(() => {
        if (!inNo) return '0';
        try {
            let n = parseInt(inNo, inDecimal);
            if (isNaN(n)) return 'Error';
            return n.toString(outDecimal).toUpperCase();
        } catch (e) {
            return 'Error';
        }
    });

    let decimalOpts = [
        {name: '二进制', value: 2},
        {name: '八进制', value: 8},
        {name: '十进制', value: 10},
        {name: '十六进制', value: 16},
        {name: '32进制', value: 32},
        {name: '36进制', value: 36},
    ];

    let decimals = Array.from({ length: 35 }, (_, i) => ({ value: i + 2, name: `${i + 2}进制` }));
</script>

<div class="flex flex-col gap-10 max-w-5xl mx-auto py-10">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-10 items-stretch">
        <!-- Input Side -->
        <Card 
            title="输入源" 
            icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>`}
            class="flex flex-col ring-1 ring-black/5 dark:ring-white/5 shadow-2xl"
            delay={0}
        >
            <div class="space-y-8 flex-1">
                <div>
                    <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 mb-5">原始进制 (Source Base)</Label>
                    <div class="grid grid-cols-3 gap-3 mb-5">
                        {#each decimalOpts as opt}
                            <button 
                                class="flex items-center justify-center p-3 rounded-xl border-2 transition-all font-bold text-xs
                                {inDecimal === opt.value 
                                    ? 'bg-primary-600 border-primary-600 text-white shadow-lg shadow-primary-500/30' 
                                    : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                                onclick={() => inDecimal = opt.value}
                            >
                                {opt.name}
                            </button>
                        {/each}
                    </div>
                    <Select size="sm" class="w-full rounded-xl bg-white dark:bg-gray-800 border-gray-100 dark:border-gray-700 font-bold" items={decimals} bind:value={inDecimal} placeholder="选择更多定制进制..."/>
                </div>
                
                <div class="pt-8 border-t border-gray-100 dark:border-gray-800/50">
                    <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 mb-4">待转换数值 (Input)</Label>
                    <div class="relative group">
                        <Input size="lg" bind:value={inNo} class="w-full font-mono text-3xl font-black bg-primary-50/10 dark:bg-primary-900/10 border-2 border-primary-100 dark:border-primary-900/30 text-primary-600 dark:text-primary-400 rounded-2xl py-6 px-6 focus:ring-4 focus:ring-primary-500/10" placeholder="0" />
                        <div class="absolute right-4 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
                            <span class="text-[10px] font-bold text-primary-500 uppercase">编辑中...</span>
                        </div>
                    </div>
                </div>
            </div>
        </Card>

        <!-- Output Side -->
        <Card 
            title="目标结果" 
            icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>`}
            class="flex flex-col ring-1 ring-black/5 dark:ring-white/5 shadow-2xl"
            delay={150}
        >
            {#snippet extra()}
                {#if outNo && outNo !== 'Error'}
                    <CopyButton text={outNo} />
                {/if}
            {/snippet}

            <div class="space-y-8 flex-1">
                <div>
                    <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 mb-5">目标进制 (Target Base)</Label>
                    <div class="grid grid-cols-3 gap-3 mb-5">
                        {#each decimalOpts as opt}
                            <button 
                                class="flex items-center justify-center p-3 rounded-xl border-2 transition-all font-bold text-xs
                                {outDecimal === opt.value 
                                    ? 'bg-emerald-600 border-emerald-600 text-white shadow-lg shadow-emerald-500/30' 
                                    : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                                onclick={() => outDecimal = opt.value}
                            >
                                {opt.name}
                            </button>
                        {/each}
                    </div>
                    <Select size="sm" class="w-full rounded-xl bg-white dark:bg-gray-800 border-gray-100 dark:border-gray-700 font-bold" items={decimals} bind:value={outDecimal} placeholder="选择更多定制进制..."/>
                </div>

                <div class="pt-8 border-t border-gray-100 dark:border-gray-800/50">
                    <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 mb-4">转换结果 (Output)</Label>
                    <div class="relative">
                        <Input size="lg" bind:value={outNo} readonly class="w-full font-mono text-3xl font-black bg-emerald-50/10 dark:bg-emerald-900/10 border-2 border-emerald-100/50 dark:border-emerald-900/30 text-emerald-600 dark:text-emerald-400 rounded-2xl py-6 px-6" placeholder="0" />
                        {#if outNo === 'Error'}
                            <div class="absolute inset-0 bg-red-500/10 rounded-2xl flex items-center justify-center border-2 border-red-500/30 backdrop-blur-sm">
                                <span class="text-red-600 font-black text-sm uppercase tracking-widest">无效输入</span>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
        </Card>
    </div>

    <!-- Info Tip -->
    <div class="flex justify-center" in:fade={{ delay: 500 }}>
        <p class="text-xs text-gray-400 font-medium flex items-center gap-2">
            <svg class="w-4 h-4 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            支持任意 2 到 36 进制间的实时转换，输入自动解析并同步。
        </p>
    </div>
</div>
