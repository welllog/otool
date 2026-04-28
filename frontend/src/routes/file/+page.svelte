<script>
    import * as app from "wjs/go/internal/App";
    import * as enc from "wjs/go/srvs/Encrypt";
    import {EventsOn,EventsOff} from 'wjs/runtime/runtime';
    import {onDestroy} from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { toast } from "$lib/ToastContainer.svelte";
    import Label from "$lib/Label.svelte";
    import Card from "$lib/Card.svelte";
    import CopyButton from "$lib/CopyButton.svelte";
    import { Textarea, Input, Button, Spinner, Modal, Progressbar } from "flowbite-svelte";

    let loading = $state(false);
    let disabled = $state(false);
    let showSecretInput = $state(true);
    let secretVisible = $state(false);
    let showOutput = $state(false);
    let popupModal = $state(false);
    let showProgress = $state(false);
    let opt = $state('encrypt');
    let secretKey = $state('');
    let outputText = $state('');
    let outputLen = $derived(outputText.length);
    let inputFile = $state('');
    let curProgress = $state(0);
    let progressClose = () => {};

    function transform() {
        if (inputFile.length === 0) {
            toast("warning", "请先选择需要处理的文件");
            return
        }

        disabled = true;
        if (opt !== 'encrypt' && opt !== 'decrypt') {
            hashFile(opt)
                .then()
                .finally(() => disabled = false)
            return
        }

        if (opt === 'decrypt') {
            decryptFile();
            return
        }

        disabled = false;
        popupModal = true;
    }

    function transformEnc() {
        disabled = true;
        encryptFile();
    }

    async function encryptFile() {
        if (secretKey.length === 0) {
            toast("danger", "请填写文件加密密钥")
            disabled = false;
            return;
        }

        loading = true;

        try {
            let eventName = await enc.EncryptFile(inputFile, secretKey);
            curProgress = 0;
            showProgress = true;
            progressClose = EventsOn(eventName, (/** @type {number} */ progress) => {
                if (progress >= 0) {
                    curProgress = progress;
                } else {
                    EventsOff(/** @type {string} */ (eventName));
                    showProgress = false;
                    disabled = false;
                    toast("success", "文件加密处理完成");
                }
            })
        } catch (/** @type {any} */ err) {
            toast("danger", err.toString());
            disabled = false;
        } finally {
            loading = false;
        }
    }

    async function decryptFile() {
        if (secretKey.length === 0) {
            toast("danger", "请填写解密密钥");
            disabled = false;
            return;
        }

        loading = true;

        try {
            let eventName = await enc.DecryptFile(inputFile, secretKey);
            curProgress = 0;
            showProgress = true;
            progressClose = EventsOn(eventName, (/** @type {number} */ progress) => {
                if (progress >= 0) {
                    curProgress = progress;
                } else {
                    EventsOff(/** @type {string} */ (eventName));
                    showProgress = false;
                    disabled = false;
                    toast("success", "文件解密处理完成");
                }
            })
        } catch (/** @type {any} */ err) {
            toast("danger", err.toString());
            disabled = false;
        } finally {
            loading = false;
        }
    }

    async function hashFile(/** @type {string} */ op) {
        let fileName = inputFile.split("/").pop();
        loading = true;

        try {
            switch (op) {
                case "md5":
                    outputText = await enc.Md5File(inputFile);
                    break;
                case "sha1":
                    outputText = await enc.Sha1File(inputFile);
                    break;
                case "sha224":
                    outputText = await enc.Sha224File(inputFile);
                    break;
                case "sha256":
                    outputText = await enc.Sha256File(inputFile);
                    break;
                case "sha384":
                    outputText = await enc.Sha384File(inputFile);
                    break;
                case "sha512":
                    outputText = await enc.Sha512File(inputFile);
                    break;
                default:
                    throw new Error("unknown op");
            }
            showOutput = true;
            toast("success", "计算完成");
            return "success";
        } catch (/** @type {any} */ err) {
            toast("danger", op + " " + fileName + ': ' + err.toString());
        } finally {
            loading = false;
        }
    }

    async function openFile() {
        try {
            disabled = true;
            const pathName = await app.OpenFileDialog();
            if (pathName.length === 0) return;
            inputFile = pathName
        } catch (/** @type {any} */ err) {
            toast("danger", err.toString());
        } finally {
            disabled = false;
        }
    }

    function selectOpt(val) {
        opt = val;
        outputText = '';
        if (["encrypt", "decrypt"].includes(val)) {
            showSecretInput = true;
            showOutput = false;
        } else {
            showOutput = false; // Reset output when switching hash types
            showSecretInput = false;
            secretKey = '';
        }
    }

    onDestroy(() => {
        progressClose();
    })

    let encOpts = [
        { name: "AES-256 加密", value: "encrypt", color: "blue" },
        { name: "MD5", value: "md5", color: "gray" },
        { name: "SHA-1", value: "sha1", color: "gray" },
        { name: "SHA-224", value: "sha224", color: "gray" },
        { name: "SHA-256", value: "sha256", color: "gray" },
        { name: "SHA-384", value: "sha384", color: "gray" },
        { name: "SHA-512", value: "sha512", color: "gray" },
    ]

    let decOpts = [
        { name: "AES-256 解密", value: "decrypt", color: "green" },
    ];
</script>

<div class="flex flex-col gap-8 max-w-5xl mx-auto py-10">
    <!-- Config Card -->
    <Card 
        title="处理模式" 
        icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" /></svg>`}
    >
        <div class="space-y-10">
            <div class="space-y-4">
                <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.5)]"></div>
                    加密与哈希 (Encrypt & Hash)
                </Label>
                <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {#each encOpts as e}
                        <button 
                            onclick={() => selectOpt(e.value)}
                            class="group relative flex flex-col items-center justify-center p-4 rounded-2xl border-2 transition-all duration-300
                            {opt === e.value 
                                ? 'bg-primary-600 border-primary-600 text-white shadow-xl shadow-primary-500/20 active:scale-95' 
                                : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                        >
                            <span class="text-xs font-black uppercase tracking-tight">{e.name}</span>
                        </button>
                    {/each}
                </div>
            </div>

            <div class="space-y-4">
                <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.5)]"></div>
                    解密与还原 (Decrypt)
                </Label>
                <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {#each decOpts as d}
                        <button 
                            onclick={() => selectOpt(d.value)}
                            class="group relative flex flex-col items-center justify-center p-4 rounded-2xl border-2 transition-all duration-300
                            {opt === d.value 
                                ? 'bg-green-600 border-green-600 text-white shadow-xl shadow-green-500/20 active:scale-95' 
                                : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                        >
                            <span class="text-xs font-black uppercase tracking-tight">{d.name}</span>
                        </button>
                    {/each}
                </div>
            </div>

            {#if showSecretInput}
                <div in:fly={{ y: 10, duration: 300 }} class="p-6 bg-primary-50/30 dark:bg-primary-900/10 rounded-[24px] border-2 border-primary-100/50 dark:border-primary-900/30">
                    <div class="flex items-center justify-between mb-3">
                        <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-primary-600">密钥 (Secret Key)</Label>
                        <button 
                            type="button"
                            onclick={() => secretVisible = !secretVisible}
                            class="text-[10px] font-bold text-primary-500 hover:text-primary-700 transition-colors uppercase tracking-widest focus:outline-none"
                        >
                            {#key secretVisible}
                                <span in:fade={{ duration: 150 }}>{secretVisible ? '隐藏内容' : '显示内容'}</span>
                            {/key}
                        </button>
                    </div>
                    <div class="relative group">
                        <input 
                            type={secretVisible ? 'text' : 'password'} 
                            bind:value={secretKey} 
                            placeholder="在此输入加解密密钥..." 
                            class="w-full bg-white dark:bg-gray-800 border-none rounded-xl py-3 px-4 font-mono text-lg font-black focus:ring-4 focus:ring-primary-500/10 transition-all placeholder:text-gray-300 dark:placeholder:text-gray-600" 
                        />
                    </div>
                </div>
            {/if}
        </div>
    </Card>

    <!-- File Selection -->
    <Card 
        title="文件路径" 
        icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>`}
    >
        <div class="p-2 flex items-center gap-4 bg-gray-50/30 dark:bg-gray-900/20 p-2 pl-6 rounded-[24px] border-2 border-gray-100 dark:border-gray-800 shadow-inner overflow-hidden">
            <div class="flex-1 min-w-0">
                <span class="block text-xs font-black text-gray-400 uppercase tracking-widest mb-1">当前选择文件</span>
                <p class="text-sm font-bold text-gray-600 dark:text-gray-300 truncate font-mono">{inputFile || '未选择任何文件'}</p>
            </div>
            <Button size="sm" onclick={openFile} disabled={disabled} class="rounded-xl px-10 py-4 bg-white dark:bg-gray-800 text-gray-900 dark:text-white border-none shadow-lg hover:shadow-xl transition-all active:scale-95 font-black text-xs uppercase tracking-widest">
                浏览文件
            </Button>
        </div>
    </Card>

    <!-- Progress -->
    {#if showProgress}
        <div in:fade class="space-y-4 px-2">
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                    <Spinner size="4" color="blue" />
                    <span class="text-xs font-black uppercase tracking-[0.2em] text-primary-600">正在处理...</span>
                </div>
                <span class="text-xl font-black text-primary-700 dark:text-primary-400">{curProgress}%</span>
            </div>
            <div class="h-4 bg-gray-100 dark:bg-gray-800/50 rounded-full overflow-hidden p-1 border border-gray-200 dark:border-gray-700">
                <div class="h-full bg-gradient-to-r from-primary-600 to-blue-400 rounded-full transition-all duration-300 shadow-[0_0_12px_rgba(37,99,235,0.4)]" style="width: {curProgress}%"></div>
            </div>
        </div>
    {/if}

    <!-- Result -->
    {#if showOutput}
        <Card 
            title="计算结果" 
            icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>`}
        >
            {#snippet extra()}
                <div class="flex items-center gap-3">
                    <span class="text-[10px] font-black text-gray-400 uppercase bg-gray-100 dark:bg-gray-800 px-3 py-1 rounded-full">LEN: {outputLen}</span>
                    <CopyButton text={outputText} />
                </div>
            {/snippet}
            <div class="relative group">
                <div class="absolute inset-0 bg-primary-500/5 opacity-0 group-hover:opacity-100 transition-opacity rounded-3xl pointer-events-none"></div>
                <Textarea 
                    readonly 
                    bind:value={outputText} 
                    rows={4} 
                    class="w-full font-mono text-base font-bold bg-gray-50/30 dark:bg-gray-900/20 border-2 border-gray-100 dark:border-gray-800 rounded-3xl p-6 focus:ring-0 transition-all" 
                    placeholder="计算结果将在此处显示..." 
                />
            </div>
        </Card>
    {/if}

    <!-- Execution Bar -->
    <div class="flex justify-center pt-6">
        <Button
            onclick={transform}
            disabled={disabled || inputFile.length === 0}
            class="px-24 py-5 bg-primary-600 hover:bg-primary-700 text-white rounded-full font-black text-lg tracking-tight shadow-2xl shadow-primary-500/30 active:scale-95 transition-all flex items-center gap-4"
        >
            {#if loading}
                <Spinner size="5" color="white"/>
            {:else}
                <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"><path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
            {/if}
            立即开始处理
        </Button>
    </div>
</div>

<Modal bind:open={popupModal} size="sm" autoclose class="rounded-[32px] overflow-hidden">
    <div class="text-center p-8">
        <div class="w-24 h-24 rounded-[32px] bg-red-50 dark:bg-red-900/20 flex items-center justify-center text-red-500 mx-auto mb-8 transform rotate-6 shadow-xl shadow-red-500/10">
            <svg class="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
        </div>
        <h3 class="mb-4 text-3xl font-black text-gray-900 dark:text-white tracking-tight">确认加密该文件?</h3>
        <p class="mb-10 text-sm text-gray-500 dark:text-gray-400 leading-relaxed font-medium px-2">
            加密操作不可逆。如果密钥丢失，文件将<span class="text-red-500 font-black underline underline-offset-8 mx-1">永久损坏</span>。
        </p>
        <div class="flex flex-col gap-3">
            <Button color="red" class="w-full rounded-2xl py-4 font-black text-base shadow-xl shadow-red-500/20 border-none" onclick={transformEnc}>确认并开始加密</Button>
            <Button color="alternative" class="w-full rounded-2xl py-4 font-bold text-gray-500 border-none bg-gray-100 dark:bg-gray-800">取消</Button>
        </div>
    </div>
</Modal>
