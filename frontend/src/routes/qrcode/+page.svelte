<script>
    import { onMount } from "svelte";
    import Alert, {showAlert, closeAlert} from "../Alert.svelte";
    import * as image from "$wailsjs/go/srvs/Image";
    import * as app from "$wailsjs/go/internal/App";

    let text = '', pathName = '', savePath = '', saveName = '';
    let disabled = false, loading = false;
    let recover = 1, size = 256;

    let recoverOptions = [
        {value: 0, name: 'L(7%)'},
        {value: 1, name: 'M(15%)'},
        {value: 2, name: 'H(25%)'},
        {value: 3, name: 'Q(30%)'},
    ]

    onMount(() => {
        app.DefaultPath().then((p) => {savePath = p})
    });

    async function openFile() {
        closeAlert();
        disabled = true;

        try {
            pathName = await image.OpenFileDialog();
            if (pathName.length === 0) {
                return;
            }

            loading = true;
            text = await image.QrDecode(pathName);

        } catch (e) {
            showAlert("danger", e.toString());
        } finally {
            disabled = false;
            loading = false;
        }
    }

    async function openFolder() {
        try {
            disabled = true;

            const path = await app.OpenDirectoryDialog();
            if (path.length === 0) return;

            savePath = path;

        } catch (err) {
            showAlert("danger", err.toString());
        } finally {
            disabled = false;
        }
    }

    async function genQrcode() {
        closeAlert();
        disabled = true;
        loading = true;

        try {
            if (saveName === '') {
                let id = await app.RandId();
                saveName = id + '.png'
            }

            await image.QrEncode(text, savePath, saveName, recover, size);
        } catch (e) {
            showAlert("danger", e.toString());
        } finally {
            disabled = false;
            loading = false;
        }
    }

    $: if (text.length >= 0) {
        app.RandId().then((id) => {
            saveName = id + '.png'
        })
    }
</script>

<div class="container-fluid">
    <Alert />

    <div class="mt-2">
        <button class="btn btn-outline-secondary" on:click={openFile} class:disabled={disabled}>
            {#if loading}
                <span
                        class="spinner-border spinner-border-sm text-primary"
                        role="status"
                        aria-hidden="true"
                />
            {/if}
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-plus-lg" viewBox="0 0 16 16">
                <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
            </svg>
            选择二维码解析
        </button>
    </div>

    <div class="mt-3">
        <label class="form-label">二维码内容</label>
        <textarea
            bind:value={text}
            class="form-control"
            id="inputText"
            rows="3"
            aria-describedby="inputHelp"
        />
    </div>

    {#if text.length > 0 }
        <div class="mt-3">
            容错率：
            {#each recoverOptions as op}
                <div class="form-check form-check-inline">
                    <input
                        bind:group={recover}
                        class="form-check-input"
                        type="radio"
                        id={op.value}
                        value={op.value}
                    />
                    <label class="form-check-label" for={op.value}>{op.name}</label>
                </div>
            {/each}
        </div>

        <div class="input-group mt-2 input-group-sm">
            <span class="input-group-text">尺寸</span>
            <input bind:value={size} type="number" class="form-control" disabled={disabled}>
            <span class="input-group-text">px</span>
        </div>

        <div class="mt-2 input-group input-group-sm">
            <button
                    on:click={openFolder}
                    disabled={disabled}
                    type="button"
                    class="btn btn-outline-secondary btn-sm"
            >
                选择保存目录
            </button>
            <input type="text" bind:value={savePath} disabled="true"  class="form-control">
        </div>

        <div class="mt-2 input-group input-group-sm">
            <span class="input-group-text">保存文件名</span>
            <input type="text" bind:value={saveName} disabled={disabled}  class="form-control">
        </div>
    {/if}

    <div class="mt-2">
        <button
            on:click={genQrcode}
            type="button"
            disabled={disabled}
            class="btn btn-outline-primary btn-sm mb-3">
            {#if loading}
                <span
                        class="spinner-border spinner-border-sm text-primary"
                        role="status"
                        aria-hidden="true"
                />
            {/if}
            生成二维码
        </button
        >
        <button
            type="button"
            disabled={disabled}
            class="btn btn-outline-secondary btn-sm mb-3">清空</button
        >
    </div>
</div>