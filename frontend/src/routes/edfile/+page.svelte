<script>
    import * as app from "$wailsjs/go/internal/App";
    import * as enc from "$wailsjs/go/srvs/Encrypt";
    import Alert, {showAlert, closeAlert} from "../Alert.svelte";

    let loading = false;
    let disabled = false;

    async function DecryptFile() {
        try {
            const pathName = await app.OpenFileDialog();
            if (pathName.length === 0) return;

            const savePath = await app.OpenDirectoryDialog();
            if (savePath.length === 0) {
                showAlert("danger", "please select save folder");
                return
            }

            loading = true;
            await enc.EncryptFile(pathName, savePath, "123456");
            return "success"
        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            loading = false;
        }
    }

    function selectFile() {
        closeAlert()
        disabled = true
        DecryptFile()
            .then((res) => {if (res === "success") showAlert("success", "encrypt success")})
            .finally(() => disabled = false)
    }
</script>

<div class="container-fluid">
    <Alert/>

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
</div>
