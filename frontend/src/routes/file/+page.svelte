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

<div class="mb-3 mt-1">
    <Label>编码</Label>
    <div class="flex flex-wrap gap-3">
        {#each encOpts as encOpt}
            <Radio value={encOpt.value} bind:group={opt} onchange={toggleSecret}>{encOpt.name}</Radio>
        {/each}
    </div>
</div>

<div class="mb-3">
    <Label>解码</Label>
    <div class="flex flex-wrap gap-3">
        {#each decOpts as decOpt}
            <Radio value={decOpt.value} bind:group={opt} onchange={toggleSecret}>{decOpt.name}</Radio>
        {/each}
    </div>
</div>

{#if showSecret}
    <div class="mb-3">
        <Label for="secretKey">密钥</Label>
        <Input size="sm" type="text" bind:value={secretKey} id="secretKey" />
    </div>
{/if}

<ButtonGroup class="w-full mb-3">
    <Button class="flex-shrink-0" color="green" onclick={openFile}>
        {#if loading}
            <Spinner size="4" class="mr-3"/>
        {/if}
        选择文件
    </Button>
    <Input class="!rounded-l-none" bind:value={inputFile} disabled/>
</ButtonGroup>

<div class="mb-3">
    <Button
        onclick={transform}
        disabled={disabled || inputFile.length === 0}
        color="blue" outline size="xs"
    >
        {#if loading}
            <Spinner size="4" class="mr-3"/>
        {/if}
        转换
    </Button>
</div>

{#if showOutput}
    <Label>输出文本</Label>
    <Textarea id="outputText" bind:value={outputText}>
        {#snippet footer()}
            <div class="text-xs text-gray-500 dark:text-gray-100" >
                <span class="text-red-500 dark:text-red-500">{outputLen}</span> chars
            </div>
        {/snippet}
    </Textarea>
{/if}

{#if showProgress }
    <Progressbar progress={curProgress} size="h-4" labelInside />
{/if}

<Modal bind:open={popupModal} size="xs" autoclose title="确认进行加密?">
    <div class="text-center">
        <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">文件加密后丢失密钥将无法恢复,请做好备份</h3>
        <Button size="xs" color="red" class="mr-2" onclick={transformEnc}>确认</Button>
        <Button size="xs" color="alternative">放弃</Button>
    </div>
</Modal>
