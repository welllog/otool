<script>
    import * as enc from "wjs/go/srvs/encrypt.js";
    import { toast } from "$lib/ToastContainer.svelte";
    import Label from "$lib/Label.svelte";
    import { Textarea , Radio, Checkbox, Input, Button } from "flowbite-svelte";

    let showSecret = false;
    let showHmac = false, checkHmac = false;
    let inputText = '', opt = '', secretKey = '', outputText = '';
    let inputLen = 0,
        outputLen = 0;

    $: inputLen = inputText.length;
    $: outputLen = outputText.length;
    $: {
        if (opt === 'opensslAesEnc' || opt === 'opensslAesDec') {
            showSecret = true;
            showHmac = false;
        } else if (isHash(opt)) {
            showSecret = false;
            showHmac = true;
        } else {
            showSecret = false;
            showHmac = false;
        }
    }

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
        return ["md5", "sha1", "sha224", "sha256", "sha384", "sha512", "sha512_224", "sha512_256"].indexOf(op) >= 0;
    }

    let encOpts = [
        {
            name: "AES-256-CBC加密",
            value: "opensslAesEnc",
        },
        {
            name: "base64编码",
            value: "base64Enc",
        },
        {
            name: "url编码",
            value: "urlEnc",
        },
        {
            name: "8进制编码",
            value: "octEnc",
        },
        {
            name: "16进制编码",
            value: "hexEnc",
        },
        {
            name: "unicode编码",
            value: "unicodeEnc",
        },
        {
            name: "utf16编码",
            value: "utf16Enc",
        },
        {
            name: "大写",
            value: "upper",
        },
        {
            name: "md5",
            value: "md5",
        },
        {
            name: "sha1",
            value: "sha1",
        },
        {
            name: "sha224",
            value: "sha224",
        },
        {
            name: "sha256",
            value: "sha256",
        },
        {
            name: "sha384",
            value: "sha384",
        },
        {
            name: "sha512",
            value: "sha512",
        },
        {
            name: "sha512/224",
            value: "sha512_224",
        },
        {
            name: "sha512/256",
            value: "sha512_256",
        }
    ];

    let decOpts = [
        {
            name: "AES-256-CBC解密",
            value: "opensslAesDec",
        },
        {
            name: "base64解码",
            value: "base64Dec",
        },
        {
            name: "url解码",
            value: "urlDec",
        },
        {
            name: "8进制解码",
            value: "octDec",
        },
        {
            name: "16进制解码",
            value: "hexDec",
        },
        {
            name: "unicode解码",
            value: "unicodeDec",
        },
        {
            name: "utf16解码",
            value: "utf16Dec",
        },
        {
            name: "小写",
            value: "lower",
        },
    ];
</script>

<div class="mb-3 mt-1">
    <Label for="inputText">编解码文本</Label>
    <Textarea id="inputText" bind:value={inputText}>
        <div slot="footer" class="text-xs text-gray-500 dark:text-gray-100" >
            <span class="text-red-500 dark:text-red-500">{inputLen}</span> chars
        </div>
    </Textarea>
</div>

<div class="mb-3">
    <Label>编码</Label>
    <div class="flex flex-wrap gap-3">
        {#each encOpts as encOpt}
            <Radio value={encOpt.value} bind:group={opt}>{encOpt.name}</Radio>
        {/each}
        {#if showHmac}
            <Checkbox bind:checked={checkHmac}>hmac</Checkbox>
        {/if}
    </div>
</div>

<div class="mb-3">
    <Label>解码</Label>
    <div class="flex flex-wrap gap-3">
        {#each decOpts as decOpt}
            <Radio value={decOpt.value} bind:group={opt}>{decOpt.name}</Radio>
        {/each}
    </div>
</div>

{#if showSecret || (showHmac && checkHmac)}
    <div class="mb-3">
        <Label for="secretKey">密钥</Label>
        <Input size="sm" type="text" bind:value={secretKey} id="secretKey" />
    </div>
{/if}

<div class="mb-3">
    <Button on:click={transform} outline color="blue" size="xs">
        转换
    </Button>
    <Button on:click={replaceInput} outline color="dark" size="xs">
        输入替换
    </Button>
    <Button on:click={clean} outline color="yellow" size="xs">
        清空
    </Button>
</div>

<div>
    <Label for="outputText">输出文本</Label>
    <Textarea id="outputText" bind:value={outputText}>
        <div slot="footer" class="text-xs text-gray-500 dark:text-gray-100" >
            <span class="text-red-500 dark:text-red-500">{outputLen}</span> chars
        </div>
    </Textarea>
</div>
