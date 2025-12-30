<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
}

const props = defineProps<Props>()

const computedClass = computed(() =>
  cn('font-semibold leading-none tracking-tight', props.class)
)
</script>

<template>
  <h3 :class="computedClass">
    <slot />
  </h3>
</template>
