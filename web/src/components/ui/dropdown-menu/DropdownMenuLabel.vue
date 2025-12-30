<script setup lang="ts">
import { DropdownMenuLabel } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
}

const props = defineProps<Props>()
</script>

<template>
  <DropdownMenuLabel
    :class="cn('px-2 py-1.5 text-sm font-semibold', props.class)"
  >
    <slot />
  </DropdownMenuLabel>
</template>
