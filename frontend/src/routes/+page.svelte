<script>
  import * as app from "wjs/go/internal/App";
  import { onMount } from "svelte";

  let version = '';

  function openUrl(e) {
      e.preventDefault();
      app.OpenURL(e.target.href);
  }

  onMount(() => {
      app.Version().then((v) => {
          version = v;
      });
  });
</script>

<div class="container-fluid d-flex flex-column justify-content-center align-items-center" style="height: 100%">
    <h1>otool</h1>
    <h2>Version {version}</h2>
    <p>
        If this project has been helpful to you, please don't hesitate to give it a star.
    </p>
    <p>Visit <a on:click={openUrl} href="https://github.com/welllog/otool">project address</a></p>
</div>
