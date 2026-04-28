<script>
    import { onMount, onDestroy } from 'svelte';

    let {
        allowMultiple = false,
        acceptedFileTypes = [],
        labelFileTypeNotAllowed = '',
        disabled = false,
        onaddfile,
        onremovefile,
    } = $props();

    /** @type {HTMLInputElement} */
    let inputEl;
    /** @type {any} */
    let pond;
    let isMounted = false;

    onMount(async () => {
        isMounted = true;

        const { create, registerPlugin } = await import('filepond');
        const FilePondPluginImageExifOrientation = (await import('filepond-plugin-image-exif-orientation')).default;
        const FilePondPluginImagePreview = (await import('filepond-plugin-image-preview')).default;
        const FilePondPluginFileValidateType = (await import('filepond-plugin-file-validate-type')).default;

        await import('filepond/dist/filepond.min.css');
        await import('filepond-plugin-image-preview/dist/filepond-plugin-image-preview.min.css');

        if (!isMounted) return;

        registerPlugin(
            FilePondPluginImageExifOrientation,
            FilePondPluginImagePreview,
            FilePondPluginFileValidateType,
        );

        pond = create(inputEl, {
            allowMultiple,
            acceptedFileTypes: acceptedFileTypes.length > 0 ? acceptedFileTypes : undefined,
            labelFileTypeNotAllowed,
            disabled,
            files: [],
            onaddfile: (err, fileItem) => {
                if (onaddfile) onaddfile(err, fileItem);
            },
            onremovefile: (err, fileItem) => {
                if (onremovefile) onremovefile(err, fileItem);
            },
        });
    });

    onDestroy(() => {
        isMounted = false;
        if (pond) {
            pond.destroy();
            pond = null;
        }
    });

    $effect(() => {
        if (pond) {
            pond.setOptions({ disabled });
        }
    });

    export function getFiles() {
        return pond ? pond.getFiles() : [];
    }

    export function removeFiles() {
        if (pond) pond.removeFiles();
    }
</script>

<input type="file" bind:this={inputEl} {disabled} />
