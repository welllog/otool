<script>
    import { OpenFileDialog } from "../../../wailsjs/go/internal/app.js";
    import Alert, { showAlert } from "../Alert.svelte";

    function selectFile(e) {
        e.preventDefault();

        console.log(e)

        OpenFileDialog()
            .then(pathName => {
                console.log(pathName)
                // 将文件名写入file input输入框
                document.getElementById("formFileLg").value = pathName;
            })
            .catch(err => {showAlert("danger", err.toString())});
    }

</script>

<div class="container-fluid">
    <Alert />

    <div>
        <button on:click={selectFile} type="button" class="btn btn-outline-secondary btn-lg">选择文件</button>
    </div>
</div>
