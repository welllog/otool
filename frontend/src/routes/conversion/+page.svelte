<script>
    import Label from "$lib/Label.svelte";
    import { Radio, Select, Input, Button } from "flowbite-svelte";

    let inNo = $state('0');
    let outNo = $state('0');
    let inDecimal = $state(10);
    let outDecimal = $state(10);

    function transform() {
        let n = parseInt(inNo, inDecimal)
        outNo = n.toString(outDecimal)
    }

    let decimalOpts = [
        {name: '二进制', value: 2},
        {name: '八进制', value: 8},
        {name: '十进制', value: 10},
        {name: '十六进制', value: 16},
        {name: '三十二进制', value: 32},
        {name: '三十六进制', value: 36},
    ];

    let decimals = [
        { value: 2, name: '二进制' },
        { value: 3, name: '三进制' },
        { value: 4, name: '四进制' },
        { value: 5, name: '五进制' },
        { value: 6, name: '六进制' },
        { value: 7, name: '七进制' },
        { value: 8, name: '八进制' },
        { value: 9, name: '九进制' },
        { value: 10, name: '十进制' },
        { value: 11, name: '十一进制' },
        { value: 12, name: '十二进制' },
        { value: 13, name: '十三进制' },
        { value: 14, name: '十四进制' },
        { value: 15, name: '十五进制' },
        { value: 16, name: '十六进制' },
        { value: 17, name: '十七进制' },
        { value: 18, name: '十八进制' },
        { value: 19, name: '十九进制' },
        { value: 20, name: '二十进制' },
        { value: 21, name: '二十一进制' },
        { value: 22, name: '二十二进制' },
        { value: 23, name: '二十三进制' },
        { value: 24, name: '二十四进制' },
        { value: 25, name: '二十五进制' },
        { value: 26, name: '二十六进制' },
        { value: 27, name: '二十七进制' },
        { value: 28, name: '二十八进制' },
        { value: 29, name: '二十九进制' },
        { value: 30, name: '三十进制' },
        { value: 31, name: '三十一进制' },
        { value: 32, name: '三十二进制' },
        { value: 33, name: '三十三进制' },
        { value: 34, name: '三十四进制' },
        { value: 35, name: '三十五进制' },
        { value: 36, name: '三十六进制' },
    ];
</script>

<div class="flex flex-col gap-6 max-w-4xl mx-auto py-8">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 relative">
        
        <!-- Input Side -->
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-6 flex flex-col min-h-[320px]">
            <div class="flex items-center gap-3 mb-6">
                <div class="w-10 h-10 rounded-full bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center text-blue-600 dark:text-blue-400">
                    <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                </div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">输入数据</h2>
            </div>

            <div class="mb-6 flex-1">
                <Label class="mb-4 text-gray-500 dark:text-gray-400 font-medium">原始进制</Label>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-y-3 gap-x-2 mb-4">
                    {#each decimalOpts as opt}
                        <Radio value={opt.value} bind:group={inDecimal} class="text-sm">{opt.name}</Radio>
                    {/each}
                </div>
                <div class="pt-4 border-t border-gray-100 dark:border-gray-700">
                    <Select size="sm" class="w-full bg-gray-50 dark:bg-gray-900" items={decimals} bind:value={inDecimal} placeholder="更多进制选择..."/>
                </div>
            </div>
            
            <div class="mt-auto">
                <Label class="mb-2 text-gray-500 dark:text-gray-400 font-medium">待转换数字</Label>
                <Input size="lg" bind:value={inNo} class="w-full font-mono text-xl tracking-wider text-blue-600 dark:text-blue-400 bg-blue-50/50 dark:bg-gray-900 border-blue-100 dark:border-gray-700 focus:ring-blue-500 focus:border-blue-500" placeholder="0" />
            </div>
        </div>

        <!-- Output Side -->
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-6 flex flex-col min-h-[320px]">
            <div class="flex items-center gap-3 mb-6">
                <div class="w-10 h-10 rounded-full bg-green-50 dark:bg-green-900/30 flex items-center justify-center text-green-600 dark:text-green-400">
                    <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                </div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">转换结果</h2>
            </div>

            <div class="mb-6 flex-1">
                <Label class="mb-4 text-gray-500 dark:text-gray-400 font-medium">目标进制</Label>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-y-3 gap-x-2 mb-4">
                    {#each decimalOpts as opt}
                        <Radio value={opt.value} bind:group={outDecimal} class="text-sm">{opt.name}</Radio>
                    {/each}
                </div>
                <div class="pt-4 border-t border-gray-100 dark:border-gray-700">
                    <Select size="sm" class="w-full bg-gray-50 dark:bg-gray-900" items={decimals} bind:value={outDecimal} placeholder="更多进制选择..."/>
                </div>
            </div>

            <div class="mt-auto">
                <Label class="mb-2 text-gray-500 dark:text-gray-400 font-medium">结果数据</Label>
                <Input size="lg" bind:value={outNo} readonly class="w-full font-mono text-xl tracking-wider text-green-600 dark:text-green-400 bg-green-50/50 dark:bg-gray-900 border-green-100 dark:border-gray-700" placeholder="0" />
            </div>
        </div>
    </div>

    <!-- Action Bar -->
    <div class="flex justify-center -mt-4 z-10 relative">
        <Button onclick={transform} color="primary" class="px-12 py-3 shadow-xl shadow-primary-500/20 text-white rounded-full flex items-center gap-2 transition-transform hover:scale-105 bg-gradient-to-r from-primary-600 to-primary-500 border-none">
            <svg class="w-5 h-5 animate-pulse" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            <span class="text-base font-semibold tracking-wide">立即转换</span>
        </Button>
    </div>
</div>
