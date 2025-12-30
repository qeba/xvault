<script setup lang="ts">
import { DropdownMenuItem } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
  destructive?: boolean
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  destructive: false,
  disabled: false,
})

const emit = defineEmits<{
  select: [event: Event]
}>()
</script>

<template>
  <DropdownMenuItem
    :disabled="props.disabled"
    :class="
      cn(
        'relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors',
        'focus:bg-accent focus:text-accent-foreground',
        'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        props.destructive && 'text-destructive focus:bg-destructive/10 focus:text-destructive',
        props.class
      )
    "
    @select="emit('select', $event)"
  >
    <slot />
  </DropdownMenuItem>
</template>
