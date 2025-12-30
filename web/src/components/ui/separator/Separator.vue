<script setup lang="ts">
import { Separator as SeparatorRoot } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
  orientation?: 'horizontal' | 'vertical'
  decorative?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  orientation: 'horizontal',
  decorative: true,
})
</script>

<template>
  <SeparatorRoot
    :orientation="props.orientation"
    :decorative="props.decorative"
    :class="
      cn(
        'shrink-0 bg-border',
        props.orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
        props.class
      )
    "
  />
</template>
