<script>
    import { twMerge } from 'tailwind-merge';

    let {
        children,
        color = 'gray',
        defaultClass = 'text-sm font-normal',
        class: className = '',
        ...restProps
    } = $props();

    /** @type {Record<string, string>} */
    const colorClasses = {
        gray: 'text-gray-700 dark:text-gray-300',
        green: 'text-green-700 dark:text-green-500',
        red: 'text-red-700 dark:text-red-500',
        disabled: 'text-gray-400 dark:text-gray-500'
    };

    let colorClass = $derived(colorClasses[color] || colorClasses.gray);
</script>

<label {...restProps} class={twMerge(defaultClass, colorClass, className)}>{@render children()}</label>
