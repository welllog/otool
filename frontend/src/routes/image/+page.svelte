<script>
    import 'filepond/dist/filepond.min.css';
    import 'filepond-plugin-image-preview/dist/filepond-plugin-image-preview.min.css';
    import FilePond, { registerPlugin } from 'svelte-filepond';
    import FilePondPluginImageExifOrientation from 'filepond-plugin-image-exif-orientation';
    import FilePondPluginImagePreview from 'filepond-plugin-image-preview';
    import FilePondPluginFileValidateType from 'filepond-plugin-file-validate-type';
    import md5 from 'md5';

    import * as image from "wjs/go/srvs/Image"
    import * as app from "wjs/go/internal/App";
    import { toast } from "$lib/ToastContainer.svelte";
    import { onMount, onDestroy } from 'svelte';
    import { srvs } from "wjs/go/models";
    import {Radio, Select, Input, Button, ButtonGroup, InputAddon, Range, Checkbox, Spinner, Progressbar} from 'flowbite-svelte';
    import Label from '$lib/Label.svelte';
    import {EventsOn,EventsOff} from 'wjs/runtime/runtime';

    // Register the plugins
    registerPlugin(
        FilePondPluginImageExifOrientation,
        FilePondPluginImagePreview,
        FilePondPluginFileValidateType,
    );

    const name = 'filepond';
    const acceptedFileTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/bmp', 'image/tiff', 'image/webp', 'image/avif'];
    let pond;
    let filesNum = 0;
    let firstWidth = 0, firstHeight = 0;
    let imgOpts = new srvs.ImageOptions();
    imgOpts.op = 0;
    imgOpts.encoder = 'png';
    imgOpts.percent = 100;

    let disabled = false, loading = false;
    setOptionForEncoder(imgOpts.encoder)
    let curProgress = 0;
    let progressClose = () => {};
    let showProgress = false;

    onMount(() => {
        app.DefaultPath().then((path) => {imgOpts.outPath = path})
    })

    function handleAddFile(err, fileItem) {
        if (acceptedFileTypes.includes(fileItem.file.type)) {
            filesNum++;
            if (firstWidth === 0 || firstHeight === 0) {
                loadImage(fileItem.file).then((img) => {
                    firstWidth = img.width
                    firstHeight = img.height
                    imgOpts.width = firstWidth
                    imgOpts.height = firstHeight
                })
            }
        }
    }

    function handleRemoveFile(err, fileItem) {
        if (acceptedFileTypes.includes(fileItem.file.type)) {
            filesNum--;
        }
    }

    function loadImage(file) {
        return new Promise(function (resolve, reject) {
            let reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = function (evt) {
                let replaceSrc = evt.target.result;
                let imageObj = new Image();
                imageObj.src = replaceSrc;
                imageObj.onload = function () {
                    resolve(imageObj);
                };
            };
        });
    }

    const ops = [
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

    const encoders = ['jpg', 'png', 'gif', 'bmp', 'tiff', 'webp', 'avif'];

    const pngCompress = [
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

    const frameOps = [
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

    function reset() {
        imgOpts.width = firstWidth;
        imgOpts.height = firstHeight;
        imgOpts.percent = 100;
    }

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            imgOpts.outPath = path;

        } catch (err) {
            toast("danger", err.toString())
        } finally {
            disabled = false;
        }
    }

    async function transform() {
        disabled = true;
        loading = true

        let files = pond.getFiles();
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

        imgOpts.height = Number(imgOpts.height)
        imgOpts.width = Number(imgOpts.width)
        imgOpts.percent = Number(imgOpts.percent)
        imgOpts.jpgQuality = Number(imgOpts.jpgQuality)
        imgOpts.gifNumColors = Number(imgOpts.gifNumColors)
        imgOpts.webpQuality = Number(imgOpts.webpQuality)
        imgOpts.avifQuality = Number(imgOpts.avifQuality)
        imgOpts.avifQualityAlpha = Number(imgOpts.avifQualityAlpha)
        imgOpts.avifSpeed = Number(imgOpts.avifSpeed)
        filesName += Math.round(Math.random() * 10000)
        let eventName = md5(filesName)
        for (let i = 0; i < imgFiles.length; i++) {
            transformFile(imgFiles[i].file, imgFiles.length, eventName)
        }

        curProgress = 0;
        showProgress = true;
        progressClose = EventsOn(eventName, (progress) => {
            if (progress >= 0) {
                curProgress = progress;
            } else {
                EventsOff(eventName);
                showProgress = false;
                disabled = false;
            }
        })

        disabled = false;
        loading = false;
    }

    function setOptionForEncoder(enc) {
        switch (enc) {
            case 'jpg':
                imgOpts.jpgQuality = 95
                break
            case 'gif':
                imgOpts.gifNumColors = 256
                imgOpts.gifDrawOnBefore = false
                imgOpts.gifDropRate = 0
                break
            case 'png':
                imgOpts.pngCompression = 0
                break
            case 'webp':
                imgOpts.webpQuality = 90
                imgOpts.webpLossless = false
                imgOpts.webpRgbInTransparent = false
            case 'avif':
                imgOpts.avifQuality = 60
                imgOpts.avifQualityAlpha = 60
                imgOpts.avifSpeed = 10
        }
    }

    function onchangeEncoder(e) {
        setOptionForEncoder(e.target.value)
    }

    function transformFile(file, filesNum, eventName) {
        let reader = new FileReader();
        reader.onloadend = function (e) {
            if (e.target.readyState === FileReader.DONE) {
                let buffer = new Uint8Array(e.target.result);
                let body = [];
                for (let i = 0; i < buffer.length; i++) {
                    body.push(buffer[i]);
                }
                let img = new srvs.ImageFile();
                img.name = file.name
                img.type = file.type
                img.body = body
                image.CropAndSave(img, imgOpts, filesNum, eventName)
            }
        };
        reader.readAsArrayBuffer(file);
    }

    function cleanFiles() {
        pond.removeFiles();
    }

    onDestroy(() => {
        progressClose();
    })
</script>

<div class="my-3">
    <FilePond bind:this={pond} {name}
        allowMultiple={true}
        onaddfile={handleAddFile}
        onremovefile={handleRemoveFile}
        acceptedFileTypes={acceptedFileTypes}
        labelFileTypeNotAllowed={'文件类型不支持'}
        disabled={disabled}
    />
</div>

{#if showProgress }
    <div class="mb-3">
        <Progressbar bind:progress={curProgress} size="h-4" labelInside />
    </div>
{/if}

{#if filesNum > 0 }
    <div class="mb-3 flex">
        <InputAddon class="flex-shrink-0">缩略方式</InputAddon>
        <Select size="sm" items={ops} bind:value={imgOpts.op} class="!rounded-l-none" disabled={disabled}/>
    </div>

    {#if imgOpts.op > 0 && imgOpts.op < 7}
        <div class="mb-3 grid gap-1 md:grid-cols-2">
            {#if imgOpts.op < 5}
                <ButtonGroup>
                    <InputAddon>宽</InputAddon>
                    <Input size="sm" bind:value={imgOpts.width} type="number" disabled={disabled}  />
                    <InputAddon>px</InputAddon>
                </ButtonGroup>
            {/if}
            {#if imgOpts.op > 2}
                <ButtonGroup>
                    <InputAddon>高</InputAddon>
                    <Input size="sm" bind:value={imgOpts.height} type="number" disabled={disabled}  />
                    <InputAddon>px</InputAddon>
                </ButtonGroup>
            {/if}
        </div>
    {/if}

    {#if imgOpts.op === 7}
        <div class="mb-3">
            <ButtonGroup class="w-full">
                <InputAddon class="flex-shrink-0">缩放百分比</InputAddon>
                <Input size="sm" bind:value={imgOpts.percent} type="number" disabled={disabled}  />
                <InputAddon>%</InputAddon>
            </ButtonGroup>
        </div>
    {/if}

    <div class="mb-3 flex flex-wrap gap-4">
        <Label>保存格式:</Label>
        {#each encoders as e}
            <Radio value={e} bind:group={imgOpts.encoder} on:change={onchangeEncoder}>{e}</Radio>
        {/each}
    </div>

    {#if imgOpts.encoder === 'jpg'}
        <div class="flex mb-3 gap-2 items-center">
            <Label>质量:</Label>
            <Range class="w-auto" bind:value={imgOpts.jpgQuality} min="1" max="100" disabled={disabled} />
            <Input class="w-fit" bind:value={imgOpts.jpgQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
        </div>
    {:else if imgOpts.encoder === 'gif'}
        <div class="flex mb-3 gap-2 items-center">
            <Label>色彩数量:</Label>
            <Range class="w-auto" bind:value={imgOpts.gifNumColors} min="1" max="256" disabled={disabled} />
            <Input class="w-fit" bind:value={imgOpts.gifNumColors} size="sm" type="number" min="1" max="256" disabled={disabled} />
        </div>
        <div class="row mb-3 flex gap-2 items-center">
            <ButtonGroup class="w-auto">
                <InputAddon class="flex-shrink-0">抽帧</InputAddon>
                <Select size="sm" items={frameOps} bind:value={imgOpts.gifDropRate} class="!rounded-l-none" disabled={disabled}/>
            </ButtonGroup>
            <Checkbox bind:checked={imgOpts.gifDrawOnBefore}  disabled={disabled}>前一帧上绘制</Checkbox>
            <Label>(仅对原始图片为gif时有效)</Label>
        </div>
    {:else if imgOpts.encoder === 'webp'}
        <div class="flex mb-3 gap-2 items-center">
            {#if !imgOpts.webpLossless}
            <Label>质量:</Label>
            <Range class="w-auto" bind:value={imgOpts.webpQuality} min="1" max="100" disabled={disabled} />
            <Input class="w-fit" bind:value={imgOpts.webpQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
            {/if}
            <Checkbox bind:checked={imgOpts.webpLossless}  disabled={disabled}>无损</Checkbox>
            {#if imgOpts.webpLossless}
            <Checkbox bind:checked={imgOpts.webpRgbInTransparent}  disabled={disabled}>保留透明区域rgb</Checkbox>
            {/if}
        </div>
    {:else if imgOpts.encoder === 'png'}
        <div class="mb-3 flex flex-wrap gap-4">
            <Label>压缩等级:</Label>
            {#each pngCompress as pc}
                <Radio value={pc.value} bind:group={imgOpts.pngCompression} disabled={disabled}>{pc.name}</Radio>
            {/each}
        </div>
    {:else if imgOpts.encoder === 'avif'}
        <div class="flex flex-wrap mb-3 gap-4">
            <div class="flex items-center">
                <Label>质量:</Label>
                <Range class="w-auto" bind:value={imgOpts.avifQuality} min="1" max="100" disabled={disabled} />
                <Input class="w-fit" bind:value={imgOpts.avifQuality} size="sm" type="number" min="1" max="100" disabled={disabled} />
            </div>

            <div class="flex items-center">
                <Label>alpha通道质量:</Label>
                <Range class="w-auto" bind:value={imgOpts.avifQualityAlpha} min="1" max="100" disabled={disabled} />
                <Input class="w-fit" bind:value={imgOpts.avifQualityAlpha} size="sm" type="number" min="1" max="100" disabled={disabled} />
            </div>

            <div class="flex items-center">
                <Label>速度:</Label>
                <Range class="w-auto" bind:value={imgOpts.avifSpeed} min="1" max="10" disabled={disabled} />
                <Input class="w-fit" bind:value={imgOpts.avifSpeed} size="sm" type="number" min="1" max="10" disabled={disabled} />
            </div>
        </div>
    {/if}

    <div class="mb-3">
        <ButtonGroup class="w-full">
            <Button class="flex-shrink-0" color="green" on:click={openFolder} disabled={disabled}>
                选择保存目录
            </Button>
            <Input size="sm" bind:value={imgOpts.outPath} disabled/>
        </ButtonGroup>
    </div>

    <div class="mb-3">
        <Button outline color="blue" size="xs" on:click={transform} disabled={!(!disabled && filesNum > 0)}>
            {#if loading}
                <Spinner size="4" class="mr-3"/>
            {/if}
            转换
        </Button>
        <Button outline color="dark" size="xs" on:click={reset}  disabled={!(!disabled && filesNum > 0)}>
            {#if filesNum < 2}恢复默认宽高{:else}采用首图宽高{/if}
        </Button>
        <Button outline color="yellow" size="xs" on:click={cleanFiles} disabled={!(!disabled && filesNum > 0)}>
            清空选中文件
        </Button>
    </div>
{/if}