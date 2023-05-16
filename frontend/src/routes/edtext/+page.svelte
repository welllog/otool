<script>
    import * as otool from "../../../wailsjs/go/main/App.js";
    import Alert, {showAlert, closeAlert} from '../Alert.svelte';

    let showSecret = false;
    let inputText, opt, secretKey, outputText;

    function trans() {
        closeAlert();

        let res;
        switch (opt) {
            case "opensslEnc":
                res = otool.OpenSSLAesEnc(inputText, secretKey);
                break;
            case "md5":
                res = otool.Md5(inputText);
                break;
            case "sha1":
                res = otool.Sha1(inputText);
                break;
            case "sha256":
                res = otool.Sha256(inputText)
                break;
            case "base64Enc":
                res = otool.Base64Enc(inputText)
                break;
            case "urlEnc":
                res = otool.UrlEnc(inputText)
                break;
            case "opensslDec":
                res = otool.OpenSSLAesDec(inputText, secretKey);
                break;
            case "base64Dec":
                res = otool.Base64Dec(inputText)
                break;
            case "urlDec":
                res = otool.UrlDec(inputText)
                break;
            default:
                showAlert("warning", "未知操作")
                return;
        }

        res
            .then(res => {outputText = res})
            .catch(err => {showAlert("danger", err.toString())})

    }

    function insteadIO() {
        inputText = outputText;
    }

    function toggleSecret(e) {
        if (["opensslEnc", "opensslDec"].indexOf(e.target.value) >= 0) {
            showSecret = true;
            return;
        }
        showSecret = false;
    }

    let encOpts = [{
        name: "openssl加密",
        value: "opensslEnc"
    }, {
        name: "md5",
        value: "md5"
    }, {
        name: "sha1",
        value: "sha1"
    }, {
        name: "sha256",
        value: "sha256"
    }, {
        name: "base64编码",
        value: "base64Enc"
    }, {
        name: "url编码",
        value: "urlEnc"
    }];

    let decOpts = [{
        name: "openssl解密",
        value: "opensslDec"
    }, {
        name: "base64解码",
        value: "base64Dec"
    }, {
        name: "url解码",
        value: "urlDec"
    }];

</script>

<div class="container-fluid">
    <Alert />

    <div class="mb-3">
        <label for="inputText" class="form-label">加解密文本</label>
        <textarea bind:value={inputText} class="form-control" id="inputText" rows="3"></textarea>
    </div>
    加密：
    {#each encOpts as encOpt}
        <div class="form-check form-check-inline mb-3">
            <input on:change={toggleSecret} bind:group={opt} class="form-check-input" type="radio" name="opt" id={encOpt.value} value={encOpt.value}>
            <label class="form-check-label" for={encOpt.value}>{encOpt.name}</label>
        </div>
    {/each}
    <br/>
    解密：
    {#each decOpts as decOpt}
        <div class="form-check form-check-inline mb-3">
            <input on:change={toggleSecret} bind:group={opt} class="form-check-input" type="radio" name="opt" id={decOpt.value} value={decOpt.value}>
            <label class="form-check-label" for={decOpt.value}>{decOpt.name}</label>
        </div>
    {/each}
    {#if showSecret}
        <div class="mb-3">
            <label for="secretKey" class="form-label">密钥</label>
            <input bind:value={secretKey} class="form-control" id="secretKey" >
        </div>
    {/if}
    <br/>
    <button on:click={trans} type="button" class="btn btn-primary mb-3">转换</button>
    <button on:click={insteadIO} type="button" class="btn btn-primary mb-3">输出替换输入</button>
    <div class="mb-3">
        <label for="outputText" class="form-label">输出文本</label>
        <textarea bind:value={outputText} class="form-control" id="outputText" rows="3"></textarea>
    </div>
</div>
