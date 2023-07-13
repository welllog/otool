<script>
    import * as image from "$wailsjs/go/srvs/Image"
    import * as app from "$wailsjs/go/internal/App";
    import Alert, {showAlert, closeAlert} from "../Alert.svelte";
    import * as rt from "$wailsjs/runtime"

    let pathName = '', savePath = '', saveName = '', noSuffixName = '';
    let img = {};
    let disabled = false, loading = false;
    let width = 0, height = 0, percent = 100;
    let encoderOption = 0;
    let encoderOptionTitle = '', encoderOptionMax , encoderOptionMin;

    let op = 0;
    let ops = [
        {
            name: '原始尺寸',
            value: 0,
        },
        {
            name: '固定宽度缩放',
            value: 1,
        },
        {
            name: '最大宽度缩放',
            value: 2,
        },
        {
            name: '固定宽高裁剪',
            value: 3,
        },
        {
            name: '最大宽高缩放',
            value: 4,
        },
        {
            name: '固定高度缩放',
            value: 5,
        },
        {
            name: '最大高度缩放',
            value: 6,
        },
        {
            name: '百分比缩放',
            value: 7,
        }
    ]
    let encoder = 'png'
    let encoders = ['jpg', 'png', 'gif', 'bmp', 'tiff'];

    let pngCompress = [
        {
            name: '默认',
            value: 0,
        },
        {
            name: '不压缩',
            value: -1,
        },
        {
            name: '速度优先',
            value: -2,
        },
        {
            name: '压缩优先',
            value: -3,
        }
    ]

    async function openFile() {
        closeAlert();
        disabled = true;

        try {
            pathName = await image.OpenFileDialog();
            if (pathName.length === 0) {
                return;
            }

            loading = true;
            img = await image.Decode(pathName);
            width = img.Width
            height = img.Height
            percent = 100
            savePath = img.Path
            noSuffixName = img.NoSuffixName

            changeEncoderOption(encoder)
        } catch (e) {
            showAlert("danger", e.toString());
        } finally {
            disabled = false;
            loading = false;
        }
    }

    function reset() {
        width = img.Width;
        height = img.Height;
        percent = 100;
    }

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            savePath = path;

        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            disabled = false;
        }
    }

    function transform() {
        disabled = true;
        loading = true
        image.Crop(op, width, height, percent, parseInt(encoderOption, 10), savePath, saveName)
            .then(() => {
                showAlert("success", "转换成功");
            })
            .catch((err) => {
                showAlert("danger", err.toString());
            })
            .finally(() => {
                disabled = false;
                loading = false;
            })
    }

    function changeEncoderOption(enc) {
        switch (enc) {
            case 'jpg':
                encoderOption = 95
                encoderOptionTitle = '质量'
                encoderOptionMax = 100
                encoderOptionMin = 1
                break
            case 'gif':
                encoderOption = 256
                encoderOptionTitle = '色彩数量'
                encoderOptionMax = 256
                encoderOptionMin = 1
                break
            case 'png':
                encoderOption = 0
                encoderOptionTitle = '压缩等级'
                break
        }
    }

    $: saveName = noSuffixName + '-' + op + '-' + width + '-' + height + '-' + percent + '.' + encoder
</script>

<div class="container-fluid">
    <Alert/>

    <div class="mt-2">
        <button class="btn btn-outline-secondary" on:click={openFile} class:disabled={disabled}>
            {#if loading}
            <span
                class="spinner-border spinner-border-sm"
                role="status"
                aria-hidden="true"
            />
            {/if}
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-plus-lg" viewBox="0 0 16 16">
                <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
            </svg>
            选择图片
        </button>
    </div>

    {#if Object.keys(img).length > 0}
    <div class="card mt-2" style="width: {img.ThumbWidth}px; min-width: 30%; mex-width: 100%;">
        <img src="data:image/png;base64,{img.Thumbnail}" class="card-img-top border" style="max-width: {img.ThumbWidth}px;" alt="...">
        <div class="card-body">
            <h5 class="card-title">{img.Name} {img.Size}</h5>
            <p class="card-text">
                format: {img.Format}&nbsp;&nbsp; wh: {img.Width} * {img.Height}
<!--                thumb size: {img.ThumbWidth} * {img.ThumbHeight} <br/>-->
            </p>
        </div>
    </div>

    <div class="col-md-auto mt-2 input-group input-group-sm">
        <span class="input-group-text">缩略方式</span>
        <select bind:value={op} id="op" class="form-select">
            {#each ops as d}
                <option value={d.value}>{d.name}</option>
            {/each}
        </select>
    </div>

    {#if op > 0 && op < 7}
    <div class="input-group mt-2 input-group-sm">
        {#if op < 5}
        <span class="input-group-text">宽</span>
        <input bind:value={width} type="number" class="form-control">
        {/if}
        {#if op > 2}
        <span class="input-group-text">高</span>
        <input bind:value={height} type="number" class="form-control">
        {/if}
    </div>
    {/if}

    {#if op === 7}
    <div class="input-group mt-2 input-group-sm">
        <span class="input-group-text">缩放百分比</span>
        <input bind:value={percent} type="number" class="form-control">
        <span class="input-group-text">%</span>
    </div>
    {/if}

    <div class="col-md-auto mt-2 input-group input-group-sm">
        <span class="input-group-text">图片保存格式</span>
        <select bind:value={encoder} on:change={(e) => {changeEncoderOption(e.target.value)}} id="encoder" class="form-select">
            {#each encoders as e}
                <option value={e}>{e}</option>
            {/each}
        </select>
    </div>

    {#if ['jpg', 'png', 'gif'].indexOf(encoder) >= 0 }
        {#if ['jpg', 'gif'].indexOf(encoder) >= 0 }
        <div class="row mt-2">
            <label for="optionRange" class="col-form-label col-auto">{encoderOptionTitle}</label>
            <div class="col-auto">
                <input bind:value={encoderOption} type="range" class="form-range form-range-sm" min="{encoderOptionMin}" max="{encoderOptionMax}" id="optionRange">
            </div>
            <div class="col-auto">
                <input bind:value={encoderOption} type="number" max="{encoderOptionMax}" min="{encoderOptionMin}" step="1" class="form-control form-control-sm">
            </div>
        </div>
        {:else if encoder === 'png'}
            <div class="input-group mt-2 input-group-sm">
            <span class="input-group-text">{encoderOptionTitle}</span>
            &nbsp;&nbsp;&nbsp;&nbsp;
            {#each pngCompress as cp}
                <div class="form-check form-check-inline">
                    <input
                        bind:group={encoderOption}
                        class="form-check-input"
                        type="radio"
                        name="opt"
                        id={cp.value}
                        value={cp.value}
                    />
                    <label class="form-check-label" for={cp.value}>{cp.name}</label>
                </div>
            {/each}
            </div>
        {/if}
    {/if}

    <div class="mt-2 input-group input-group-sm">
        <button
            on:click={openFolder}
            class:disabled={disabled}
            type="button"
            class="btn btn-outline-secondary btn-sm"
        >
        {#if loading}
        <span
            class="spinner-border spinner-border-sm"
            role="status"
            aria-hidden="true"
        />
        {/if}
            选择保存目录
        </button>
        <input type="text" bind:value={savePath} disabled="true"  class="form-control">
    </div>
    <div class="mt-2 input-group input-group-sm">
        <span class="input-group-text">保存文件名</span>
        <textarea bind:value={saveName} class:disabled={disabled}  class="form-control"></textarea>
    </div>
    {/if}


    <div class="mt-2">
        <button
            on:click={transform}
            type="button"
            class="btn btn-outline-primary btn-sm"
            class:disabled={!(!disabled && Object.keys(img).length > 0)}
        >
        转换
        </button>
        <button
            on:click={reset}
            class:disabled={!(!disabled && Object.keys(img).length > 0)}
            type="button"
            class="btn btn-outline-warning btn-sm">
            恢复默认宽高
        </button>
    </div>
</div>