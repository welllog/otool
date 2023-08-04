<script context="module">
  import { writable } from "svelte/store";

  let show = writable(false);
  // 'primary', 'secondary', 'success', 'danger', 'warning', 'info', 'light', 'dark'
  let alertType = "primary";
  let alertText = "";

  export function showAlert(type, text) {
    alertType = type;
    alertText = text;
    show.set(true);
  }

  export function closeAlert() {
    show.set(false);
    alertText = "";
  }
</script>

{#if $show}
  <div
    class="alert alert-{alertType} alert-dismissible fade show mt-3"
    role="alert"
  >
    {alertText}
    <button on:click={closeAlert} type="button" class="btn-close" />
  </div>
{/if}
