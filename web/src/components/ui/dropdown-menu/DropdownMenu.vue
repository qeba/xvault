<script setup lang="ts">
import { ref, provide } from 'vue'
import { DropdownMenuRoot } from 'radix-vue'

const open = ref(false)
provide('dropdownOpen', open)
</script>

<template>
  <DropdownMenuRoot v-model:open="open">
    <slot />
  </DropdownMenuRoot>
</template>
