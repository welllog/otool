<script>
    import * as app from "$wailsjs/go/internal/App";
    import * as enc from "$wailsjs/go/srvs/Encrypt";
    import * as rt from "$wailsjs/runtime/runtime"
    import Alert, {showAlert, closeAlert} from "../Alert.svelte";

    let loading = false, disabled = false, showSecret = true, showOutput = false;
    let opt = 'encrypt', secretKey = '', outputText = '';
    let outputLen = 0
    let inputFile = '', outputPath = '', outputName = '';

    function transform() {
        closeAlert()

        if (inputFile.length === 0) {
            showAlert("danger", "input file is empty");
            return
        }

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

    async function encryptFile() {
        try {
            if (secretKey.length === 0) {
                throw new Error("secret key is empty");
            }

            loading = true;
            await enc.EncryptFile(inputFile, outputPath, outputName, secretKey);
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

            loading = true;
            await enc.DecryptFile(inputFile, outputPath, outputName, secretKey);
            return "success";
        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    async function hashFile(op) {
        try {
            loading = true;
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

            outputLen = outputText.length
            return "success";
        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    async function openFile() {
        closeAlert();

        try {
            disabled = true;

            const pathName = await app.OpenFileDialog();
            if (pathName.length === 0) return;
            inputFile = pathName

            switch (opt) {
                case "encrypt":
                    [outputPath, outputName] = await enc.DefaultEncryptFilePath(pathName)
                    break;
                case "decrypt":
                    [outputPath, outputName] = await enc.DefaultDecryptFilePath(pathName)
                    break;
            }

        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            disabled = false;
        }
    }

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            outputPath = path;

        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            disabled = false;
        }
    }

    function toggleSecret(e) {
        closeAlert();
        outputText = '';
        outputLen = 0;

        if (["encrypt", "decrypt"].indexOf(e.target.value) >= 0) {
            if (inputFile.length > 0) {
                if (e.target.value === "encrypt") {
                    enc.DefaultEncryptFilePath(inputFile)
                        .then(([path, name]) => {
                            if (outputPath.length === 0) {
                                outputPath = path;
                            }
                            outputName = name;
                        })
                        .catch((err) => {
                            showAlert("danger", err.toString());
                        });
                } else if (e.target.value === "decrypt") {
                    enc.DefaultDecryptFilePath(inputFile)
                        .then(([path, name]) => {
                            if (outputPath.length === 0) {
                                outputPath = path;
                            }
                            outputName = name;
                        })
                        .catch((err) => {
                            showAlert("danger", err.toString());
                        });
                }
            }
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
            <input bind:value={secretKey} class:disabled={disabled} class="form-control" id="secretKey" />
        </div>
    {/if}

    <div class="mt-3 input-group">
        <button
            on:click={openFile}
            class:disabled={disabled}
            type="button"
            class="btn btn-outline-secondary btn-sm"
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
        <input type="text" bind:value={inputFile} disabled="true"  class="form-control">
    </div>

    {#if !showOutput}
    <div class="mt-3 input-group">
        <button
            on:click={openFolder}
            class:disabled={disabled}
            type="button"
            class="btn btn-outline-secondary btn-sm"
        >
        {#if loading}
        <span
            class="spinner-border spinner-border-sm"
            role="status"
            aria-hidden="true"
        />
        {/if}
            选择保存目录
        </button>
        <input type="text" bind:value={outputPath} disabled="true"  class="form-control">
        <span class="input-group-text">保存文件名</span>
        <input type="text" bind:value={outputName} class:disabled={disabled}  class="form-control">
    </div>
    {/if}

    <button
        on:click={transform}
        class:disabled={!(!disabled && inputFile.length > 0)}
        type="button"
        class="btn btn-outline-primary btn-sm mt-3"
    >
        {#if loading}
        <span
            class="spinner-border spinner-border-sm"
            role="status"
            aria-hidden="true"
        />
        {/if}
        转换
    </button>

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
