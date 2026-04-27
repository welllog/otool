<script>
    import FilePondWrapper from '$lib/FilePondWrapper.svelte';
    import md5 from 'md5';

    import * as image from "wjs/go/srvs/Image"
    import * as app from "wjs/go/internal/App";
    import { toast } from "$lib/ToastContainer.svelte";
    import { onMount, onDestroy } from 'svelte';
    import { srvs } from "wjs/go/models";
    import {Radio, Select, Input, Button, ButtonGroup, InputAddon, Range, Checkbox, Spinner, Progressbar} from 'flowbite-svelte';
    import Label from '$lib/Label.svelte';
    import {EventsOn,EventsOff} from 'wjs/runtime/runtime';

    const acceptedFileTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/bmp', 'image/tiff', 'image/webp', 'image/avif'];

    /** @type {any} */
    let pondRef;
    let filesNum = $state(0);
    let firstWidth = $state(0);
    let firstHeight = $state(0);
    let op = $state(0);
    let encoder = $state('png');
    let outPath = $state('');
    let width = $state(0);
    let height = $state(0);
    let percent = $state(100);
    let jpgQuality = $state(95);
    let pngCompression = $state(0);
    let gifNumColors = $state(256);
    let gifDropRate = $state(0);
    let gifDrawOnBefore = $state(false);
    let webpLossless = $state(false);
    let webpQuality = $state(90);
    let webpRgbInTransparent = $state(false);
    let avifQuality = $state(60);
    let avifQualityAlpha = $state(60);
    let avifSpeed = $state(10);

    let disabled = $state(false);
    let loading = $state(false);
    let curProgress = $state(0);
    let progressClose = () => {};
    let showProgress = $state(false);

    onMount(() => {
        app.DefaultPath().then((path) => {outPath = path})
    })

    function handleAddFile(/** @type {any} */ err, /** @type {any} */ fileItem) {
        if (acceptedFileTypes.includes(fileItem.file.type)) {
            filesNum++;
            if (firstWidth === 0 || firstHeight === 0) {
                loadImage(fileItem.file).then((img) => {
                    firstWidth = img.width
                    firstHeight = img.height
                    width = firstWidth
                    height = firstHeight
                })
            }
        }
    }

    function handleRemoveFile(/** @type {any} */ err, /** @type {any} */ fileItem) {
        if (acceptedFileTypes.includes(fileItem.file.type)) {
            filesNum--;
        }
    }

    function loadImage(/** @type {File} */ file) {
        return new Promise(function (resolve, reject) {
            let reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = function (evt) {
                let replaceSrc = /** @type {string} */ (evt.target?.result);
                let imageObj = new Image();
                imageObj.src = replaceSrc;
                imageObj.onload = function () {
                    resolve(imageObj);
                };
            };
        });
    }

    const ops = [
        { name: '原始尺寸', value: 0 },
        { name: '固定宽度缩放', value: 1 },
        { name: '最大宽度缩放', value: 2 },
        { name: '固定宽高裁剪', value: 3 },
        { name: '最大宽高缩放', value: 4 },
        { name: '固定高度缩放', value: 5 },
        { name: '最大高度缩放', value: 6 },
        { name: '百分比缩放', value: 7 },
    ]

    const encoders = ['jpg', 'png', 'gif', 'bmp', 'tiff', 'webp', 'avif'];

    const pngCompress = [
        { name: '默认', value: 0 },
        { name: '不压缩', value: -1 },
        { name: '速度优先', value: -2 },
        { name: '压缩优先', value: -3 },
    ]

    const frameOps = [
        { name: "不处理", value: 0 },
        { name: "抽取1/2", value: 2 },
        { name: "抽取1/3", value: 3 },
        { name: "抽取1/4", value: 4 },
        { name: "抽取1/5", value: 5 },
        { name: "抽取1/6", value: 6 },
        { name: "抽取1/7", value: 7 },
        { name: "抽取1/8", value: 8 },
    ]

    function reset() {
        width = firstWidth;
        height = firstHeight;
        percent = 100;
    }

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            outPath = path;

        } catch (/** @type {any} */ err) {
            toast("danger", err.toString())
        } finally {
            disabled = false;
        }
    }

    async function transform() {
        disabled = true;
        loading = true

        let files = pondRef.getFiles();
        // @ts-ignore
        let imgFiles = files.filter(f => acceptedFileTypes.includes(f.file.type));
        // @ts-ignore
        let filesName = imgFiles.map(f => f.file.name).join(',') + ',';

        if (imgFiles.length === 0) {
            toast("danger", "没有支持的图片文件")
            disabled = false;
            loading = false;
            return
        }

        let opts = new srvs.ImageOptions({
            op: Number(op),
            encoder: encoder,
            outPath: outPath,
            width: Number(width),
            height: Number(height),
            percent: Number(percent),
            jpgQuality: Number(jpgQuality),
            pngCompression: Number(pngCompression),
            gifNumColors: Number(gifNumColors),
            gifDropRate: Number(gifDropRate),
            gifDrawOnBefore: Boolean(gifDrawOnBefore),
            webpLossless: Boolean(webpLossless),
            webpQuality: Number(webpQuality),
            webpRgbInTransparent: Boolean(webpRgbInTransparent),
            avifQuality: Number(avifQuality),
            avifQualityAlpha: Number(avifQualityAlpha),
            avifSpeed: Number(avifSpeed),
        });

        filesName += Math.round(Math.random() * 10000)
        let eventName = /** @type {string} */ (md5(filesName))
        for (let i = 0; i < imgFiles.length; i++) {
            transformFile(imgFiles[i].file, imgFiles.length, eventName, opts)
        }

        curProgress = 0;
        showProgress = true;
        progressClose = EventsOn(eventName, (/** @type {number} */ progress) => {
            if (progress >= 0) {
                curProgress = progress;
            } else {
                EventsOff(/** @type {string} */ (eventName));
                showProgress = false;
                disabled = false;
            }
        })

        disabled = false;
        loading = false;
    }

    function setOptionForEncoder(/** @type {string} */ enc) {
        switch (enc) {
            case 'jpg':
                jpgQuality = 95
                break
            case 'gif':
                gifNumColors = 256
                gifDrawOnBefore = false
                gifDropRate = 0
                break
            case 'png':
                pngCompression = 0
                break
            case 'webp':
                webpQuality = 90
                webpLossless = false
                webpRgbInTransparent = false
                break
            case 'avif':
                avifQuality = 60
                avifQualityAlpha = 60
                avifSpeed = 10
        }
    }

    function onchangeEncoder(/** @type {Event} */ e) {
        setOptionForEncoder(/** @type {HTMLSelectElement} */ (e.target).value)
    }

    function transformFile(/** @type {File} */ file, /** @type {number} */ filesNum, /** @type {string} */ eventName, /** @type {srvs.ImageOptions} */ opts) {
        let reader = new FileReader();
        reader.onloadend = function (e) {
            if (e.target?.readyState === FileReader.DONE) {
                let buffer = new Uint8Array(/** @type {ArrayBuffer} */ (e.target?.result));
                let body = Array.from(buffer);
                let img = new srvs.ImageFile();
                img.name = file.name
                img.type = file.type
                img.body = body
                image.CropAndSave(img, opts, filesNum, eventName)
            }
        };
        reader.readAsArrayBuffer(file);
    }

    function cleanFiles() {
        pondRef.removeFiles();
    }

    onDestroy(() => {
        progressClose();
    })
</script>

<style>
    /* 滑动条基础样式 */
    .custom-range {
        -webkit-appearance: none;
        appearance: none;
        display: block; /* 确保作为块级元素占满宽度 */
        width: 100%;
        min-width: 100px; /* 防止在极端情况下缩回圆形 */
        height: 6px;
        background: #e5e7eb;
        border-radius: 999px;
        outline: none;
        cursor: pointer;
        transition: background 0.2s;
    }

    /* 深色模式轨道 */
    :global(.dark) .custom-range {
        background: #374151;
    }

    /* Webkit 滑块 (macOS/Wails) */
    .custom-range::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 18px;
        height: 18px;
        background: #7c3aed; /* primary-600 */
        border: 3px solid white;
        border-radius: 50%;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        cursor: grab;
        transition: all 0.2s;
        /* 这里的 margin-top 需要根据轨道高度调整以居中 */
        margin-top: -6px; 
    }

    .custom-range:active::-webkit-slider-thumb {
        cursor: grabbing;
        transform: scale(1.2);
        box-shadow: 0 0 15px rgba(124, 58, 237, 0.5);
    }

    :global(.dark) .custom-range::-webkit-slider-thumb {
        border-color: #1f2937;
        background: #8b5cf6; /* primary-500 */
    }

    /* 悬停轨道 */
    .custom-range:hover {
        background: #d1d5db;
    }
    :global(.dark) .custom-range:hover {
        background: #4b5563;
    }

    /* 禁用状态 */
    .custom-range:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    .custom-range:disabled::-webkit-slider-thumb {
        cursor: not-allowed;
    }
</style>

<div class="flex flex-col gap-4 max-w-5xl mx-auto py-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
        <div class="flex items-center gap-2 mb-4">
            <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">源图片</h2>
        </div>
        <FilePondWrapper bind:this={pondRef}
            allowMultiple={true}
            onaddfile={handleAddFile}
            onremovefile={handleRemoveFile}
            {acceptedFileTypes}
            labelFileTypeNotAllowed={'文件类型不支持'}
            {disabled}
        />
    </div>

    {#if showProgress }
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
            <Progressbar progress={curProgress} size="h-4" labelInside />
        </div>
    {/if}

    {#if filesNum > 0 }
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
                <div class="flex items-center gap-2 mb-4">
                    <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                    </svg>
                    <h2 class="text-lg font-semibold text-gray-900 dark:text-white">尺寸调整</h2>
                </div>
                
                <div class="mb-4">
                    <Label class="mb-2 text-gray-500 dark:text-gray-400 text-sm">缩略方式</Label>
                    <Select size="sm" items={ops} bind:value={op} disabled={disabled} class="w-full bg-gray-50 dark:bg-gray-900"/>
                </div>

                {#if op > 0 && op < 7}
                    <div class="grid grid-cols-2 gap-3">
                        {#if op < 5}
                            <div>
                                <Label class="mb-2 text-gray-500 dark:text-gray-400 text-sm">宽 (px)</Label>
                                <Input size="sm" bind:value={width} type="number" disabled={disabled} class="w-full bg-gray-50 dark:bg-gray-900" />
                            </div>
                        {/if}
                        {#if op > 2}
                            <div>
                                <Label class="mb-2 text-gray-500 dark:text-gray-400 text-sm">高 (px)</Label>
                                <Input size="sm" bind:value={height} type="number" disabled={disabled} class="w-full bg-gray-50 dark:bg-gray-900" />
                            </div>
                        {/if}
                    </div>
                {/if}

                {#if op === 7}
                    <div>
                        <Label class="mb-2 text-gray-500 dark:text-gray-400 text-sm">缩放百分比 (%)</Label>
                        <Input size="sm" bind:value={percent} type="number" disabled={disabled} class="w-full bg-gray-50 dark:bg-gray-900" />
                    </div>
                {/if}
            </div>

            <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
                <div class="flex items-center gap-2 mb-4">
                    <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                    <h2 class="text-lg font-semibold text-gray-900 dark:text-white">输出格式</h2>
                </div>

                <div class="mb-4">
                    <Label class="mb-2 text-gray-500 dark:text-gray-400 text-sm">保存格式</Label>
                    <div class="flex flex-wrap gap-4 p-2 bg-gray-50 dark:bg-gray-900 rounded-lg">
                        {#each encoders as e}
                            <Radio value={e} bind:group={encoder} onchange={onchangeEncoder} disabled={disabled} class="!p-0">{e}</Radio>
                        {/each}
                    </div>
                </div>

                <div class="pt-4 mt-2 border-t border-gray-100 dark:border-gray-700">
                    {#if encoder === 'jpg'}
                        <div class="flex flex-col gap-2">
                            <Label class="text-gray-500 dark:text-gray-400 text-sm">质量</Label>
                            <div class="flex gap-4 items-center group">
                                <div class="flex-1 py-3">
                                    <input type="range" bind:value={jpgQuality} min="1" max="100" disabled={disabled}
                                        class="custom-range" />
                                </div>
                                <Input class="w-20 text-center font-medium bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 focus:border-primary-500" bind:value={jpgQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
                            </div>
                        </div>
                    {:else if encoder === 'gif'}
                        <div class="flex flex-col gap-4">
                            <div class="flex flex-col gap-2">
                                <Label class="text-gray-500 dark:text-gray-400 text-sm">色彩数量</Label>
                                <div class="flex gap-4 items-center group">
                                    <div class="flex-1 py-3">
                                        <input type="range" bind:value={gifNumColors} min="1" max="256" disabled={disabled}
                                            class="custom-range" />
                                    </div>
                                    <Input class="w-20 text-center font-medium bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 focus:border-primary-500" bind:value={gifNumColors} size="sm" type="number" min="1" max="256" disabled={disabled} />
                                </div>
                            </div>
                            <div class="flex flex-col gap-2">
                                <Label class="text-gray-500 dark:text-gray-400 text-sm">抽帧</Label>
                                <Select size="sm" items={frameOps} bind:value={gifDropRate} disabled={disabled} class="bg-gray-50 dark:bg-gray-900"/>
                            </div>
                            <div class="flex flex-col gap-1">
                                <Checkbox bind:checked={gifDrawOnBefore} disabled={disabled}>前一帧上绘制</Checkbox>
                                <span class="text-xs text-gray-400 ml-6">(仅对原始图片为gif时有效)</span>
                            </div>
                        </div>
                    {:else if encoder === 'webp'}
                        <div class="flex flex-col gap-4">
                            {#if !webpLossless}
                            <div class="flex flex-col gap-2">
                                <Label class="text-gray-500 dark:text-gray-400 text-sm">质量</Label>
                                <div class="flex gap-4 items-center group">
                                    <div class="flex-1 py-3">
                                        <input type="range" bind:value={webpQuality} min="1" max="100" disabled={disabled}
                                            class="custom-range" />
                                    </div>
                                    <Input class="w-20 text-center font-medium bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 focus:border-primary-500" bind:value={webpQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
                                </div>
                            </div>
                            {/if}
                            <div class="flex flex-col gap-2">
                                <Checkbox bind:checked={webpLossless} disabled={disabled}>无损</Checkbox>
                                {#if webpLossless}
                                <Checkbox class="ml-6" bind:checked={webpRgbInTransparent} disabled={disabled}>保留透明区域rgb</Checkbox>
                                {/if}
                            </div>
                        </div>
                    {:else if encoder === 'png'}
                        <div class="flex flex-col gap-2">
                            <Label class="text-gray-500 dark:text-gray-400 text-sm">压缩等级</Label>
                            <div class="flex flex-col gap-2 p-3 bg-gray-50 dark:bg-gray-900 rounded-lg">
                                {#each pngCompress as pc}
                                    <Radio value={pc.value} bind:group={pngCompression} disabled={disabled} class="!p-0">{pc.name}</Radio>
                                {/each}
                            </div>
                        </div>
                    {:else if encoder === 'avif'}
                        <div class="flex flex-col gap-4">
                            <div class="flex flex-col gap-2">
                                <Label class="text-gray-500 dark:text-gray-400 text-sm">质量</Label>
                                <div class="flex gap-4 items-center group">
                                    <div class="flex-1 py-3">
                                        <input type="range" bind:value={avifQuality} min="1" max="100" disabled={disabled}
                                            class="custom-range" />
                                    </div>
                                    <Input class="w-20 text-center font-medium bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 focus:border-primary-500" bind:value={avifQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
                                </div>
                            </div>
                            <div class="flex flex-col gap-2">
                                <Label class="text-gray-500 dark:text-gray-400 text-sm">alpha通道质量</Label>
                                <div class="flex gap-4 items-center group">
                                    <div class="flex-1 py-3">
                                        <input type="range" bind:value={avifQualityAlpha} min="1" max="100" disabled={disabled}
                                            class="custom-range" />
                                    </div>
                                    <Input class="w-20 text-center font-medium bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 focus:border-primary-500" bind:value={avifQualityAlpha} size="sm" type="number" min="1" max="100" disabled={disabled} />
                                </div>
                            </div>
                            <div class="flex flex-col gap-2">
                                <Label class="text-gray-500 dark:text-gray-400 text-sm">速度</Label>
                                <div class="flex gap-4 items-center group">
                                    <div class="flex-1 py-3">
                                        <input type="range" bind:value={avifSpeed} min="1" max="10" disabled={disabled}
                                            class="custom-range" />
                                    </div>
                                    <Input class="w-20 text-center font-medium bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 focus:border-primary-500" bind:value={avifSpeed} size="sm" type="number" min="1" max="10" disabled={disabled} />
                                </div>
                            </div>
                        </div>
                    {/if}
                </div>
            </div>
        </div>

        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5 flex flex-col md:flex-row gap-4 items-center justify-between">
            <div class="w-full md:w-3/5 flex flex-col gap-2">
                <Label class="text-gray-500 dark:text-gray-400 text-sm">保存目录</Label>
                <ButtonGroup class="w-full">
                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <InputAddon class="flex-shrink-0 cursor-pointer bg-primary-600 text-white hover:bg-primary-700 border-primary-600 transition-colors" onclick={openFolder}>
                        选择目录
                    </InputAddon>
                    <Input bind:value={outPath} class="w-full !rounded-l-none bg-gray-50 dark:bg-gray-900" disabled/>
                </ButtonGroup>
            </div>
            
            <div class="w-full md:w-auto flex flex-col sm:flex-row gap-3 md:pt-6">
                <Button color="light" onclick={reset} disabled={disabled || filesNum === 0} class="w-full sm:w-auto dark:bg-gray-900 dark:hover:bg-gray-700">
                    {#if filesNum < 2}恢复默认宽高{:else}首图宽高{/if}
                </Button>
                <Button color="red" outline onclick={cleanFiles} disabled={disabled || filesNum === 0} class="w-full sm:w-auto">
                    清空列表
                </Button>
                <Button color="primary" onclick={transform} disabled={disabled || filesNum === 0} class="w-full sm:w-auto px-6 shadow-md shadow-primary-500/30">
                    {#if loading}
                        <Spinner size="4" class="mr-2"/>
                    {/if}
                    开始转换
                </Button>
            </div>
        </div>
    {/if}
</div>
