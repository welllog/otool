<script>
    import * as enc from "wjs/go/srvs/Encrypt.js";
    import { toast } from "$lib/ToastContainer.svelte";
    import Label from "$lib/Label.svelte";
    import { Textarea , Radio, Checkbox, Input, Button } from "flowbite-svelte";

    let checkHmac = $state(false);
    let inputText = $state('');
    let opt = $state('');
    let secretKey = $state('');
    let outputText = $state('');

    let inputLen = $derived(inputText.length);
    let outputLen = $derived(outputText.length);

    let showSecret = $derived(opt === 'opensslAesEnc' || opt === 'opensslAesDec');
    let showHmac = $derived(isHash(opt));

    function transform() {
        let res;
        if (showHmac && checkHmac) {
            res = enc.Hmac(inputText, secretKey, opt);
        } else {
            switch (opt) {
                case "opensslAesEnc":
                    res = enc.OpenSSLAesEnc(inputText, secretKey);
                    break;
                case "base64Enc":
                    res = enc.Base64Enc(inputText);
                    break;
                case "urlEnc":
                    res = enc.UrlEnc(inputText);
                    break;
                case "octEnc":
                    res = enc.OctEnc(inputText);
                    break;
                case "hexEnc":
                    res = enc.HexEnc(inputText);
                    break;
                case "unicodeEnc":
                    res = enc.UnicodeEnc(inputText)
                    break;
                case "utf16Enc":
                    res = enc.Utf16Enc(inputText)
                    break;
                case "opensslAesDec":
                    res = enc.OpenSSLAesDec(inputText, secretKey);
                    break;
                case "base64Dec":
                    res = enc.Base64Dec(inputText);
                    break;
                case "urlDec":
                    res = enc.UrlDec(inputText);
                    break;
                case "octDec":
                    res = enc.OctDec(inputText);
                    break;
                case "hexDec":
                    res = enc.HexDec(inputText);
                    break;
                case "unicodeDec":
                    res = enc.UnicodeDec(inputText)
                    break;
                case "utf16Dec":
                    res = enc.Utf16Dec(inputText)
                    break;
                case "upper":
                    outputText = inputText.toUpperCase();
                    return;
                case "lower":
                    outputText = inputText.toLowerCase();
                    return;
                case "md5":
                    res = enc.Md5(inputText);
                    break;
                case "sha1":
                    res = enc.Sha1(inputText);
                    break;
                case "sha224":
                    res = enc.Sha224(inputText);
                    break;
                case "sha256":
                    res = enc.Sha256(inputText);
                    break;
                case "sha384":
                    res = enc.Sha384(inputText);
                    break;
                case "sha512":
                    res = enc.Sha512(inputText);
                    break;
                case "sha512_224":
                    res = enc.Sha512_224(inputText);
                    break;
                case "sha512_256":
                    res = enc.Sha512_256(inputText);
                    break;
                default:
                    toast('danger', '未知操作');
                    return;
            }
        }

        res
            .then((res) => {
                outputText = res;
            })
            .catch((err) => {
                toast('danger', err.toString());
            });
    }

    function replaceInput() {
        inputText = outputText;
    }

    function clean() {
        inputText = "";
        outputText = "";
        secretKey = "";
    }

    function isHash(op = '') {
        return ["md5", "sha1", "sha224", "sha256", "sha384", "sha512", "sha512_224", "sha512_256"].includes(op);
    }

    let encOpts = [
        { name: "AES-256-CBC加密", value: "opensslAesEnc" },
        { name: "base64编码", value: "base64Enc" },
        { name: "url编码", value: "urlEnc" },
        { name: "8进制编码", value: "octEnc" },
        { name: "16进制编码", value: "hexEnc" },
        { name: "unicode编码", value: "unicodeEnc" },
        { name: "utf16编码", value: "utf16Enc" },
        { name: "大写", value: "upper" },
        { name: "md5", value: "md5" },
        { name: "sha1", value: "sha1" },
        { name: "sha224", value: "sha224" },
        { name: "sha256", value: "sha256" },
        { name: "sha384", value: "sha384" },
        { name: "sha512", value: "sha512" },
        { name: "sha512/224", value: "sha512_224" },
        { name: "sha512/256", value: "sha512_256" }
    ];

    let decOpts = [
        { name: "AES-256-CBC解密", value: "opensslAesDec" },
        { name: "base64解码", value: "base64Dec" },
        { name: "url解码", value: "urlDec" },
        { name: "8进制解码", value: "octDec" },
        { name: "16进制解码", value: "hexDec" },
        { name: "unicode解码", value: "unicodeDec" },
        { name: "utf16解码", value: "utf16Dec" },
        { name: "小写", value: "lower" },
    ];
</script>
<div class="flex flex-col gap-4 max-w-5xl mx-auto py-4">
    <!-- Input Section -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
        <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
                <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">源文本</h2>
            </div>
            <span class="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded-md">
                长度: <span class="font-mono text-primary-600 dark:text-primary-400 font-semibold">{inputLen}</span>
            </span>
        </div>
        <Textarea id="inputText" bind:value={inputText} rows={6} class="w-full font-mono text-sm bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700" placeholder="请输入需要处理的文本..." />
    </div>

    <!-- Operations Section -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
        <div class="flex items-center gap-2 mb-6">
            <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">操作配置</h2>
        </div>
        
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <div class="p-4 bg-gray-50/50 dark:bg-gray-900/50 rounded-lg border border-gray-100 dark:border-gray-700">
                <Label class="mb-4 text-gray-700 dark:text-gray-200 font-medium flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full bg-blue-500"></div>
                    编码 / 加密 / 哈希
                </Label>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {#each encOpts as encOpt}
                        <Radio value={encOpt.value} bind:group={opt} class="text-sm">{encOpt.name}</Radio>
                    {/each}
                </div>
                {#if showHmac}
                    <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                        <Checkbox bind:checked={checkHmac} class="text-primary-600">附加 HMAC 签名</Checkbox>
                    </div>
                {/if}
            </div>

            <div class="p-4 bg-gray-50/50 dark:bg-gray-900/50 rounded-lg border border-gray-100 dark:border-gray-700">
                <Label class="mb-4 text-gray-700 dark:text-gray-200 font-medium flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full bg-green-500"></div>
                    解码 / 解密
                </Label>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {#each decOpts as decOpt}
                        <Radio value={decOpt.value} bind:group={opt} class="text-sm">{decOpt.name}</Radio>
                    {/each}
                </div>
            </div>
        </div>

        {#if showSecret || (showHmac && checkHmac)}
            <div class="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-100 dark:border-blue-800">
                <Label for="secretKey" class="mb-2 text-blue-800 dark:text-blue-300 font-medium">密钥 (Secret Key)</Label>
                <Input size="md" type="text" bind:value={secretKey} id="secretKey" placeholder="请输入用于加密或签名的密钥..." class="w-full bg-white dark:bg-gray-800" />
            </div>
        {/if}

        <div class="mt-6 flex flex-col sm:flex-row gap-3 justify-end items-center pt-4 border-t border-gray-100 dark:border-gray-700">
            <Button color="light" onclick={clean} class="w-full sm:w-auto dark:bg-gray-900 dark:hover:bg-gray-700">
                清空数据
            </Button>
            <Button color="dark" outline onclick={replaceInput} class="w-full sm:w-auto border-gray-300 dark:border-gray-600">
                将结果作为输入
                <svg class="w-4 h-4 ml-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
                </svg>
            </Button>
            <Button color="primary" onclick={transform} class="w-full sm:w-auto px-8 shadow-md shadow-primary-500/30 text-white">
                执行处理
                <svg class="w-4 h-4 ml-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
            </Button>
        </div>
    </div>

    <!-- Output Section -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-5">
        <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
                <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">处理结果</h2>
            </div>
            <span class="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded-md">
                长度: <span class="font-mono text-primary-600 dark:text-primary-400 font-semibold">{outputLen}</span>
            </span>
        </div>
        <Textarea id="outputText" bind:value={outputText} rows={8} class="w-full font-mono text-sm bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700" readonly placeholder="处理结果将在此处显示..." />
    </div>
</div>
