<script>
    import * as image from "wjs/go/srvs/Image"
    import * as app from "wjs/go/internal/App";
    import { toast } from "$lib/ToastContainer.svelte";
    import { onDestroy } from 'svelte';
    import { srvs } from "wjs/go/models"
    import { Dropzone } from 'flowbite-svelte';

    let img = new srvs.ImageInfo();
    let imgOpts = new srvs.ImageOptions();
    imgOpts.op = 0;

    let disabled = false, loading = false;
    let encoderOptionTitle = '', encoderOptionMax , encoderOptionMin;

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
    let encoders = ['jpg', 'png', 'gif', 'bmp', 'tiff', 'webp'];

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

    let frameOps = [
        {
            name: "不处理",
            value: 0,
        },
        {
            name: "抽取1/2",
            value: 2,
        },
        {
            name: "抽取1/3",
            value: 3,
        },
        {
            name: "抽取1/4",
            value: 4,
        },
        {
            name: "抽取1/5",
            value: 5,
        },
        {
            name: "抽取1/6",
            value: 6,
        },
        {
            name: "抽取1/7",
            value: 7,
        },
        {
            name: "抽取1/8",
            value: 8,
        },
    ]

    async function openFile() {
        disabled = true;

        try {
            let pathName = await image.OpenFileDialog();
            if (pathName.length === 0) {
                return;
            }

            loading = true;
            img = await image.Decode(pathName);
            imgOpts.width = img.width
            imgOpts.height = img.height
            imgOpts.percent = 100
            imgOpts.savePath = img.path

            changeEncoderOption(encoder)
        } catch (e) {
            toast("danger", e.toString())
        } finally {
            disabled = false;
            loading = false;
        }
    }

    function reset() {
        imgOpts.width = img.width;
        imgOpts.height = img.height;
        imgOpts.percent = 100;

        outputName()
    }

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            imgOpts.savePath = path;

        } catch (err) {
            toast("danger", err.toString())
        } finally {
            disabled = false;
        }
    }

    function transform() {
        disabled = true;
        loading = true

        image.CropAndSave(imgOpts).then(() => {
            toast("success", "转换成功");
        })
        .catch((err) => {
            toast("danger", err.toString())
        })
        .finally(() => {
            disabled = false;
            loading = false;
        })
    }

    function changeEncoderOption(enc) {
        switch (enc) {
            case 'jpg':
                imgOpts.jpgQuality = 95

                encoderOptionTitle = '质量'
                encoderOptionMax = 100
                encoderOptionMin = 1
                break
            case 'gif':
                imgOpts.gifNumColors = 256
                imgOpts.gifDrawOnBefore = false
                imgOpts.gifDropRate = 0

                encoderOptionTitle = '色彩数量'
                encoderOptionMax = 256
                encoderOptionMin = 1
                break
            case 'png':
                imgOpts.pngCompression = 0

                encoderOptionTitle = '压缩等级'
                break
            case 'webp':
                imgOpts.webpQuality = 90
                imgOpts.webpLossless = false
                imgOpts.webpRgbInTransparent = false

                encoderOptionTitle = '质量'
                encoderOptionMax = 100
                encoderOptionMin = 1
        }

        imgOpts.saveName = img.noSuffixName + '-' + imgOpts.width + '-' + imgOpts.height + '-' + imgOpts.percent + '.' + enc
    }

    function clean() {
        img = {}
        // image.Clean()
    }

    onDestroy(() => {
        clean()
    })

    function outputName() {
        imgOpts.saveName = img.noSuffixName + '-' + imgOpts.width + '-' + imgOpts.height + '-' + imgOpts.percent + '.' + encoder
    }

    let value = [];
    const dropHandle = (event) => {
        value = [];
        event.preventDefault();
        if (event.dataTransfer.items) {
            [...event.dataTransfer.items].forEach((item, i) => {
                if (item.kind === 'file') {
                    const file = item.getAsFile();
                    value.push(file.name);
                    value = value;
                }
            });
        } else {
            [...event.dataTransfer.files].forEach((file, i) => {
                value = file.name;
            });
        }
    };

    const handleChange = (event) => {
        const files = event.target.files;
        if (files.length > 0) {
            value.push(files[0].name);
            value = value;
        }
    };

    const showFiles = (files) => {
        if (files.length === 1) return files[0];
        let concat = '';
        files.map((file) => {
            concat += file;
            concat += ',';
            concat += ' ';
        });

        if (concat.length > 40) concat = concat.slice(0, 40);
        concat += '...';
        return concat;
    };
</script>

<Dropzone
        class="mt-3"
        id="dropzone"
        on:drop={dropHandle}
        on:dragover={(event) => {
    event.preventDefault();
  }}
        on:change={handleChange}>
    <svg aria-hidden="true" class="mb-3 w-10 h-10 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
    {#if value.length === 0}
        <p class="mb-2 text-sm text-gray-500 dark:text-gray-400"><span class="font-semibold">Click to upload</span> or drag and drop</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">SVG, PNG, JPG or GIF (MAX. 800x400px)</p>
    {:else}
        <p>{showFiles(value)}</p>
    {/if}
</Dropzone>

<div class="container-fluid">
    {#if img.name === undefined}
        <div class="mt-2">
            <button class="btn btn-outline-secondary" on:click={openFile} class:disabled={disabled}>
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-plus-lg" viewBox="0 0 16 16">
                    <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
                </svg>
                选择图片
            </button>
        </div>
    {/if}

    {#if img.name !== undefined}
    <div class="card mt-2 position-relative" style="width: {img.thumbWidth}px; min-width: 30%; mex-width: 100%;">
        <div on:click={clean} class="position-absolute top-0 end-0 bg-secondary opacity-75 text-warning">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-trash3" viewBox="0 0 16 16">
                <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5ZM11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H2.506a.58.58 0 0 0-.01 0H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84l.853-10.66h.538a.5.5 0 0 0 0-1h-.995a.59.59 0 0 0-.01 0H11Zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5h9.916Zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5.03a.5.5 0 0 1 .47-.53Zm5.058 0a.5.5 0 0 1 .47.53l-.5 8.5a.5.5 0 1 1-.998-.06l.5-8.5a.5.5 0 0 1 .528-.47ZM8 4.5a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5Z"/>
            </svg>
        </div>
        <img src="data:image/png;base64,{img.thumbnail}" class="card-img-top border" style="max-width: {img.thumbWidth}px;" alt="...">
        <div class="card-body">
            <h5 class="card-title">{img.name} {img.size}</h5>
            <p class="card-text">
                format: {img.format}&nbsp;&nbsp; wh: {img.width} * {img.height}
                {#if img.format === 'GIF'}<br/> frames: {img.frames}{/if}
<!--                thumb size: {img.thumbWidth} * {img.thumbHeight} <br/>-->
            </p>
        </div>
    </div>

    <div class="col-md-auto mt-2 input-group input-group-sm">
        <span class="input-group-text">缩略方式</span>
        <select bind:value={imgOpts.op} id="op" class="form-select" disabled={disabled}>
            {#each ops as d}
                <option value={d.value}>{d.name}</option>
            {/each}
        </select>
    </div>

    {#if imgOpts.op > 0 && imgOpts.op < 7}
        <div class="input-group mt-2 input-group-sm">
            {#if imgOpts.op < 5}
            <span class="input-group-text">宽</span>
            <input bind:value={imgOpts.width} on:change={outputName} type="number" class="form-control" disabled={disabled}>
            <span class="input-group-text">px</span>
            {/if}
            {#if imgOpts.op > 2}
            <span class="input-group-text">高</span>
            <input bind:value={imgOpts.height} on:change={outputName} type="number" class="form-control" disabled={disabled}>
            <span class="input-group-text">px</span>
            {/if}
        </div>
    {/if}

    {#if imgOpts.op === 7}
        <div class="input-group mt-2 input-group-sm">
            <span class="input-group-text">缩放百分比</span>
            <input bind:value={imgOpts.percent} on:change={outputName} type="number" class="form-control" disabled={disabled}>
            <span class="input-group-text">%</span>
        </div>
    {/if}

    <div class="mt-2 d-flex flex-wrap">
        保存格式：&nbsp;
        {#each encoders as e}
            <div class="form-check form-check-inline">
                <input
                    on:change={(event) => {changeEncoderOption(event.target.value)}}
                    bind:group={encoder}
                    class="form-check-input"
                    type="radio"
                    id={e}
                    value={e}
                />
                <label class="form-check-label" for={e}>{e}</label>
            </div>
        {/each}
    </div>

    {#if encoder === 'jpg'}
        <div class="row mt-2">
            <label for="optionRange" class="col-form-label col-auto">{encoderOptionTitle}</label>
            <div class="col-auto">
                <input bind:value={imgOpts.jpgQuality} type="range" class="form-range form-range-sm" min="{encoderOptionMin}" max="{encoderOptionMax}" disabled={disabled}>
            </div>
            <div class="col-auto">
                <input bind:value={imgOpts.jpgQuality} type="number" max="{encoderOptionMax}" min="{encoderOptionMin}" step="1" class="form-control form-control-sm" disabled={disabled}>
            </div>
        </div>
    {:else if encoder === 'gif'}
        <div class="row mt-2">
            <label for="optionRange" class="col-form-label col-auto">{encoderOptionTitle}</label>
            <div class="col-auto">
                <input bind:value={imgOpts.gifNumColors} type="range" class="form-range form-range-sm" min="{encoderOptionMin}" max="{encoderOptionMax}" disabled={disabled}>
            </div>
            <div class="col-auto">
                <input bind:value={imgOpts.gifNumColors} type="number" max="{encoderOptionMax}" min="{encoderOptionMin}" step="1" class="form-control form-control-sm" disabled={disabled}>
            </div>
        </div>
    {:else if encoder === 'webp'}
        <div class="row mt-2">
            <label for="optionRange" class="col-form-label col-auto">{encoderOptionTitle}</label>
            <div class="col-auto">
                <input bind:value={imgOpts.webpQuality} type="range" class="form-range form-range-sm" min="{encoderOptionMin}" max="{encoderOptionMax}" id="optionRange" disabled={disabled}>
            </div>
            <div class="col-auto">
                <input bind:value={imgOpts.webpQuality} type="number" max="{encoderOptionMax}" min="{encoderOptionMin}" step="1" class="form-control form-control-sm" disabled={disabled}>
            </div>
            <div class="form-check col-auto">
                <input class="form-check-input" type="checkbox" bind:checked={imgOpts.webpLossless}  disabled={disabled}>
                <label class="form-check-label">
                    无损
                </label>
            </div>
        </div>
    {:else if encoder === 'png'}
        <div class="mt-2 d-flex flex-wrap">
            {encoderOptionTitle}: &nbsp;
            {#each pngCompress as cp}
                <div class="form-check form-check-inline">
                    <input
                        bind:group={imgOpts.pngCompression}
                        class="form-check-input"
                        type="radio"
                        name="opt"
                        id={cp.value}
                        value={cp.value}
                        disabled={disabled}
                    />
                    <label class="form-check-label" for={cp.value}>{cp.name}</label>
                </div>
            {/each}
        </div>
    {/if}

    {#if img.format === 'GIF' && img.frames > 1 && encoder === 'gif'}
        <div class="row mt-2">
            <div class="input-group input-group-sm col">
                <span class="input-group-text">抽帧</span>
                <select bind:value={imgOpts.gifDropRate} id="frameOp" class="form-select" disabled={disabled}>
                    {#each frameOps as d}
                        <option value={d.value}>{d.name}</option>
                    {/each}
                </select>
            </div>
            <div class="form-check col">
                <input class="form-check-input" type="checkbox" bind:checked={imgOpts.gifDrawOnBefore} id="flexCheckDefault" disabled={disabled}>
                <label class="form-check-label" for="flexCheckDefault">
                    前一帧上绘制
                </label>
            </div>
        </div>
    {/if}

    <div class="mt-2 input-group input-group-sm">
        <button
            on:click={openFolder}
            disabled={disabled}
            type="button"
            class="btn btn-outline-secondary btn-sm"
        >
            选择保存目录
        </button>
        <input type="text" bind:value={imgOpts.savePath} disabled  class="form-control">
    </div>
    <div class="mt-2 input-group input-group-sm">
        <span class="input-group-text">保存文件名</span>
        <input type="text" bind:value={imgOpts.saveName} disabled={disabled}  class="form-control">
    </div>

    <div class="mt-2">
        <button
            on:click={transform}
            type="button"
            class="btn btn-outline-primary btn-sm"
            class:disabled={!(!disabled && img.name !== undefined)}
        >
        {#if loading}
            <span
                    class="spinner-border spinner-border-sm text-primary"
                    role="status"
                    aria-hidden="true"
            />
        {/if}
            转换
        </button>
        <button
                on:click={reset}
                class:disabled={!(!disabled && img.name !== undefined)}
                type="button"
                class="btn btn-outline-warning btn-sm">
            恢复默认宽高
        </button>
    </div>
    {/if}
</div>