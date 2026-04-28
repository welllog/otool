<script>
    import * as enc from "wjs/go/srvs/Encrypt.js";
    import { toast } from "$lib/ToastContainer.svelte";
    import { fade, fly, slide } from 'svelte/transition';
    import Label from "$lib/Label.svelte";
    import Card from "$lib/Card.svelte";
    import CopyButton from "$lib/CopyButton.svelte";
    import { Textarea, Checkbox, Input, Button } from "flowbite-svelte";

    let checkHmac = $state(false);
    let inputText = $state('');
    let opt = $state('');
    let secretKey = $state('');
    let secretVisible = $state(false);
    let outputText = $state('');

    let inputLen = $derived(inputText.length);
    let outputLen = $derived(outputText.length);

    let showSecret = $derived(opt === 'opensslAesEnc' || opt === 'opensslAesDec');
    let showHmac = $derived(isHash(opt));

    function transform() {
        if (!inputText) {
            toast('warning', '请输入源文本');
            return;
        }

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
                    toast('success', '已转换为大写');
                    return;
                case "lower":
                    outputText = inputText.toLowerCase();
                    toast('success', '已转换为小写');
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
                    toast('danger', '请先选择一个操作');
                    return;
            }
        }

        res
            .then((res) => {
                outputText = res;
                toast('success', '处理完成');
            })
            .catch((err) => {
                toast('danger', err.toString());
            });
    }

    function replaceInput() {
        if (!outputText) return;
        inputText = outputText;
        toast('success', '结果已填入输入框');
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
        { name: "AES-256", value: "opensslAesEnc", group: "加密" },
        { name: "Base64", value: "base64Enc", group: "编码" },
        { name: "URL", value: "urlEnc", group: "编码" },
        { name: "Hex", value: "hexEnc", group: "编码" },
        { name: "Octal", value: "octEnc", group: "编码" },
        { name: "Unicode", value: "unicodeEnc", group: "编码" },
        { name: "UTF-16", value: "utf16Enc", group: "编码" },
        { name: "UPPER", value: "upper", group: "转换" },
        { name: "MD5", value: "md5", group: "哈希" },
        { name: "SHA1", value: "sha1", group: "哈希" },
        { name: "SHA256", value: "sha256", group: "哈希" },
        { name: "SHA512", value: "sha512", group: "哈希" }
    ];

    let decOpts = [
        { name: "AES-256", value: "opensslAesDec", group: "解密" },
        { name: "Base64", value: "base64Dec", group: "解码" },
        { name: "URL", value: "urlDec", group: "解码" },
        { name: "Hex", value: "hexDec", group: "解码" },
        { name: "Octal", value: "octDec", group: "解码" },
        { name: "Unicode", value: "unicodeDec", group: "解码" },
        { name: "UTF-16", value: "utf16Dec", group: "解码" },
        { name: "lower", value: "lower", group: "转换" },
    ];
</script>

<div class="flex flex-col gap-8 max-w-6xl mx-auto py-10">
    <!-- Input Section -->
    <Card 
        title="输入文本" 
        icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>`}
    >
        {#snippet extra()}
            <span class="text-[10px] px-3 py-1 bg-gray-100 dark:bg-gray-800 text-gray-500 rounded-full font-black uppercase tracking-widest">
                LEN: {inputLen}
            </span>
        {/snippet}

        <div class="p-2 relative group">
            <div class="absolute inset-0 bg-primary-500/5 opacity-0 group-hover:opacity-100 transition-opacity rounded-[24px] pointer-events-none"></div>
            <Textarea 
                id="inputText" 
                bind:value={inputText} 
                rows={6} 
                class="w-full font-mono text-base font-bold bg-gray-50/30 dark:bg-gray-900/20 border-2 border-gray-100 dark:border-gray-800 rounded-[24px] p-6 focus:ring-4 focus:ring-primary-500/10 transition-all placeholder:text-gray-300 dark:placeholder:text-gray-600" 
                placeholder="在此粘贴需要处理的内容..." 
            />
        </div>
    </Card>

    <!-- Operations Section -->
    <Card 
        title="配置与操作" 
        icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" /></svg>`}
    >
        <div class="space-y-10">
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-10">
                <!-- Encoders -->
                <div class="space-y-4">
                    <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 flex items-center gap-2">
                        <div class="w-1.5 h-1.5 rounded-full bg-blue-500"></div>
                        编码 / 加密 / 哈希
                    </Label>
                    <div class="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                        {#each encOpts as e}
                            <button 
                                onclick={() => opt = e.value}
                                class="flex flex-col items-center justify-center p-3.5 rounded-2xl border-2 transition-all duration-200
                                {opt === e.value 
                                    ? 'bg-primary-600 border-primary-600 text-white shadow-xl shadow-primary-500/20 active:scale-95' 
                                    : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                            >
                                <span class="text-[10px] font-bold opacity-60 uppercase mb-1">{e.group}</span>
                                <span class="text-xs font-black uppercase tracking-tight">{e.name}</span>
                            </button>
                        {/each}
                    </div>
                </div>

                <!-- Decoders -->
                <div class="space-y-4">
                    <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-gray-400 flex items-center gap-2">
                        <div class="w-1.5 h-1.5 rounded-full bg-green-500"></div>
                        解码 / 解密 / 转换
                    </Label>
                    <div class="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                        {#each decOpts as d}
                            <button 
                                onclick={() => opt = d.value}
                                class="flex flex-col items-center justify-center p-3.5 rounded-2xl border-2 transition-all duration-200
                                {opt === d.value 
                                    ? 'bg-green-600 border-green-600 text-white shadow-xl shadow-green-500/20 active:scale-95' 
                                    : 'bg-gray-50/50 dark:bg-gray-800/30 border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-200 dark:hover:border-gray-700'}"
                            >
                                <span class="text-[10px] font-bold opacity-60 uppercase mb-1">{d.group}</span>
                                <span class="text-xs font-black uppercase tracking-tight">{d.name}</span>
                            </button>
                        {/each}
                    </div>
                </div>
            </div>

            <!-- Contextual Options -->
            {#if showHmac}
                <div in:slide class="p-5 bg-gray-50/50 dark:bg-gray-800/30 rounded-[24px] border-2 border-gray-100 dark:border-gray-800 flex items-center justify-between">
                    <div class="flex items-center gap-4">
                        <div class="w-10 h-10 rounded-xl bg-primary-50 dark:bg-primary-900/20 flex items-center justify-center text-primary-600">
                            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>
                        </div>
                        <div>
                            <p class="text-sm font-black text-gray-900 dark:text-white">附加 HMAC 签名</p>
                            <p class="text-xs text-gray-400">使用密钥对生成的哈希值进行数字签名</p>
                        </div>
                    </div>
                    <Checkbox bind:checked={checkHmac} class="text-primary-600 scale-125" />
                </div>
            {/if}

            {#if showSecret || (showHmac && checkHmac)}
                <div in:fly={{ y: 10, duration: 300 }} class="p-6 bg-primary-50/30 dark:bg-primary-900/10 rounded-[24px] border-2 border-primary-100/50 dark:border-primary-900/30">
                    <div class="flex items-center justify-between mb-3">
                        <Label class="text-[10px] font-black uppercase tracking-[0.2em] text-primary-600">处理密钥 (Secret Key)</Label>
                        <button 
                            onclick={() => secretVisible = !secretVisible}
                            class="text-[10px] font-bold text-primary-500 hover:text-primary-700 transition-colors uppercase tracking-widest"
                        >
                            {secretVisible ? '隐藏内容' : '显示内容'}
                        </button>
                    </div>
                    <input 
                        type={secretVisible ? 'text' : 'password'} 
                        bind:value={secretKey} 
                        placeholder="在此输入密钥..." 
                        class="w-full bg-white dark:bg-gray-800 border-none rounded-xl py-3 px-4 font-mono text-lg font-black focus:ring-4 focus:ring-primary-500/10 transition-all placeholder:text-gray-300 dark:placeholder:text-gray-600" 
                    />
                </div>
            {/if}

            <div class="flex flex-col sm:flex-row gap-4 justify-between items-center pt-8 border-t border-gray-100 dark:border-gray-800/50">
                <div class="flex gap-2">
                    <Button color="alternative" onclick={clean} class="rounded-2xl py-3 px-6 font-bold border-none bg-gray-100 dark:bg-gray-800">
                        清空数据
                    </Button>
                    <Button color="alternative" onclick={replaceInput} class="rounded-2xl py-3 px-6 font-bold border-none bg-gray-100 dark:bg-gray-800 flex items-center gap-2">
                        结果回填
                        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16V4m0 0L3 8m4-4l4 4" /></svg>
                    </Button>
                </div>
                
                <Button 
                    onclick={transform} 
                    class="w-full sm:w-auto px-20 py-4 bg-primary-600 hover:bg-primary-700 text-white rounded-full font-black text-lg tracking-tight shadow-2xl shadow-primary-500/30 active:scale-95 transition-all flex items-center gap-4"
                >
                    <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"><path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
                    立即执行处理
                </Button>
            </div>
        </div>
    </Card>

    <!-- Output Section -->
    <Card 
        title="处理结果" 
        icon={`<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>`}
    >
        {#snippet extra()}
            <div class="flex items-center gap-3">
                <span class="text-[10px] font-black text-gray-400 uppercase bg-gray-100 dark:bg-gray-800 px-3 py-1 rounded-full">LEN: {outputLen}</span>
                <CopyButton text={outputText} />
            </div>
        {/snippet}

        <div class="p-2 relative group">
            <div class="absolute inset-0 bg-primary-500/5 opacity-0 group-hover:opacity-100 transition-opacity rounded-[24px] pointer-events-none"></div>
            <Textarea 
                readonly 
                bind:value={outputText} 
                rows={8} 
                class="w-full font-mono text-base font-bold bg-gray-50/30 dark:bg-gray-900/20 border-2 border-gray-100 dark:border-gray-800 rounded-[24px] p-6 focus:ring-0 transition-all" 
                placeholder="处理结果将在此处显示..." 
            />
        </div>
    </Card>
</div>
