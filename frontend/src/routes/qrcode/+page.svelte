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

    let text = $state('');
    let outPath = $state('');
    let fileName = $state('');
    let disabled = $state(false);
    let loading = $state(false);
    let recover = $state(1);
    let size = $state(256);

    let recoverOptions = [
        {value: 0, name: 'L(7%)'},
        {value: 1, name: 'M(15%)'},
        {value: 2, name: 'H(25%)'},
        {value: 3, name: 'Q(30%)'},
    ]

    onMount(() => {
        app.DefaultPath().then((path) => {outPath = path})
    });

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            outPath = path;

        } catch (/** @type {any} */ err) {
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
        } catch (/** @type {any} */ e) {
            toast("danger", e.toString());
        } finally {
            disabled = false;
            loading = false;
        }
    }

    function handleDrop(/** @type {DragEvent} */ event) {
        fileName = '';
        event.preventDefault();
        const dt = event.dataTransfer;
        if (!dt) return;
        if (dt.items) {
            const item = Array.from(dt.items).find(item => item.kind === 'file');
            if (item) {
                const file = item.getAsFile();
                if (file) {
                    fileName = file.name;
                    decodeQr(file);
                }
            }
        } else if (dt.files.length > 0) {
            const file = dt.files[0];
            fileName = file.name;
            decodeQr(file);
        }
    }

    function handleDragOver(/** @type {DragEvent} */ event) {
        event.preventDefault();
    }

    function handleChange(/** @type {Event} */ event) {
        const files = /** @type {HTMLInputElement} */ (event.target).files;
        if (files && files.length > 0) {
            fileName = files[0].name;
            decodeQr(files[0])
        }
    }

    function decodeQr(/** @type {File} */ file) {
        text = '';
        disabled = true;
        loading = true

        let reader = new FileReader();
        reader.onloadend = function (e) {
            if (e.target?.readyState === FileReader.DONE) {
                let buffer = new Uint8Array(/** @type {ArrayBuffer} */ (e.target?.result));
                let body = Array.from(buffer);

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

<div class="flex flex-col gap-6 max-w-4xl mx-auto py-8">
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-2 sm:p-6">
        <Tabs tabStyle="underline" contentClass="p-4 mt-4 bg-transparent">
            <!-- Decode Tab -->
            <TabItem open title="二维码解析">
                
                <div class="flex flex-col gap-6">
                    <div class="w-full">
                        <Dropzone
                            id="dropzone"
                            ondrop={handleDrop}
                            ondragover={handleDragOver}
                            onchange={handleChange}
                            disabled={disabled}
                            class="border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-2xl bg-gray-50 dark:bg-gray-900/50 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors py-12"
                        >
                            <svg aria-hidden="true" class="mb-4 w-12 h-12 text-gray-400 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
                            {#if fileName.length === 0}
                                <p class="mb-2 text-base text-gray-600 dark:text-gray-300"><span class="font-semibold text-primary-600 dark:text-primary-400">点击选择图片</span> 或将图片拖拽到此处</p>
                                <p class="text-sm text-gray-500 dark:text-gray-500">支持 PNG, JPG, WEBP, GIF 等常用图片格式</p>
                            {:else}
                                <div class="flex flex-col items-center gap-2">
                                    <div class="p-3 bg-green-100 dark:bg-green-900/30 rounded-full text-green-600 dark:text-green-400">
                                        <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                        </svg>
                                    </div>
                                    <p class="font-medium text-gray-800 dark:text-gray-200">{fileName}</p>
                                </div>
                            {/if}
                        </Dropzone>
                    </div>

                    <div class="w-full">
                        <Label class="mb-3 text-gray-700 dark:text-gray-300 font-medium">解析结果</Label>
                        <Textarea class="w-full font-mono text-sm bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 rounded-xl" bind:value={text} rows={6} disabled placeholder="解析结果将在此处显示..."/>
                    </div>
                </div>
            </TabItem>

            <!-- Encode Tab -->
            <TabItem title="生成二维码">
                
                <div class="flex flex-col gap-6">
                    <div>
                        <Label class="mb-3 text-gray-700 dark:text-gray-300 font-medium">二维码内容</Label>
                        <Textarea class="w-full font-mono text-sm bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 rounded-xl" bind:value={text} disabled={disabled} rows={4} placeholder="请输入要生成二维码的文本、链接..."/>
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="p-5 bg-gray-50/50 dark:bg-gray-900/50 rounded-xl border border-gray-100 dark:border-gray-700">
                            <Label class="mb-4 text-gray-700 dark:text-gray-300 font-medium flex items-center gap-2">
                                <div class="w-1.5 h-1.5 rounded-full bg-blue-500"></div>
                                容错率
                            </Label>
                            <div class="grid grid-cols-2 gap-y-3">
                                {#each recoverOptions as op}
                                    <Radio value={op.value} bind:group={recover} class="text-sm">{op.name}</Radio>
                                {/each}
                            </div>
                        </div>

                        <div class="p-5 bg-gray-50/50 dark:bg-gray-900/50 rounded-xl border border-gray-100 dark:border-gray-700">
                            <Label class="mb-4 text-gray-700 dark:text-gray-300 font-medium flex items-center gap-2">
                                <div class="w-1.5 h-1.5 rounded-full bg-purple-500"></div>
                                尺寸设置
                            </Label>
                            <ButtonGroup class="w-full pt-1">
                                <Input size="sm" type="number" bind:value={size} class="w-full bg-white dark:bg-gray-800 text-center font-mono" />
                                <InputAddon class="bg-gray-100 dark:bg-gray-700 px-4">px</InputAddon>
                            </ButtonGroup>
                        </div>
                    </div>

                    <div class="p-5 bg-gray-50/50 dark:bg-gray-900/50 rounded-xl border border-gray-100 dark:border-gray-700">
                        <Label class="mb-4 text-gray-700 dark:text-gray-300 font-medium flex items-center gap-2">
                            <div class="w-1.5 h-1.5 rounded-full bg-green-500"></div>
                            保存位置
                        </Label>
                        <ButtonGroup class="w-full">
                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                            <InputAddon class="flex-shrink-0 cursor-pointer bg-primary-600 text-white hover:bg-primary-700 border-none transition-colors px-6" onclick={openFolder}>
                                选择目录
                            </InputAddon>
                            <Input bind:value={outPath} class="w-full !rounded-l-none bg-white dark:bg-gray-800" disabled/>
                        </ButtonGroup>
                    </div>

                    <div class="flex justify-center mt-4 mb-2">
                        <Button
                            onclick={genQrcode}
                            disabled={disabled || text.length === 0}
                            color="primary" class="px-12 py-3 shadow-xl shadow-primary-500/20 text-white rounded-full flex items-center gap-2 transition-transform hover:scale-105 bg-gradient-to-r from-primary-600 to-primary-500 border-none"
                        >
                            {#if loading}
                                <Spinner size="4" class="mr-2"/>
                            {:else}
                                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                                </svg>
                            {/if}
                            <span class="text-base font-semibold tracking-wide">生成并保存</span>
                        </Button>
                    </div>
                </div>
            </TabItem>
        </Tabs>
    </div>
</div>
