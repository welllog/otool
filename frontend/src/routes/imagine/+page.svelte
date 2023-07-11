<script>
    import * as image from "$wailsjs/go/srvs/Image"
    import Alert, {showAlert, closeAlert} from "../Alert.svelte";
    import * as rt from "$wailsjs/runtime"

    let pathName = '';
    let img = {};
    let disabled = false;
    let width = 0, height = 0, percent = 100;

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
    let encoders = ['jpg', 'png', 'gif', 'webp', 'bmp', 'tiff'];

    async function openFile() {
        closeAlert();
        img = {};
        disabled = true;

        try {
            pathName = await image.OpenFileDialog();
            if (pathName.length === 0) {
                throw new Error("Please select a picture");
            }

            img = await image.Decode(pathName);
            width = img.Width
            height = img.Height
            percent = 100
        } catch (e) {
            showAlert("danger", e.toString());
        } finally {
            disabled = false;
        }
    }
</script>

<div class="m-3">
    <Alert/>

    <button class="btn btn-outline-secondary" on:click={openFile} class:disabled={disabled}>
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-plus-lg" viewBox="0 0 16 16">
            <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
        </svg>
        选择图片
    </button>

    {#if Object.keys(img).length > 0}
    <div class="card mt-3" style="width: {img.ThumbWidth}px">
        <img src="data:image/png;base64,{img.Thumbnail}" width={img.ThumbWidth} class="card-img-top" alt="...">
        <div class="card-body">
            <h5 class="card-title">{img.Name} {img.Size}</h5>
            <p class="card-text">
                format: {img.Format} <br/>
                wh: {img.Width} * {img.Height} <br/>
<!--                thumb size: {img.ThumbWidth} * {img.ThumbHeight} <br/>-->
            </p>
        </div>
    </div>

    <div class="col-md-auto mt-3 input-group">
        <span class="input-group-text">缩略方式</span>
        <select bind:value={op} id="op" class="form-select form-select-sm">
            {#each ops as d}
                <option value={d.value}>{d.name}</option>
            {/each}
        </select>
    </div>

    {#if op > 0 && op < 7}
    <div class="input-group mt-3">
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
    <div class="input-group mt-3">
        <span class="input-group-text">缩放百分比</span>
        <input type="number" class="form-control">
        <span class="input-group-text">%</span>
    </div>
    {/if}


    <div class="col-md-auto mt-3 input-group">
        <span class="input-group-text">图片保存格式</span>
        <select bind:value={encoder} id="encoder" class="form-select form-select-sm">
            {#each encoders as e}
                <option value={e}>{e}</option>
            {/each}
        </select>
    </div>
    {/if}

    <br/>
    <button
        type="button"
        class="btn btn-outline-primary btn-sm mt-3"
        class:disabled={!(!disabled && Object.keys(img).length > 0)}
    >
    转换
    </button>
</div>