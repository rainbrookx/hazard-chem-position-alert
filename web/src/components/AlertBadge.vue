<template>
  <el-tag :type="tagType" :effect="effect" size="small">
    <slot />
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    severity: number
    alertType?: number
  }>(),
  { alertType: 0 },
)

const tagType = computed(() => {
  // SEVERITY_YELLOW=1, SEVERITY_ORANGE=2, SEVERITY_RED=3
  switch (props.severity) {
    case 3:
      return 'danger'
    case 2:
      return 'warning'
    case 1:
      return 'warning'
    default:
      return 'info'
  }
})

const effect = computed(() => {
  return props.severity === 2 ? 'dark' : 'light'
})
</script>
