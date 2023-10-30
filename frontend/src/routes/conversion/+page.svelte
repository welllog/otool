<script>
    import Label from "$lib/Label.svelte";
    import { Radio, Select, Input, Button } from "flowbite-svelte";

    let inNo = '0', outNo = '0', inDecimal = 10, outDecimal = 10;

    function transform() {
        let n = parseInt(inNo, inDecimal)
        outNo = n.toString(outDecimal)
    }

    // 数字进制
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

<div class="mb-3 mt-3 flex flex-wrap gap-3">
    {#each decimalOpts as opt}
        <Radio value={opt.value} bind:group={inDecimal}>{opt.name}</Radio>
    {/each}
    <Select size="sm" class="w-fit" items={decimals} bind:value={inDecimal} />
</div>

<div class="mb-3">
    <Label>待转换数字</Label>
    <Input bind:value={inNo} />
</div>

<div class="mb-3 flex flex-wrap gap-3">
    {#each decimalOpts as opt}
        <Radio value={opt.value} bind:group={outDecimal}>{opt.name}</Radio>
    {/each}
    <Select size="sm" class="w-fit" items={decimals} bind:value={outDecimal} />
</div>

<div class="mb-3">
    <Button on:click={transform} color="blue" outline size="xs">转换</Button>
</div>

<div>
    <Label>转换结果</Label>
    <Input bind:value={outNo} />
</div>
