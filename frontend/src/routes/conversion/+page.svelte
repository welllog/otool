<script>
    let inNo = 0, outNo = 0, inDecimal = 10, outDecimal = 10;

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

    let decimals = [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
        22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36];

</script>

<div class="container-fluid">
    <div class="row mt-3 mb-3">
        <div class="col-auto">
            {#each decimalOpts as opt}
                <div class="form-check form-check-inline">
                    <input bind:group={inDecimal} class="form-check-input" type="radio" name="inDecimal" id="in{opt.value}" value={opt.value}>
                    <label class="form-check-label" for="in{opt.value}">{opt.name}</label>
                </div>
            {/each}
        </div>
        <div class="col-auto">
            <select bind:value={inDecimal} id="inDecimal" class="form-select form-select-sm">
                {#each decimals as d}
                    <option value={d}>{d}</option>
                {/each}
            </select>
        </div>
    </div>
    <div class="mb-3 form-floating">
        <input bind:value={inNo} type="text" class="form-control" id="inNo">
        <label for="inNo" class="form-label">待转换数字</label>
    </div>
    <div class="row mb-3">
        <div class="col-auto">
            {#each decimalOpts as opt}
                <div class="form-check form-check-inline">
                    <input bind:group={outDecimal} class="form-check-input" type="radio" name="outDecimal" id="out{opt.value}" value={opt.value}>
                    <label class="form-check-label" for="out{opt.value}">{opt.name}</label>
                </div>
            {/each}
        </div>
        <div class="col-auto">
            <select bind:value={outDecimal} id="outDecimal" class="form-select form-select-sm">
                {#each decimals as d}
                    <option value={d}>{d}</option>
                {/each}
            </select>
        </div>
    </div>
    <button on:click={transform} type="button" class="btn btn-outline-primary btn-sm mb-3">转换</button>
    <div class="form-floating">
        <input bind:value={outNo} type="text" class="form-control" id="outNo">
        <label for="outNo" class="form-label">转换结果</label>
    </div>
</div>
