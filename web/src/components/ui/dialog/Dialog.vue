<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  open?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
})

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
}>()

const internalOpen = ref(props.open)

watch(() => props.open, (newVal) => {
  internalOpen.value = newVal
})

watch(internalOpen, (newVal) => {
  emit('update:open', newVal)
})

function close() {
  internalOpen.value = false
}

const computedClass = computed(() =>
  cn(
    'relative z-50',
    props.class
  )
)
</script>

<template>
  <div v-if="internalOpen" :class="computedClass">
    <!-- Backdrop -->
    <div class="fixed inset-0 bg-black/50" @click="close" />

    <!-- Dialog -->
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        class="relative bg-background rounded-lg shadow-lg w-full max-w-md max-h-[90vh] overflow-y-auto"
        role="dialog"
        @click.stop
      >
        <slot :close="close" />
      </div>
    </div>
  </div>
</template>
