<script>
    import * as enc from "$wailsjs/go/srvs/encrypt.js";
    import Alert, { showAlert, closeAlert } from "../Alert.svelte";

    let showSecret = false;
    let inputText, opt, secretKey, outputText;
    let inputLen = 0,
        outputLen = 0;

    function transform() {
        closeAlert();
        inputLen = inputText.length;

        let res;
        switch (opt) {
            case "opensslAesEnc":
                res = enc.OpenSSLAesEnc(inputText, secretKey);
                break;
            case "md5":
                res = enc.Md5(inputText);
                break;
            case "sha1":
                res = enc.Sha1(inputText);
                break;
            case "sha256":
                res = enc.Sha256(inputText);
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
            case "upper":
                outputText = inputText.toUpperCase();
                outputLen = outputText.length;
                return;
            case "lower":
                outputText = inputText.toLowerCase();
                outputLen = outputText.length;
                return;
            default:
                showAlert("warning", "未知操作");
                return;
        }

        res
            .then((res) => {
                outputText = res;
                outputLen = outputText.length;
            })
            .catch((err) => {
                showAlert("danger", err.toString());
            });
    }

    function replaceInput() {
        inputText = outputText;
    }

    function toggleSecret(e) {
        if (["opensslAesEnc", "opensslAesDec"].indexOf(e.target.value) >= 0) {
            showSecret = true;
            return;
        }
        showSecret = false;
    }

    function clean() {
        inputText = "";
        outputText = "";
        secretKey = "";
        inputLen = 0;
        outputLen = 0;
    }

    let encOpts = [
        {
            name: "AES-256-CBC加密",
            value: "opensslAesEnc",
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
            name: "sha256",
            value: "sha256",
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
            name: "大写",
            value: "upper",
        },
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
            name: "小写",
            value: "lower",
        },
    ];
</script>

<div class="container-fluid">
    <Alert />

    <div class="mb-3 mt-3">
        <label for="inputText" class="form-label">编解码文本</label>
        <textarea
            bind:value={inputText}
            class="form-control"
            id="inputText"
            rows="3"
            aria-describedby="inputHelp"
        />
        {#if inputLen > 0}
            <div id="inputHelp" class="form-text">
                <span class="text-danger">{inputLen}</span> chars
            </div>
        {/if}
    </div>
    编码：
    {#each encOpts as encOpt}
        <div class="form-check form-check-inline">
            <input
                on:change={toggleSecret}
                bind:group={opt}
                class="form-check-input"
                type="radio"
                name="opt"
                id={encOpt.value}
                value={encOpt.value}
            />
            <label class="form-check-label" for={encOpt.value}>{encOpt.name}</label>
        </div>
    {/each}
    <hr />
    解码：
    {#each decOpts as decOpt}
        <div class="form-check form-check-inline">
            <input
                on:change={toggleSecret}
                bind:group={opt}
                class="form-check-input"
                type="radio"
                name="opt"
                id={decOpt.value}
                value={decOpt.value}
            />
            <label class="form-check-label" for={decOpt.value}>{decOpt.name}</label>
        </div>
    {/each}
    <hr />
    {#if showSecret}
        <div class="mb-3">
            <label for="secretKey" class="form-label">密钥</label>
            <input bind:value={secretKey} class="form-control" id="secretKey" />
        </div>
    {/if}
    <button
        on:click={transform}
        type="button"
        class="btn btn-outline-primary btn-sm mb-3">转换</button
    >
    <button
        on:click={replaceInput}
        type="button"
        class="btn btn-outline-secondary btn-sm mb-3">输入替换</button
    >
    <button
        on:click={clean}
        type="button"
        class="btn btn-outline-warning btn-sm mb-3">清空</button
    >
    <div>
        <label for="outputText" class="form-label">输出文本</label>
        <textarea
            bind:value={outputText}
            class="form-control"
            id="outputText"
            rows="3"
            aria-describedby="outputHelp"
        />
        {#if outputLen > 0}
            <div id="outputHelp" class="form-text">
                <span class="text-danger">{outputLen}</span> chars
            </div>
        {/if}
    </div>
</div>
