<script>
  import { onMount } from "svelte";

  let menus = [
    { name: "文本编解码", url: "/text", selected: false },
    { name: "文件编解码", url: "/file", selected: false },
    { name: "进制转换", url: "/conversion", selected: false },
    { name: "图片转换", url: "/image", selected: false },
    { name: "二维码工具", url: "/qrcode", selected: false },
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

<div class="flex-shrink-0 text-bg-dark p-2 overflow-scroll w-25" style="max-width: 220px;">
  <a
    href="/"
    on:click={selectedMenu}
    class="d-flex justify-content-center text-white text-decoration-none"
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
          class="nav-link text-white d-flex justify-content-center"
          class:active={menuItem.selected}
          on:click={selectedMenu}
        >
          {menuItem.name}
        </a>
      </li>
    {/each}
  </ul>
</div>
