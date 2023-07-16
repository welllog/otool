<script>
    import 'bootstrap/dist/css/bootstrap.min.css'
    import 'bootstrap/dist/js/bootstrap.bundle.min.js'
    import Menu from './Menu.svelte'

</script>

<div class="main d-flex">
    <Menu />
    <div class="flex-fill overflow-scroll">
        <slot></slot>
    </div>
</div>

<style>
    .main {
        height: 100vh;
        width: 100vw;
        overflow-x: auto;
        overflow-y: hidden;
    }
</style>