<script>
    export let color = 'gray';
    export let defaultClass = 'text-sm font-medium';
    import { twMerge } from 'tailwind-merge';
    const colorClasses = {
        gray: 'text-gray-500 dark:text-gray-300',
        green: 'text-green-700 dark:text-green-500',
        red: 'text-red-700 dark:text-red-500',
        disabled: 'text-gray-400 dark:text-gray-500'
    };

    let colorClass = colorClasses[color] || colorClasses.gray;
</script>

<label {...$$restProps} class={twMerge(defaultClass, colorClass, $$props.class)}><slot /></label>
