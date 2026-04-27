<script>
    import * as app from "wjs/go/internal/App";
    import * as enc from "wjs/go/srvs/Encrypt";
    import {EventsOn,EventsOff} from 'wjs/runtime/runtime';
    import {onDestroy} from 'svelte';
    import { toast } from "$lib/ToastContainer.svelte";
    import Label from "$lib/Label.svelte";
    import { Textarea , Radio, Input, InputAddon, Button, ButtonGroup, Spinner, Modal, Progressbar } from "flowbite-svelte";

    let loading = $state(false);
    let disabled = $state(false);
    let showSecret = $state(true);
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
            toast("danger", "请选择文件");
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
            toast("danger", "请填写密钥")
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
                }
            })
        } catch (/** @type {any} */ err) {
            toast("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    async function decryptFile() {
        if (secretKey.length === 0) {
            toast("danger", "请填写密钥");
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
                }
            })
        } catch (/** @type {any} */ err) {
            toast("danger", err.toString());
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

    function toggleSecret(/** @type {Event} */ e) {
        outputText = '';

        if (["encrypt", "decrypt"].includes(/** @type {HTMLInputElement} */ (e.target).value)) {
            showSecret = true;
            showOutput = false;
            return;
        }
        showOutput = true;
        showSecret = false;
        secretKey = '';
    }

    onDestroy(() => {
        progressClose();
    })

    let encOpts = [
        { name: "加密", value: "encrypt" },
        { name: "md5", value: "md5" },
        { name: "sha1", value: "sha1" },
        { name: "sha224", value: "sha224" },
        { name: "sha256", value: "sha256" },
        { name: "sha384", value: "sha384" },
        { name: "sha512", value: "sha512" },
    ]

    let decOpts = [
        { name: "解密", value: "decrypt" },
    ];
</script>

<div class="flex flex-col gap-6 max-w-4xl mx-auto py-8">
    
    <!-- Config Section -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
        <div class="flex items-center gap-3 mb-6">
            <div class="w-10 h-10 rounded-full bg-indigo-50 dark:bg-indigo-900/30 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
            </div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">操作配置</h2>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
            <div class="p-4 bg-gray-50/50 dark:bg-gray-900/50 rounded-xl border border-gray-100 dark:border-gray-700">
                <Label class="mb-3 text-gray-700 dark:text-gray-200 font-medium flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full bg-blue-500"></div>
                    编码 / 加密 / 哈希
                </Label>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {#each encOpts as encOpt}
                        <Radio value={encOpt.value} bind:group={opt} onchange={toggleSecret} class="text-sm">{encOpt.name}</Radio>
                    {/each}
                </div>
            </div>

            <div class="p-4 bg-gray-50/50 dark:bg-gray-900/50 rounded-xl border border-gray-100 dark:border-gray-700">
                <Label class="mb-3 text-gray-700 dark:text-gray-200 font-medium flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full bg-green-500"></div>
                    解码 / 解密
                </Label>
                <div class="grid grid-cols-2 gap-3">
                    {#each decOpts as decOpt}
                        <Radio value={decOpt.value} bind:group={opt} onchange={toggleSecret} class="text-sm">{decOpt.name}</Radio>
                    {/each}
                </div>
            </div>
        </div>

        {#if showSecret}
            <div class="p-5 bg-blue-50/50 dark:bg-blue-900/10 rounded-xl border border-blue-100 dark:border-blue-800/30 transition-all">
                <Label for="secretKey" class="mb-2 text-blue-800 dark:text-blue-300 font-medium">文件密钥 (Secret Key)</Label>
                <Input size="md" type="text" bind:value={secretKey} id="secretKey" class="w-full bg-white dark:bg-gray-800 border-blue-200 dark:border-blue-800/50 focus:ring-blue-500 focus:border-blue-500" placeholder="请输入用于加解密的密钥..." />
            </div>
        {/if}
    </div>

    <!-- File Input Section -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
        <div class="flex items-center gap-3 mb-6">
            <div class="w-10 h-10 rounded-full bg-purple-50 dark:bg-purple-900/30 flex items-center justify-center text-purple-600 dark:text-purple-400">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
            </div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">源文件</h2>
        </div>

        <Label class="mb-3 text-gray-500 dark:text-gray-400 font-medium">选择需要处理的文件</Label>
        <ButtonGroup class="w-full">
            <Button class="flex-shrink-0 bg-primary-600 hover:bg-primary-700 dark:bg-primary-500 dark:hover:bg-primary-400 border-none transition-all px-6 text-white shadow-md shadow-primary-500/20 dark:shadow-primary-500/40" onclick={openFile}>
                {#if loading}
                    <Spinner size="4" class="mr-2"/>
                {/if}
                选择文件
            </Button>
            <Input class="w-full !rounded-l-none bg-gray-50 dark:bg-gray-900 text-gray-600 dark:text-gray-300 font-mono" bind:value={inputFile} disabled placeholder="未选择文件"/>
        </ButtonGroup>
    </div>

    {#if showProgress}
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
            <Label class="mb-3 text-gray-500 dark:text-gray-400 font-medium flex items-center gap-2">
                <Spinner size="4"/> 处理进度
            </Label>
            <Progressbar progress={curProgress} size="h-4" labelInside color="blue" class="rounded-full" />
        </div>
    {/if}

    <!-- Output Section -->
    {#if showOutput}
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
            <div class="flex items-center justify-between mb-4">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-full bg-teal-50 dark:bg-teal-900/30 flex items-center justify-center text-teal-600 dark:text-teal-400">
                        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                    </div>
                    <h2 class="text-xl font-semibold text-gray-900 dark:text-white">处理结果</h2>
                </div>
                <span class="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded-md">
                    长度: <span class="font-mono text-primary-600 dark:text-primary-400 font-semibold">{outputLen}</span>
                </span>
            </div>

            <Textarea id="outputText" bind:value={outputText} rows={6} class="w-full font-mono text-sm bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700" readonly placeholder="处理结果将在此处显示..." />
        </div>
    {/if}

    <!-- Action Bar -->
    <div class="flex justify-center -mt-2 mb-4 z-10 relative">
        <Button
            onclick={transform}
            disabled={disabled || inputFile.length === 0}
            color="primary"
            class="px-12 py-3 shadow-xl shadow-primary-500/20 text-white rounded-full flex items-center gap-2 transition-transform hover:scale-105 bg-gradient-to-r from-primary-600 to-primary-500 border-none"
        >
            {#if loading}
                <Spinner size="4" class="mr-2"/>
            {:else}
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
            {/if}
            <span class="text-base font-semibold tracking-wide">立即处理</span>
        </Button>
    </div>

</div>

<Modal bind:open={popupModal} size="sm" autoclose class="rounded-2xl">
    <div class="text-center p-4">
        <div class="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center text-red-600 mx-auto mb-4">
            <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
        </div>
        <h3 class="mb-2 text-xl font-bold text-gray-900 dark:text-white">确认进行加密?</h3>
        <p class="mb-6 text-sm text-gray-500 dark:text-gray-400">文件加密后丢失密钥将<span class="text-red-500 font-bold">无法恢复</span>，请务必做好备份！</p>
        <div class="flex justify-center gap-4">
            <Button color="red" class="px-6" onclick={transformEnc}>确认加密</Button>
            <Button color="alternative" class="px-6">放弃</Button>
        </div>
    </div>
</Modal>
