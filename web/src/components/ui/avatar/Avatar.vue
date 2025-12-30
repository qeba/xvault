<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
  src?: string
  alt?: string
  fallback?: string
  size?: 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
})

const sizeClasses = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'h-6 w-6 text-xs'
    case 'lg':
      return 'h-12 w-12 text-lg'
    default:
      return 'h-8 w-8 text-sm'
  }
})

const initials = computed(() => {
  if (props.fallback) return props.fallback
  if (props.alt) {
    const parts = props.alt.split(' ')
    if (parts.length >= 2 && parts[0] && parts[1]) {
      return `${parts[0][0] || ''}${parts[1][0] || ''}`.toUpperCase()
    }
    return props.alt.substring(0, 2).toUpperCase()
  }
  return '?'
})
</script>

<template>
  <span
    :class="
      cn(
        'relative flex shrink-0 overflow-hidden rounded-full',
        sizeClasses,
        props.class
      )
    "
  >
    <img
      v-if="props.src"
      :src="props.src"
      :alt="props.alt || ''"
      class="aspect-square h-full w-full object-cover"
      @error="($event.target as HTMLImageElement).style.display = 'none'"
    />
    <span
      class="flex h-full w-full items-center justify-center rounded-full bg-muted font-medium"
    >
      {{ initials }}
    </span>
  </span>
</template>
