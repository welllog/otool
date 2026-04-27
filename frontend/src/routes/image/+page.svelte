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
        let imgFiles = [];
        let filesName = '';
        for (let i = 0; i < files.length; i++) {
            if (acceptedFileTypes.includes(files[i].file.type)) {
                imgFiles.push(files[i])
                filesName += files[i].file.name + ','
            }
        }

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
                let body = [];
                for (let i = 0; i < buffer.length; i++) {
                    body.push(buffer[i]);
                }
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

<div class="my-3">
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
    <div class="mb-3">
        <Progressbar progress={curProgress} size="h-4" labelInside />
    </div>
{/if}

{#if filesNum > 0 }
    <ButtonGroup class="w-full mb-3">
        <InputAddon class="flex-shrink-0">缩略方式</InputAddon>
        <Select size="sm" items={ops} bind:value={op} class="!rounded-l-none" disabled={disabled}/>
    </ButtonGroup>

    {#if op > 0 && op < 7}
        <div class="mb-3 grid gap-1 md:grid-cols-2">
            {#if op < 5}
                <ButtonGroup>
                    <InputAddon>宽</InputAddon>
                    <Input size="sm" bind:value={width} type="number" disabled={disabled}  />
                    <InputAddon>px</InputAddon>
                </ButtonGroup>
            {/if}
            {#if op > 2}
                <ButtonGroup>
                    <InputAddon>高</InputAddon>
                    <Input size="sm" bind:value={height} type="number" disabled={disabled}  />
                    <InputAddon>px</InputAddon>
                </ButtonGroup>
            {/if}
        </div>
    {/if}

    {#if op === 7}
        <div class="mb-3">
            <ButtonGroup class="w-full">
                <InputAddon class="flex-shrink-0">缩放百分比</InputAddon>
                <Input size="sm" bind:value={percent} type="number" disabled={disabled}  />
                <InputAddon>%</InputAddon>
            </ButtonGroup>
        </div>
    {/if}

    <div class="mb-3 flex flex-wrap gap-4">
        <Label>保存格式:</Label>
        {#each encoders as e}
            <Radio value={e} bind:group={encoder} onchange={onchangeEncoder}>{e}</Radio>
        {/each}
    </div>

    {#if encoder === 'jpg'}
        <div class="flex mb-3 gap-2 items-center">
            <Label>质量:</Label>
            <Range class="w-auto" bind:value={jpgQuality} min="1" max="100" disabled={disabled} />
            <Input class="w-fit" bind:value={jpgQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
        </div>
    {:else if encoder === 'gif'}
        <div class="flex mb-3 gap-2 items-center">
            <Label>色彩数量:</Label>
            <Range class="w-auto" bind:value={gifNumColors} min="1" max="256" disabled={disabled} />
            <Input class="w-fit" bind:value={gifNumColors} size="sm" type="number" min="1" max="256" disabled={disabled} />
        </div>
        <div class="row mb-3 flex gap-2 items-center">
            <ButtonGroup class="w-auto">
                <InputAddon class="flex-shrink-0">抽帧</InputAddon>
                <Select size="sm" items={frameOps} bind:value={gifDropRate} class="!rounded-l-none" disabled={disabled}/>
            </ButtonGroup>
            <Checkbox bind:checked={gifDrawOnBefore}  disabled={disabled}>前一帧上绘制</Checkbox>
            <Label>(仅对原始图片为gif时有效)</Label>
        </div>
    {:else if encoder === 'webp'}
        <div class="flex mb-3 gap-2 items-center">
            {#if !webpLossless}
            <Label>质量:</Label>
            <Range class="w-auto" bind:value={webpQuality} min="1" max="100" disabled={disabled} />
            <Input class="w-fit" bind:value={webpQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
            {/if}
            <Checkbox bind:checked={webpLossless}  disabled={disabled}>无损</Checkbox>
            {#if webpLossless}
            <Checkbox bind:checked={webpRgbInTransparent}  disabled={disabled}>保留透明区域rgb</Checkbox>
            {/if}
        </div>
    {:else if encoder === 'png'}
        <div class="mb-3 flex flex-wrap gap-4">
            <Label>压缩等级:</Label>
            {#each pngCompress as pc}
                <Radio value={pc.value} bind:group={pngCompression} disabled={disabled}>{pc.name}</Radio>
            {/each}
        </div>
    {:else if encoder === 'avif'}
        <div class="flex flex-wrap mb-3 gap-4">
            <div class="flex items-center">
                <Label>质量:</Label>
                <Range class="w-auto" bind:value={avifQuality} min="1" max="100" disabled={disabled} />
                <Input class="w-fit" bind:value={avifQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
            </div>

            <div class="flex items-center">
                <Label>alpha通道质量:</Label>
                <Range class="w-auto" bind:value={avifQualityAlpha} min="1" max="100" disabled={disabled} />
                <Input class="w-fit" bind:value={avifQualityAlpha} size="sm" type="number" min="1" max="100" disabled={disabled} />
            </div>

            <div class="flex items-center">
                <Label>速度:</Label>
                <Range class="w-auto" bind:value={avifSpeed} min="1" max="10" disabled={disabled} />
                <Input class="w-fit" bind:value={avifSpeed} size="sm" type="number" min="1" max="10" disabled={disabled} />
            </div>
        </div>
    {/if}

    <ButtonGroup class="w-full mb-3">
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <InputAddon class="flex-shrink-0 cursor-pointer bg-green-500 text-white hover:bg-green-600 border-green-600" onclick={openFolder}>
            选择保存目录
        </InputAddon>
        <Input bind:value={outPath} class="!rounded-l-none" disabled/>
    </ButtonGroup>

    <div class="mb-3">
        <Button outline color="blue" size="xs" onclick={transform} disabled={!(!disabled && filesNum > 0)}>
            {#if loading}
                <Spinner size="4" class="mr-3"/>
            {/if}
            转换
        </Button>
        <Button outline color="dark" size="xs" onclick={reset}  disabled={!(!disabled && filesNum > 0)}>
            {#if filesNum < 2}恢复默认宽高{:else}采用首图宽高{/if}
        </Button>
        <Button outline color="yellow" size="xs" onclick={cleanFiles} disabled={!(!disabled && filesNum > 0)}>
            清空选中文件
        </Button>
    </div>
{/if}
