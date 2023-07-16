<script>
    let activeItem = null;
    let menus = [
        { name: "文本编解码", url: "/text" },
        { name: "文件编解码", url: "/file" },
        { name: "进制转换", url: "/conversion" },
        { name: "图片转换", url: "/image" },
        { name: "二维码工具", url: "/qrcode" },
    ];

    function setActive(index) {
        activeItem = index;
    }
</script>

<div class="flex-shrink-0 text-bg-dark p-2 overflow-scroll w-25" style="max-width: 220px;">
    <a
        href="/"
        on:click={() => setActive(-1)}
        class="d-flex justify-content-center text-white text-decoration-none"
    >
        <span class="fs-4">otool</span>
    </a>
    <hr />
    <ul class="nav nav-pills flex-column mb-auto">
        {#each menus as item, index}
            <li class="nav-item">
                <a
                    href={item.url}
                    class="nav-link text-white d-flex justify-content-center"
                    class:active={activeItem === index}
                    on:click={() => setActive(index)}
                >
                    {item.name}
                </a>
            </li>
        {/each}
    </ul>
</div>
