<script>
  import * as app from "$wailsjs/go/internal/App";

  function openUrl(e) {
      e.preventDefault();
      app.OpenURL(e.target.href);
  }
</script>

<div class="container-fluid d-flex flex-column justify-content-center align-items-center" style="height: 100%">
    <h1>Welcome to otool</h1>
    <p>
        Visit <a on:click={openUrl} href="https://github.com/welllog/otool">project address</a> to read the
        documentation
    </p>
</div>
