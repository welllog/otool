<script>
    import {
        Dropzone,
        Textarea,
        Input,
        Button,
        ButtonGroup,
        InputAddon,
        Spinner
    } from 'flowbite-svelte';
    import { onMount } from "svelte";
    import { fade, fly } from 'svelte/transition';
    import { toast } from "$lib/ToastContainer.svelte";
    import Label from "$lib/Label.svelte";
    import Card from "$lib/Card.svelte";
    import CopyButton from "$lib/CopyButton.svelte";
    import * as image from "wjs/go/srvs/Image";
    import * as app from "wjs/go/internal/App";

    let activeTab = $state('decode'); // 'decode' or 'encode'
    let text = $state('');
    let outPath = $state('');
    let fileName = $state('');
    let disabled = $state(false);
    let loading = $state(false);
    let recover = $state(1);
    let size = $state(256);

    let recoverOptions = [
        {value: 0, name: 'L (7%)'},
        {value: 1, name: 'M (15%)'},
        {value: 2, name: 'H (25%)'},
        {value: 3, name: 'Q (30%)'},
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
        if (!text) {
            toast('warning', '请先输入二维码内容');
            return;
        }
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
        event.preventDefault();
        event.stopPropagation();
        
        const files = event.dataTransfer?.files;
        if (files && files.length > 0) {
            const file = files[0];
            fileName = file.name;
            toast('info', `正在读取文件: ${fileName}`);
            decodeQr(file);
        } else {
            toast('warning', '未检测到有效文件');
        }
    }

    function handleDragOver(/** @type {DragEvent} */ event) {
        event.preventDefault();
        event.stopPropagation();
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
        loading = true;

        const reader = new FileReader();
        reader.onload = function (e) {
            const buffer = new Uint8Array(/** @type {ArrayBuffer} */ (e.target?.result));
            const body = Array.from(buffer);

            toast('info', '正在解析二维码内容...');
            image.QrDecode(body).then((content) => {
                if (content) {
                    text = content;
                    toast('success', '二维码解析成功');
                } else {
                    toast('warning', '未能识别出二维码内容');
                }
            }).catch((err) => {
                toast('danger', '识别失败: ' + err.toString());
            }).finally(() => {
                disabled = false;
                loading = false;
            })
        };
        reader.onerror = () => {
            toast('danger', '文件读取失败');
            disabled = false;
            loading = false;
        };
        reader.readAsArrayBuffer(file);
    }
</script>

<div class="flex flex-col gap-8 max-w-5xl mx-auto py-10">
    <!-- Premium Custom Tabs -->
    <div class="flex justify-center">
        <div class="inline-flex p-1.5 bg-gray-100/50 dark:bg-gray-800/50 backdrop-blur-xl rounded-[20px] border border-gray-200/50 dark:border-white/5 shadow-inner">
            <button 
                onclick={() => activeTab = 'decode'}
                class="px-8 py-3 rounded-[16px] text-sm font-black transition-all duration-300 flex items-center gap-2
                {activeTab === 'decode' ? 'bg-white dark:bg-gray-700 text-primary-600 dark:text-primary-400 shadow-xl' : 'text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}"
            >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" /></svg>
                解析二维码
            </button>
            <button 
                onclick={() => activeTab = 'encode'}
                class="px-8 py-3 rounded-[16px] text-sm font-black transition-all duration-300 flex items-center gap-2
                {activeTab === 'encode' ? 'bg-white dark:bg-gray-700 text-primary-600 dark:text-primary-400 shadow-xl' : 'text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}"
            >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                生成二维码
            </button>
        </div>
    </div>

    {#if activeTab === 'decode'}
        <div in:fly={{ y: 20, duration: 500 }} class="space-y-8">
            <Card title="上传二维码图片" icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>`}>
                <div class="p-2">
                    <div
                        role="button"
                        tabindex="0"
                        ondrop={handleDrop}
                        ondragover={handleDragOver}
                        class="border-4 border-dashed border-gray-100 dark:border-gray-800 rounded-[32px] bg-gray-50/30 dark:bg-gray-900/20 hover:bg-white dark:hover:bg-gray-800/50 hover:border-primary-500/30 transition-all duration-500 py-20 group relative overflow-hidden flex flex-col items-center justify-center cursor-pointer"
                        onclick={() => document.getElementById('fileInput')?.click()}
                        onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') document.getElementById('fileInput')?.click(); }}
                    >
                        <input type="file" id="fileInput" class="hidden" accept="image/*" onchange={handleChange} />
                        <div class="absolute inset-0 bg-gradient-to-br from-primary-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                        
                        {#if fileName.length === 0}
                            <div class="flex flex-col items-center gap-6 relative z-10">
                                <div class="w-20 h-20 bg-primary-50 dark:bg-primary-900/30 rounded-[28px] flex items-center justify-center text-primary-600 dark:text-primary-400 group-hover:scale-110 group-hover:rotate-6 transition-all duration-500 shadow-lg shadow-primary-500/10">
                                    <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
                                </div>
                                <div class="text-center space-y-2">
                                    <p class="text-xl font-black text-gray-800 dark:text-white">拖入或点击上传二维码</p>
                                    <p class="text-xs font-bold text-gray-400 uppercase tracking-widest">支持 JPG, PNG, WEBP, GIF</p>
                                </div>
                            </div>
                        {:else}
                            <div class="flex flex-col items-center gap-6 relative z-10">
                                <div class="w-20 h-20 bg-green-50 dark:bg-green-900/30 rounded-[28px] flex items-center justify-center text-green-600 dark:text-green-400 {loading ? 'animate-pulse' : 'animate-bounce'}">
                                    <svg class="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
                                </div>
                                <div class="text-center space-y-1">
                                    <p class="text-lg font-black text-gray-900 dark:text-white truncate max-w-xs">{fileName}</p>
                                    <p class="text-[10px] font-black text-green-500 uppercase tracking-widest">{loading ? '正在解析...' : '解析完成'}</p>
                                </div>
                                <button 
                                    class="text-xs font-bold text-primary-600 hover:underline z-20"
                                    onclick={(e) => { e.stopPropagation(); fileName = ''; text = ''; }}
                                >
                                    重新选择
                                </button>
                            </div>
                        {/if}
                    </div>
                </div>
            </Card>

            <Card title="解析内容" icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>`}>
                {#snippet extra()}
                    {#if text} <CopyButton text={text} /> {/if}
                {/snippet}
                <div class="relative">
                    <Textarea class="w-full font-mono text-base font-bold bg-gray-50/30 dark:bg-gray-900/20 border-2 border-gray-100 dark:border-gray-800 rounded-3xl p-6 focus:ring-4 focus:ring-primary-500/10 transition-all" bind:value={text} rows={6} readonly placeholder="解析结果将在此处显示..."/>
                    {#if loading}
                        <div class="absolute inset-0 bg-white/50 dark:bg-black/50 backdrop-blur-sm rounded-3xl flex items-center justify-center">
                            <Spinner size="8" color="blue" />
                        </div>
                    {/if}
                </div>
            </Card>
        </div>
    {:else}
        <div in:fly={{ y: 20, duration: 500 }} class="space-y-8">
            <Card title="生成配置" icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>`}>
                <div class="space-y-8">
                    <div class="space-y-4">
                        <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400">二维码内容 (Link or Text)</Label>
                        <Textarea class="w-full font-mono text-base font-bold bg-gray-50/30 dark:bg-gray-900/20 border-2 border-gray-100 dark:border-gray-800 rounded-3xl p-6 focus:ring-4 focus:ring-primary-500/10 transition-all" bind:value={text} disabled={disabled} rows={4} placeholder="在此处输入内容..."/>
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-10">
                        <div class="space-y-5">
                            <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400">容错率等级</Label>
                            <div class="grid grid-cols-2 gap-4">
                                {#each recoverOptions as op}
                                    <button 
                                        onclick={() => recover = op.value}
                                        class="flex items-center justify-center p-4 rounded-2xl border-2 font-black text-xs transition-all
                                        {recover === op.value ? 'bg-primary-600 border-primary-600 text-white shadow-lg shadow-primary-500/20' : 'bg-gray-50/30 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200'}"
                                    >
                                        {op.name}
                                    </button>
                                {/each}
                            </div>
                        </div>

                        <div class="space-y-5">
                            <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400">输出尺寸 (Pixels)</Label>
                            <div class="flex items-center gap-3 bg-gray-50/30 dark:bg-gray-900/20 p-2 pl-6 rounded-2xl border-2 border-gray-100 dark:border-gray-800 shadow-inner">
                                <input type="number" bind:value={size} class="flex-1 bg-transparent border-none focus:ring-0 font-mono text-2xl font-black text-gray-900 dark:text-white" />
                                <span class="bg-gray-200 dark:bg-gray-700 px-6 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest">PX</span>
                            </div>
                        </div>
                    </div>

                    <div class="pt-8 border-t border-gray-100 dark:border-gray-800/50 space-y-5">
                        <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400">保存位置</Label>
                        <div class="flex items-center gap-2 bg-gray-50/30 dark:bg-gray-900/20 p-2 pl-6 rounded-2xl border-2 border-gray-100 dark:border-gray-800 shadow-inner overflow-hidden">
                            <span class="flex-1 text-sm text-gray-500 truncate font-mono font-bold">{outPath || '请选择目录'}</span>
                            <Button size="sm" onclick={openFolder} class="rounded-xl px-8 bg-primary-600 hover:bg-primary-700 shadow-lg shadow-primary-500/20 text-white border-none font-black text-xs">选择</Button>
                        </div>
                    </div>

                    <div class="flex justify-center pt-6">
                        <Button
                            onclick={genQrcode}
                            disabled={disabled || text.length === 0}
                            class="px-20 py-5 bg-primary-600 hover:bg-primary-700 text-white rounded-full font-black text-lg tracking-tight shadow-2xl shadow-primary-500/30 active:scale-95 transition-all flex items-center gap-4"
                        >
                            {#if loading}
                                <Spinner size="5" class="mr-2" color="white"/>
                            {:else}
                                <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
                            {/if}
                            立即生成并保存
                        </Button>
                    </div>
                </div>
            </Card>
        </div>
    {/if}
</div>
