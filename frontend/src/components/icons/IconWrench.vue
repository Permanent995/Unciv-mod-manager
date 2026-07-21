<script setup lang="ts">
withDefaults(defineProps<{ size?: number; class?: string }>(), { size: 20 })
</script>
<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
    <path d="M11.42 15.17l2.414-2.414m0 0a5.25 5.25 0 004.272-4.272l-2.414 2.414a1.5 1.5 0 01-2.121 0l-2.121-2.121a1.5 1.5 0 010-2.121l2.414-2.414a5.252 5.252 0 00-4.272 4.272m0 0l-2.414 2.414m0 0a1.5 1.5 0 000 2.121l2.121 2.121a1.5 1.5 0 002.121 0z" />
  </svg>
</template>
