<script>
    import Cropper from 'cropperjs';
    import 'cropperjs/dist/cropper.css';
    import { Button, Spinner } from 'flowbite-svelte';
    import Label from '$lib/Label.svelte';

    let {
        open = false,
        file = null,
        oncancel,
        onconfirm,
    } = $props();

    /** @type {string} */
    let imageUrl = $state('');
    /** @type {boolean} */
    let imgLoaded = $state(false);
    /** @type {boolean} */
    let loading = $state(false);
    /** @type {HTMLImageElement} */
    let imgEl = $state();
    /** @type {Cropper | null} */
    let cropper = null;
    let ready = $state(false); // cropper 是否初始化完成

    // 打开时把文件读成 data URL；EXIF 旋转由 cropperjs checkOrientation 处理，
    // 其 getData() 返回已旋转正立空间的坐标，与后端 imaging.AutoOrientation 一致
    $effect(() => {
        if (open && file) {
            loading = true;
            imageUrl = '';
            imgLoaded = false;
            const reader = new FileReader();
            reader.onload = (e) => {
                imageUrl = /** @type {string} */ (e.target?.result);
                loading = false;
            };
            reader.readAsDataURL(file.file);
        }
    });

    // 图片加载完成后初始化裁剪器（自由比例）
    $effect(() => {
        if (open && imageUrl && imgEl && imgLoaded) {
            if (cropper) {
                cropper.destroy();
                cropper = null;
            }
            cropper = new Cropper(imgEl, {
                viewMode: 1,
                autoCropArea: 0.8,
                background: false,
                checkOrientation: true,
            });
            ready = true;
            return () => {
                ready = false;
                if (cropper) {
                    cropper.destroy();
                    cropper = null;
                }
            };
        }
    });

    function confirm() {
        if (!cropper) return;
        const d = cropper.getData();
        onconfirm?.({
            x: Math.round(d.x),
            y: Math.round(d.y),
            width: Math.round(d.width),
            height: Math.round(d.height),
        });
    }
</script>

{#if open}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
        onclick={(e) => { if (e.target === e.currentTarget) oncancel?.(); }}
    >
        <div
            class="w-full max-w-4xl bg-white dark:bg-gray-900 rounded-2xl shadow-2xl p-6 space-y-4"
            onclick={(e) => e.stopPropagation()}
        >
            <div class="flex items-center justify-between gap-4">
                <Label class="text-sm font-black uppercase tracking-wider text-gray-700 dark:text-gray-200">自由裁剪</Label>
                <span class="text-xs text-gray-400 font-mono truncate">{file?.file?.name}</span>
            </div>

            <div class="relative w-full h-[60vh] bg-gray-100 dark:bg-gray-800 rounded-xl overflow-hidden">
                {#if imageUrl}
                    <img
                        bind:this={imgEl}
                        src={imageUrl}
                        alt="裁剪预览"
                        class="block max-w-full"
                        onload={() => (imgLoaded = true)}
                    />
                {:else if loading}
                    <div class="absolute inset-0 flex items-center justify-center">
                        <Spinner size="8" color="blue" />
                    </div>
                {/if}
            </div>

            <div class="flex items-center justify-end gap-3">
                <span class="mr-auto text-xs text-gray-400">拖拽或缩放选择裁剪区域</span>
                <Button color="light" onclick={oncancel}>取消</Button>
                <Button color="primary" onclick={confirm} disabled={!ready}>确定裁剪</Button>
            </div>
        </div>
    </div>
{/if}
