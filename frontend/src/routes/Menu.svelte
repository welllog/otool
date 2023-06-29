<script>
  import { onMount } from "svelte";

  let menus = [
    { name: "文本编解码", url: "/edtext", selected: false },
    { name: "文件编解码", url: "/edfile", selected: false },
    { name: "进制转换", url: "/conversion", selected: false },
    { name: "图片转换", url: "/imagine", selected: false },
  ];

  onMount(() => {
    let path = window.location.pathname;
    menus.forEach((item, index) => {
      if (item.url == path) {
        item.selected = true;
      } else {
        item.selected = false;
      }
    });
    menus = menus;
  });

  function cleanMenuState() {
    menus.forEach((item, index) => {
      item.selected = false;
    });
    menus = menus;
  }

  function selectedMenu(e) {
    if (!e.target.attributes["menu-id"]) {
      cleanMenuState();
      return;
    }

    let id = e.target.attributes["menu-id"].value;
    menus.forEach((item, index) => {
      if (index == id) {
        item.selected = true;
      } else {
        item.selected = false;
      }
    });
    menus = menus;
  }
</script>

<div
  class="d-flex flex-column flex-shrink-0 p-3 text-bg-dark"
  style="width: 280px;"
>
  <a
    href="/"
    on:click={selectedMenu}
    class="d-flex align-items-center mb-3 mb-md-0 me-md-auto text-white text-decoration-none"
  >
    <svg class="bi pe-none me-2" width="40" height="32"
      ><use xlink:href="#bootstrap" /></svg
    >
    <span class="fs-4">otool</span>
  </a>
  <hr />
  <ul class="nav nav-pills flex-column mb-auto">
    {#each menus as menuItem, i}
      <li class="nav-item">
        <a
          href={menuItem.url}
          menu-id={i}
          class="nav-link text-white"
          class:active={menuItem.selected}
          on:click={selectedMenu}
        >
          <svg class="bi pe-none me-2" width="16" height="16"
            ><use xlink:href="#home" /></svg
          >
          {menuItem.name}
        </a>
      </li>
    {/each}
  </ul>
</div>

<style>
  .bi {
    vertical-align: -0.125em;
    fill: currentColor;
  }
</style>
