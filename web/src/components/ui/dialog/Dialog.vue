<script setup lang="ts">
import { ref, watch, computed } from 'vue'

interface Props {
  open?: boolean
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  size: 'md',
})

const sizeClasses = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'max-w-sm'
    case 'md':
      return 'max-w-md'
    case 'lg':
      return 'max-w-3xl'
    case 'xl':
      return 'max-w-[90vw] w-[90vw]'
    case 'full':
      return 'max-w-[95vw] w-full'
    default:
      return 'max-w-md'
  }
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
</script>

<template>
  <!-- Backdrop -->
  <Teleport to="body">
    <div v-if="internalOpen" class="fixed inset-0 z-50 bg-black/50" @click="close" />

    <!-- Dialog -->
    <div v-if="internalOpen" class="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div
        :class="['relative bg-background rounded-lg shadow-lg w-full max-h-[90vh] overflow-y-auto', sizeClasses]"
        role="dialog"
        @click.stop
      >
        <slot :close="close" />
      </div>
    </div>
  </Teleport>
</template>
