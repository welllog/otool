<script>
    import FilePondWrapper from '$lib/FilePondWrapper.svelte';
    import md5 from 'md5';

    import * as image from "wjs/go/srvs/Image"
    import * as app from "wjs/go/internal/App";
    import { toast } from "$lib/ToastContainer.svelte";
    import { onMount, onDestroy } from 'svelte';
    import { srvs } from "wjs/go/models";
    import {Radio, Select, Input, Button, Checkbox, Spinner, Progressbar} from 'flowbite-svelte';
    import Label from '$lib/Label.svelte';
    import Card from '$lib/Card.svelte';
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
            if (filesNum === 0) {
                firstWidth = 0;
                firstHeight = 0;
            }
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
        { name: '保持原尺寸', value: 0 },
        { name: '固定宽度缩放', value: 1 },
        { name: '最大宽度缩放', value: 2 },
        { name: '固定宽高裁剪', value: 3 },
        { name: '最大宽高缩放', value: 4 },
        { name: '固定高度缩放', value: 5 },
        { name: '最大高度缩放', value: 6 },
        { name: '按百分比缩放', value: 7 },
    ]

    const encoders = ['jpg', 'png', 'gif', 'bmp', 'tiff', 'webp', 'avif'];

    const pngCompress = [
        { name: '默认级别', value: 0 },
        { name: '不压缩', value: -1 },
        { name: '速度优先', value: -2 },
        { name: '压缩优先', value: -3 },
    ]

    const frameOps = [
        { name: "不处理", value: 0 },
        { name: "抽取 1/2", value: 2 },
        { name: "抽取 1/3", value: 3 },
        { name: "抽取 1/4", value: 4 },
        { name: "抽取 1/5", value: 5 },
        { name: "抽取 1/6", value: 6 },
        { name: "抽取 1/7", value: 7 },
        { name: "抽取 1/8", value: 8 },
    ]

    function reset() {
        width = firstWidth;
        height = firstHeight;
        percent = 100;
        toast('success', '已恢复初始宽高配置');
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
            toast("warning", "请先添加支持的图片文件")
            disabled = false;
            loading = false;
            return
        }

        if (!outPath) {
            toast("warning", "请选择保存目录");
            disabled = false;
            loading = false;
            return;
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
                toast('success', '批量转换已完成');
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
        filesNum = 0;
        firstWidth = 0;
        firstHeight = 0;
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
        display: block; 
        width: 100%;
        min-width: 100px;
        height: 6px;
        background: #e5e7eb;
        border-radius: 999px;
        outline: none;
        cursor: pointer;
        transition: background 0.2s;
    }

    :global(.dark) .custom-range {
        background: #374151;
    }

    .custom-range::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 18px;
        height: 18px;
        background: #7c3aed; 
        border: 3px solid white;
        border-radius: 50%;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        cursor: grab;
        transition: all 0.2s;
        margin-top: -6px; 
    }

    .custom-range:active::-webkit-slider-thumb {
        cursor: grabbing;
        transform: scale(1.2);
        box-shadow: 0 0 15px rgba(124, 58, 237, 0.5);
    }

    :global(.dark) .custom-range::-webkit-slider-thumb {
        border-color: #1f2937;
        background: #8b5cf6; 
    }

    .custom-range:hover {
        background: #d1d5db;
    }
    :global(.dark) .custom-range:hover {
        background: #4b5563;
    }

    .custom-range:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    .custom-range:disabled::-webkit-slider-thumb {
        cursor: not-allowed;
    }
</style>

<div class="flex flex-col gap-6 max-w-6xl mx-auto py-6">
    <!-- Source Image Card -->
    <Card 
        title="源图片" 
        icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>`}
        delay={0}
    >
        <FilePondWrapper bind:this={pondRef}
            allowMultiple={true}
            onaddfile={handleAddFile}
            onremovefile={handleRemoveFile}
            {acceptedFileTypes}
            labelFileTypeNotAllowed={'文件类型不支持'}
            {disabled}
        />
    </Card>

    <!-- Progress Card -->
    {#if showProgress }
        <Card delay={0} class="border-primary-200 dark:border-primary-900 shadow-xl shadow-primary-500/5">
            <div class="space-y-4">
                <div class="flex items-center justify-between">
                    <Label class="text-xs font-bold uppercase tracking-wider text-primary-600 flex items-center gap-2">
                        <Spinner size="4" color="blue"/> 正在批量处理中...
                    </Label>
                    <span class="text-sm font-black text-primary-700 dark:text-primary-400">{curProgress}%</span>
                </div>
                <Progressbar progress={curProgress} size="h-3" color="blue" class="rounded-full bg-gray-100 dark:bg-gray-800" />
            </div>
        </Card>
    {/if}

    {#if filesNum > 0 }
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch">
            <!-- Resize Card -->
            <Card 
                title="尺寸调整" 
                icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" /></svg>`}
                delay={100}
                class="flex flex-col"
            >
                <div class="space-y-6 flex-1">
                    <div>
                        <Label class="text-xs font-bold uppercase tracking-wider text-gray-400 mb-3">缩略模式</Label>
                        <Select size="md" items={ops} bind:value={op} disabled={disabled} class="w-full bg-gray-50/50 dark:bg-gray-800/50 border-gray-100 dark:border-gray-800 rounded-xl"/>
                    </div>
                    
                    {#if op > 0 && op < 7}
                        <div class="grid grid-cols-2 gap-4 pt-4 border-t border-gray-50 dark:border-gray-800">
                            {#if op < 5}
                                <div class="space-y-2">
                                    <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">宽度 (PX)</Label>
                                    <Input size="md" bind:value={width} type="number" disabled={disabled} class="w-full bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 font-mono text-center" />
                                </div>
                            {/if}
                            {#if op > 2}
                                <div class="space-y-2">
                                    <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">高度 (PX)</Label>
                                    <Input size="md" bind:value={height} type="number" disabled={disabled} class="w-full bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 font-mono text-center" />
                                </div>
                            {/if}
                        </div>
                    {/if}

                    {#if op === 7}
                        <div class="pt-4 border-t border-gray-50 dark:border-gray-800">
                            <Label class="text-xs font-bold uppercase tracking-wider text-gray-400 mb-3">缩放比例 (%)</Label>
                            <Input size="lg" bind:value={percent} type="number" disabled={disabled} class="w-full bg-primary-50/20 dark:bg-primary-900/10 border-primary-100/50 dark:border-primary-900/20 font-mono text-center text-primary-600 dark:text-primary-400" />
                        </div>
                    {/if}
                </div>
            </Card>

            <!-- Encoder Card -->
            <Card 
                title="参数配置" 
                icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /></svg>`}
                delay={200}
                class="flex flex-col"
            >
                <div class="space-y-6 flex-1">
                    <div>
                        <Label class="text-xs font-bold uppercase tracking-wider text-gray-400 mb-4">输出保存格式</Label>
                        <div class="flex flex-wrap gap-3">
                            {#each encoders as e}
                                <button 
                                    class="flex items-center px-4 py-2 rounded-xl border-2 transition-all font-black text-xs uppercase tracking-tight
                                    {encoder === e 
                                        ? 'bg-primary-600 border-primary-600 text-white shadow-lg shadow-primary-500/20' 
                                        : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                                    onclick={() => { encoder = e; setOptionForEncoder(e); }}
                                    disabled={disabled}
                                >
                                    {e}
                                </button>
                            {/each}
                        </div>
                    </div>

                    <div class="pt-8 border-t border-gray-100 dark:border-gray-800">
                        {#if encoder === 'jpg'}
                            <div class="space-y-5">
                                <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">压缩质量 (Quality: {jpgQuality})</Label>
                                <div class="flex gap-6 items-center bg-gray-50/50 dark:bg-gray-800/30 p-4 rounded-2xl border border-gray-100 dark:border-gray-700">
                                    <div class="flex-1">
                                        <input type="range" bind:value={jpgQuality} min="1" max="100" disabled={disabled} class="custom-range" />
                                    </div>
                                    <span class="w-10 text-right font-black text-primary-600 dark:text-primary-400 text-lg">{jpgQuality}</span>
                                </div>
                            </div>
                        {:else if encoder === 'gif'}
                            <div class="space-y-8">
                                <div class="space-y-5">
                                    <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">色彩数量 (Colors: {gifNumColors})</Label>
                                    <div class="flex gap-6 items-center bg-gray-50/50 dark:bg-gray-800/30 p-4 rounded-2xl border border-gray-100 dark:border-gray-700">
                                        <div class="flex-1">
                                            <input type="range" bind:value={gifNumColors} min="1" max="256" disabled={disabled} class="custom-range" />
                                        </div>
                                        <span class="w-10 text-right font-black text-primary-600 dark:text-primary-400 text-lg">{gifNumColors}</span>
                                    </div>
                                </div>
                                <div class="grid grid-cols-1 sm:grid-cols-2 gap-6 pt-2">
                                    <div class="space-y-3">
                                        <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">抽帧处理 (Drop Rate)</Label>
                                        <Select size="md" items={frameOps} bind:value={gifDropRate} disabled={disabled} class="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 rounded-xl font-bold"/>
                                    </div>
                                    <div class="flex flex-col justify-end">
                                        <div class="flex items-center gap-3 p-3 bg-gray-50/50 dark:bg-gray-800/50 rounded-xl border border-transparent hover:border-gray-200 transition-all cursor-pointer">
                                            <Checkbox bind:checked={gifDrawOnBefore} disabled={disabled} class="w-5 h-5 rounded-lg" />
                                            <span class="text-xs font-bold text-gray-500 dark:text-gray-400">在前一帧上绘制</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {:else if encoder === 'webp'}
                            <div class="space-y-8">
                                <div class="space-y-5">
                                    <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">WebP 质量 (Quality: {webpQuality})</Label>
                                    <div class="flex gap-6 items-center bg-gray-50/50 dark:bg-gray-800/30 p-4 rounded-2xl border border-gray-100 dark:border-gray-700">
                                        <div class="flex-1">
                                            <input type="range" bind:value={webpQuality} min="1" max="100" disabled={disabled} class="custom-range" />
                                        </div>
                                        <span class="w-10 text-right font-black text-primary-600 dark:text-primary-400 text-lg">{webpQuality}</span>
                                    </div>
                                </div>
                                <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                                    <div class="flex items-center gap-3 p-4 bg-gray-50/50 dark:bg-gray-800/50 rounded-2xl border border-transparent hover:border-gray-200 transition-all cursor-pointer">
                                        <Checkbox bind:checked={webpLossless} disabled={disabled} class="w-5 h-5 rounded-lg" />
                                        <span class="text-xs font-bold text-gray-500 dark:text-gray-400">无损压缩 (Lossless)</span>
                                    </div>
                                    {#if webpLossless}
                                        <div class="flex items-center gap-3 p-4 bg-gray-50/50 dark:bg-gray-800/50 rounded-2xl border border-transparent hover:border-gray-200 transition-all cursor-pointer">
                                            <Checkbox bind:checked={webpRgbInTransparent} disabled={disabled} class="w-5 h-5 rounded-lg" />
                                            <span class="text-xs font-bold text-gray-500 dark:text-gray-400">保留透明色 RGB</span>
                                        </div>
                                    {/if}
                                </div>
                            </div>
                        {:else if encoder === 'png'}
                            <div class="space-y-6">
                                <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">PNG 压缩策略 (Compression)</Label>
                                <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                                    {#each pngCompress as pc}
                                        <button 
                                            class="flex items-center justify-between p-4 rounded-2xl border-2 transition-all
                                            {pngCompression === pc.value 
                                                ? 'bg-primary-600 border-primary-600 text-white shadow-lg shadow-primary-500/20' 
                                                : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                                            onclick={() => pngCompression = pc.value}
                                            disabled={disabled}
                                        >
                                            <span class="text-xs font-bold uppercase tracking-tight">{pc.name}</span>
                                            {#if pngCompression === pc.value}
                                                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                                            {/if}
                                        </button>
                                    {/each}
                                </div>
                            </div>
                        {:else if encoder === 'avif'}
                            <div class="space-y-8">
                                <div class="space-y-6">
                                    <div class="space-y-4">
                                        <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">AVIF 质量 (Quality: {avifQuality})</Label>
                                        <div class="flex gap-6 items-center bg-gray-50/50 dark:bg-gray-800/30 p-4 rounded-2xl border border-gray-100 dark:border-gray-700">
                                            <input type="range" bind:value={avifQuality} min="1" max="100" disabled={disabled} class="custom-range flex-1" />
                                            <span class="w-10 text-right font-black text-primary-600 dark:text-primary-400">{avifQuality}</span>
                                        </div>
                                    </div>
                                    <div class="space-y-4">
                                        <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">Alpha 质量 (Alpha: {avifQualityAlpha})</Label>
                                        <div class="flex gap-6 items-center bg-gray-50/50 dark:bg-gray-800/30 p-4 rounded-2xl border border-gray-100 dark:border-gray-700">
                                            <input type="range" bind:value={avifQualityAlpha} min="1" max="100" disabled={disabled} class="custom-range flex-1" />
                                            <span class="w-10 text-right font-black text-primary-600 dark:text-primary-400">{avifQualityAlpha}</span>
                                        </div>
                                    </div>
                                    <div class="space-y-4">
                                        <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">编码速度 (Speed: {avifSpeed})</Label>
                                        <div class="flex gap-6 items-center bg-gray-50/50 dark:bg-gray-800/30 p-4 rounded-2xl border border-gray-100 dark:border-gray-700">
                                            <input type="range" bind:value={avifSpeed} min="0" max="10" disabled={disabled} class="custom-range flex-1" />
                                            <span class="w-10 text-right font-black text-primary-600 dark:text-primary-400">{avifSpeed}</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {:else}
                            <div class="h-24 flex flex-col items-center justify-center bg-gray-50/50 dark:bg-gray-800/30 rounded-3xl border-2 border-dashed border-gray-100 dark:border-gray-800">
                                <svg class="w-8 h-8 text-gray-300 dark:text-gray-600 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                                <span class="text-xs font-bold text-gray-400 uppercase tracking-widest">此格式无需额外配置</span>
                            </div>
                        {/if}
                    </div>
                </div>
            </Card>
        </div>

        <!-- Footer Actions Card -->
        <Card delay={300} class="p-0 overflow-hidden border-none shadow-xl shadow-gray-200/20 dark:shadow-none">
            <div class="p-6 bg-white dark:bg-gray-900/40 flex flex-col md:flex-row gap-8 items-center justify-between">
                <div class="w-full md:w-2/3 space-y-4">
                    <Label class="text-xs font-bold uppercase tracking-wider text-gray-400">保存路径</Label>
                    <div class="flex items-center gap-2 bg-gray-50/50 dark:bg-gray-800/50 p-2 pl-4 rounded-2xl border border-gray-100 dark:border-gray-700 shadow-sm overflow-hidden">
                        <span class="flex-1 text-sm text-gray-500 truncate font-mono">{outPath || '请选择保存目录'}</span>
                        <Button size="sm" color="primary" class="rounded-xl px-6 bg-primary-600 hover:bg-primary-700 border-none shadow-lg shadow-primary-500/20 transition-all active:scale-95" onclick={openFolder} disabled={disabled}>
                            更改路径
                        </Button>
                    </div>
                </div>
                
                <div class="w-full md:w-auto flex flex-col sm:flex-row gap-3 pt-6 md:pt-4">
                    <Button color="light" onclick={reset} disabled={disabled || filesNum === 0} class="rounded-2xl border-gray-100 dark:border-gray-700 dark:bg-gray-800 py-3 font-bold text-xs uppercase tracking-wider">
                        重置宽高
                    </Button>
                    <Button color="alternative" onclick={cleanFiles} disabled={disabled || filesNum === 0} class="rounded-2xl py-3 font-bold text-xs uppercase tracking-wider border-none bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600">
                        清空列表
                    </Button>
                    <Button color="primary" onclick={transform} disabled={disabled || filesNum === 0} class="rounded-2xl px-10 py-3 bg-primary-600 hover:bg-primary-700 shadow-xl shadow-primary-500/20 border-none transition-all active:scale-95 text-white font-black text-sm">
                        {#if loading}
                            <Spinner size="4" class="mr-2"/>
                        {/if}
                        开始转换
                    </Button>
                </div>
            </div>
        </Card>
    {/if}
</div>
