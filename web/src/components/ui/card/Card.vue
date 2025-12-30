<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
}

const props = defineProps<Props>()

const computedClass = computed(() =>
  cn('rounded-xl border bg-card text-card-foreground shadow', props.class)
)
</script>

<template>
  <div :class="computedClass">
    <slot />
  </div>
</template>
