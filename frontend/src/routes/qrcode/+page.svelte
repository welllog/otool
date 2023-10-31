<script>
    import {
        Tabs,
        TabItem,
        Dropzone,
        Textarea,
        Radio,
        Input,
        Button,
        ButtonGroup,
        InputAddon,
        Spinner
    } from 'flowbite-svelte';
    import { onMount } from "svelte";
    import { toast } from "$lib/ToastContainer.svelte";
    import Label from "$lib/Label.svelte";
    import * as image from "wjs/go/srvs/Image";
    import * as app from "wjs/go/internal/App";

    let text = '', outPath = '', fileName = '';
    let disabled = false, loading = false;
    let recover = 1, size = 256;

    let recoverOptions = [
        {value: 0, name: 'L(7%)'},
        {value: 1, name: 'M(15%)'},
        {value: 2, name: 'H(25%)'},
        {value: 3, name: 'Q(30%)'},
    ]

    onMount(() => {
        app.DefaultPath().then((p) => {outPath = p})
    });

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            outPath = path;

        } catch (err) {
            toast("danger", err.toString());
        } finally {
            disabled = false;
        }
    }

    async function genQrcode() {
        disabled = true;
        loading = true;

        try {
            let saveName = await image.QrEncode(text, outPath, recover, Number(size));
            toast('success', '生成' + saveName + '成功');
        } catch (e) {
            toast("danger", e.toString());
        } finally {
            disabled = false;
            loading = false;
        }
    }

    const dropHandle = (event) => {
        fileName = '';
        event.preventDefault();
        if (event.dataTransfer.items) {
            for (let i = 0; i < event.dataTransfer.items.length; i++) {
                if (event.dataTransfer.items[i].kind === 'file') {
                    const file = event.dataTransfer.items[i].getAsFile();
                    fileName = file.name;
                    decodeQr(file);
                    break;
                }
            }
        } else {
            for (let i = 0; i < event.dataTransfer.files.length; i++) {
                const file = event.dataTransfer.files[i];
                fileName = file.name;
                decodeQr(file);
                break;
            }
        }
    };

    const handleChange = (event) => {
        const files = event.target.files;
        if (files.length > 0) {
            fileName = files[0].name;
            decodeQr(files[0])
        }
    };

    function decodeQr(file) {
        disabled = true;
        loading = true

        let reader = new FileReader();
        reader.onloadend = function (e) {
            if (e.target.readyState === FileReader.DONE) {
                let buffer = new Uint8Array(e.target.result);
                let body = [];

                for (let i = 0; i < buffer.length; i++) {
                    body.push(buffer[i]);
                }

                image.QrDecode(body).then((content) => {
                    text = content;
                    toast('success', '解析成功');
                }).catch((err) => {
                    toast('danger', err.toString());
                }).finally(() => {
                    disabled = false;
                    loading = false;
                })
            }
        };
        reader.readAsArrayBuffer(file);
    }
</script>

<Tabs style="underline">
    <TabItem open title="二维码解析">
        <Dropzone
            id="dropzone"
            on:drop={dropHandle}
            on:dragover={(event) => {
                event.preventDefault();
            }}
            on:change={handleChange}
            bind:disabled={disabled}
        >
            <svg aria-hidden="true" class="mb-3 w-10 h-10 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
            {#if fileName.length === 0}
                <p class="mb-2 text-sm text-gray-500 dark:text-gray-400"><span class="font-semibold">Click to upload</span> or drag and drop</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">SVG, PNG, JPG or GIF (MAX. 800x400px)</p>
            {:else}
                <p>{fileName}</p>
            {/if}
        </Dropzone>
        <Label>解析结果</Label>
        <Textarea bind:value={text} disabled/>
    </TabItem>
    <TabItem title="生成二维码">
        <div class="mb-3">
            <Label>二维码内容</Label>
            <Textarea bind:value={text} bind:disabled={disabled} />
        </div>

        <div class="mb-3">
            <Label>容错率</Label>
            <div class="flex flex-wrap gap-3">
                {#each recoverOptions as op}
                    <Radio value={op.value} bind:group={recover}>{op.name}</Radio>
                {/each}
            </div>
        </div>

        <div class="mb-3">
            <Label>尺寸</Label>
            <ButtonGroup class="w-full">
                <Input type="number" bind:value={size} />
                <InputAddon>px</InputAddon>
            </ButtonGroup>
        </div>

        <div class="mb-3">
            <ButtonGroup class="mb-3 w-full">
                <Button size="xs" color="blue" on:click={openFolder}>
                    选择保存目录
                </Button>
                <Input bind:value={outPath} disabled/>
            </ButtonGroup>
        </div>

        <Button
            on:click={genQrcode}
            bind:disable={disabled}
            color="blue" outline size="xs"
        >
            {#if loading}
                <Spinner size="4" class="mr-3"/>
            {/if}
            生成二维码
        </Button>
    </TabItem>
</Tabs>