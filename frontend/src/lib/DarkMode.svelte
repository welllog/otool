<script>
    import { onMount } from 'svelte';
    import { WindowSetDarkTheme, WindowSetLightTheme } from 'wjs/runtime/runtime'
    import { ButtonGroup, Button } from 'flowbite-svelte';

    const themeOps = new Map([
        ['light', 0],
        ['dark', 1],
        ['auto', 2],
    ]);

    let sysTheme = 'dark';
    let theme = 'auto';
    let active = themeOps.get(theme);

    onMount(() => {
        theme = localStorage.getItem('theme') || 'auto';

        let media = window.matchMedia('(prefers-color-scheme: dark)');
        if (!media.matches) {
            sysTheme = 'light';
        }

        if (themeOps.get(theme) === null) {
            theme = 'auto';
        }

        let callback = (e) => {
            let prefersDarkMode = e.matches;
            if (prefersDarkMode) {
                sysTheme = 'dark';
                if (theme === 'auto') {
                    setDark();
                }
            } else {
                sysTheme = 'light';
                if (theme === "auto") {
                    setLight();
                }
            }
        };
        if (typeof media.addEventListener === 'function') {
            media.addEventListener('change', callback);
        } else if (typeof media.addListener === 'function') {
            media.addListener(callback);
        }

        switch (theme) {
            case 'light':
                setLight();
                break;
            case 'dark':
                setDark();
                break;
            case 'auto':
                if (sysTheme === 'dark') {
                    setDark();
                } else {
                    setLight();
                }
                break;
        }
        active = themeOps.get(theme);
    });

    const changeTheme = i => {
        active = i;
        switch (i) {
            case 0:
                theme = 'light';
                setLight();
                localStorage.setItem('theme', 'light');
                break;
            case 1:
                theme = 'dark'
                setDark();
                localStorage.setItem('theme', 'dark');
                break;
            case 2:
                theme = 'auto';
                if (sysTheme === 'dark') {
                    setDark();
                } else {
                    setLight();
                }
                localStorage.removeItem('theme')
                break;
        }
    };

    function setDark() {
        document.documentElement.classList.add('dark');
        WindowSetDarkTheme();
    }

    function setLight() {
        document.documentElement.classList.remove('dark');
        WindowSetLightTheme();
    }
</script>

<ButtonGroup divClass="inline-flex rounded-lg shadow-sm my-1">
    <Button size="xs" class="hover:fill-orange-500 dark:hover:fill-orange-600 {active === 0 ? 'fill-orange-500' : ''}"
        on:click={() => changeTheme(0)}
    >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="18" height="18"><path d="M12 18C8.68629 18 6 15.3137 6 12C6 8.68629 8.68629 6 12 6C15.3137 6 18 8.68629 18 12C18 15.3137 15.3137 18 12 18ZM12 16C14.2091 16 16 14.2091 16 12C16 9.79086 14.2091 8 12 8C9.79086 8 8 9.79086 8 12C8 14.2091 9.79086 16 12 16ZM11 1H13V4H11V1ZM11 20H13V23H11V20ZM3.51472 4.92893L4.92893 3.51472L7.05025 5.63604L5.63604 7.05025L3.51472 4.92893ZM16.9497 18.364L18.364 16.9497L20.4853 19.0711L19.0711 20.4853L16.9497 18.364ZM19.0711 3.51472L20.4853 4.92893L18.364 7.05025L16.9497 5.63604L19.0711 3.51472ZM5.63604 16.9497L7.05025 18.364L4.92893 20.4853L3.51472 19.0711L5.63604 16.9497ZM23 11V13H20V11H23ZM4 11V13H1V11H4Z"></path></svg>
    </Button>
    <Button size="xs" class="hover:fill-orange-500 dark:hover:fill-orange-600 {active === 1 ? 'fill-orange-500' : ''}"
        on:click={() => changeTheme(1)}
    >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="18" height="18"><path d="M10 7C10 10.866 13.134 14 17 14C18.9584 14 20.729 13.1957 21.9995 11.8995C22 11.933 22 11.9665 22 12C22 17.5228 17.5228 22 12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C12.0335 2 12.067 2 12.1005 2.00049C10.8043 3.27098 10 5.04157 10 7ZM4 12C4 16.4183 7.58172 20 12 20C15.0583 20 17.7158 18.2839 19.062 15.7621C18.3945 15.9187 17.7035 16 17 16C12.0294 16 8 11.9706 8 7C8 6.29648 8.08133 5.60547 8.2379 4.938C5.71611 6.28423 4 8.9417 4 12Z"></path></svg>
    </Button>
    <Button size="xs" class="hover:fill-orange-500 dark:hover:fill-orange-600 {active === 2 ? 'fill-orange-500' : ''}"
        on:click={() => changeTheme(2)}
    >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="18" height="18"><path d="M12 21.9966C6.47715 21.9966 2 17.5194 2 11.9966C2 6.47373 6.47715 1.99658 12 1.99658C17.5228 1.99658 22 6.47373 22 11.9966C22 17.5194 17.5228 21.9966 12 21.9966ZM12 19.9966C16.4183 19.9966 20 16.4149 20 11.9966C20 7.5783 16.4183 3.99658 12 3.99658C7.58172 3.99658 4 7.5783 4 11.9966C4 16.4149 7.58172 19.9966 12 19.9966ZM7.00035 15.3158C9.07995 15.1645 11.117 14.2938 12.7071 12.7037C14.2972 11.1136 15.1679 9.07654 15.3193 6.99694C15.6454 7.21396 15.955 7.46629 16.2426 7.75394C18.5858 10.0971 18.5858 13.8961 16.2426 16.2392C13.8995 18.5824 10.1005 18.5824 7.75736 16.2392C7.46971 15.9516 7.21738 15.642 7.00035 15.3158Z"></path></svg>
    </Button>
</ButtonGroup>