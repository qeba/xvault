<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  open?: boolean
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
</script>

<template>
  <!-- Backdrop -->
  <Teleport to="body">
    <div v-if="internalOpen" class="fixed inset-0 z-50 bg-black/50" @click="close" />

    <!-- Dialog -->
    <div v-if="internalOpen" class="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div
        class="relative bg-background rounded-lg shadow-lg w-full max-w-md max-h-[90vh] overflow-y-auto"
        role="dialog"
        @click.stop
      >
        <slot :close="close" />
      </div>
    </div>
  </Teleport>
</template>
