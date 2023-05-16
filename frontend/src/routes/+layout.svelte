<script>
    import 'bootstrap/dist/css/bootstrap.min.css'
    import 'bootstrap/dist/js/bootstrap.bundle.min.js'
    import Menu from './Menu.svelte'

</script>

<div class="d-flex main">
    <Menu />
    <div class="flex-grow-1">
        <slot></slot>
    </div>
</div>

<style>
    .main {
        height: 100vh;
        /*max-height: 100vh;*/
        overflow-x: auto;
        overflow-y: hidden;
    }
</style>