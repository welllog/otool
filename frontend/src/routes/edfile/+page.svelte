<script>
    import * as app from "$wailsjs/go/internal/App";
    import * as enc from "$wailsjs/go/srvs/Encrypt";
    import Alert, {showAlert, closeAlert} from "../Alert.svelte";

    let loading = false, disabled = false, showSecret = false, showOutput = false;
    let opt = '', secretKey = '', outputText = '';
    let outputLen = 0

    async function encryptFile() {
        try {
            if (secretKey.length === 0) {
                throw new Error("secret key is empty");
            }

            const pathName = await app.OpenFileDialog();
            if (pathName.length === 0) return;

            const savePath = await app.OpenDirectoryDialog();
            if (savePath.length === 0) {
                showAlert("danger", "please select save folder");
                return;
            }

            loading = true;
            await enc.EncryptFile(pathName, savePath, secretKey);
            return "success";
        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    async function decryptFile() {
        try {
            if (secretKey.length === 0) {
                throw new Error("secret key is empty");
            }

            const pathName = await app.OpenFileDialog();
            if (pathName.length === 0) return;

            const savePath = await app.OpenDirectoryDialog();
            if (savePath.length === 0) {
                showAlert("danger", "please select save folder");
                return;
            }

            loading = true;
            await enc.DecryptFile(pathName, savePath, secretKey);
            return "success";
        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    async function hashFile(op) {
        try {
            const pathName = await app.OpenFileDialog();
            if (pathName.length === 0) return;

            loading = true;
            switch (op) {
                case "md5":
                    outputText = await enc.Md5File(pathName);
                    break;
                case "sha1":
                    outputText = await enc.Sha1File(pathName);
                    break;
                case "sha256":
                    outputText = await enc.Sha256File(pathName);
                    break;
                default:
                    throw new Error("unknown op");
            }

            outputLen = outputText.length
            return "success";
        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    function selectFile() {
        closeAlert();
        disabled = true;

        switch (opt) {
            case "encrypt":
                encryptFile()
                    .then((res) => {if (res === "success") showAlert("success", "encrypt success")})
                    .finally(() => disabled = false)
                return
            case "decrypt":
                decryptFile()
                    .then((res) => {if (res === "success") showAlert("success", "decrypt success")})
                    .finally(() => disabled = false)
                return
            default:
                hashFile(opt)
                    .then()
                    .finally(() => disabled = false)
                return
        }
    }

    function toggleSecret(e) {
        closeAlert();
        outputText = '';
        outputLen = 0;

        if (["encrypt", "decrypt"].indexOf(e.target.value) >= 0) {
            showSecret = true;
            showOutput = false;
            return;
        }
        showOutput = true;
        showSecret = false;
        secretKey = '';
    }

    let encOpts = [
        {
            name: "加密",
            value: "encrypt",
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
        }
    ]

    let decOpts = [
        {
            name: "解密",
            value: "decrypt",
        }
    ];
</script>

<div class="container-fluid">
    <Alert/>

    编码：
    {#each encOpts as encOpt}
        <div class="form-check form-check-inline mt-3">
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

    <div class="mt-3">
        <button
                on:click={selectFile}
                class:disabled={disabled}
                type="button"
                class="btn btn-outline-secondary btn-lg"
        >
            {#if loading}
        <span
                class="spinner-border spinner-border-sm"
                role="status"
                aria-hidden="true"
        />
            {/if}
            选择文件
        </button>
    </div>

    {#if showOutput}
    <div class="mt-3">
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
    {/if}
</div>
