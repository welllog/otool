<script>
  import { Version, OpenURL } from "wjs/go/internal/App";
  import { onMount } from "svelte";

  let version = '';

  function openUrl(e) {
      e.preventDefault();
      OpenURL(e.target.href);
  }

  onMount(() => {
      Version().then((v) => {
          version = v;
      });
  });
</script>

<div class="mt-16 w-full h-full text-gray-800 dark:text-gray-200 flex flex-col items-center">
    <h1 class="text-4xl font-medium">OTOOL</h1>
    <h2 class="text-2xl">Version {version}</h2>
    <p>
        If this project has been helpful to you, please don't hesitate to give it a star.
    </p>
    <p>Visit <a on:click={openUrl} class="hover:text-primary-700" href="https://github.com/welllog/otool">project address</a></p>
</div>
