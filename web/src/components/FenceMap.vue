<template>
  <div ref="containerRef" class="fence-map-container">
    <svg :viewBox="viewBox" class="fence-svg">
      <!-- 背景网格 -->
      <defs>
        <pattern id="grid" width="50" height="50" patternUnits="userSpaceOnUse">
          <path d="M 50 0 L 0 0 0 50" fill="none" stroke="#e0e0e0" stroke-width="0.5" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#grid)" />

      <!-- 围栏多边形 -->
      <g v-for="fence in fences" :key="fence.zone_id">
        <polygon
          :points="polygonPoints(fence.vertices)"
          :fill="fillColor(fence.type)"
          :stroke="strokeColor(fence.type)"
          stroke-width="2"
          fill-opacity="0.15"
          class="fence-polygon"
          :class="{ selected: fence.zone_id === selectedId }"
          @click="$emit('select', fence.zone_id)"
        />
        <!-- 顶点圆（可拖拽） -->
        <circle
          v-for="(v, idx) in fence.vertices"
          :key="idx"
          :cx="v.x"
          :cy="v.y"
          r="4"
          :fill="strokeColor(fence.type)"
          stroke="white"
          stroke-width="1"
          class="vertex-circle"
          @mousedown.prevent="startDrag(fence.zone_id, idx, $event)"
        />
        <!-- 围栏名称标签 -->
        <text
          :x="labelX(fence.vertices)"
          :y="labelY(fence.vertices)"
          text-anchor="middle"
          font-size="11"
          fill="#333"
        >
          {{ fence.name }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface Point {
  x: number
  y: number
}
interface FenceInfo {
  zone_id: string
  name: string
  type: number
  vertices: Point[]
  is_active: boolean
}

const props = defineProps<{
  fences: FenceInfo[]
  selectedId: string | null
}>()

const emit = defineEmits<{
  select: [id: string]
  'vertex-move': [fenceId: string, vertexIndex: number, x: number, y: number]
}>()

const containerRef = ref<HTMLElement | null>(null)
const viewBox = '0 0 800 600'

function polygonPoints(vertices: Point[]): string {
  return vertices.map((v) => `${v.x},${v.y}`).join(' ')
}

function strokeColor(type: number): string {
  switch (type) {
    case 1:
      return '#F56C6C' // forbidden red
    case 2:
      return '#E6A23C' // restricted orange
    case 3:
      return '#67C23A' // safe green
    default:
      return '#999'
  }
}

function fillColor(type: number): string {
  return strokeColor(type)
}

function labelX(vertices: Point[]): number {
  if (vertices.length === 0) return 0
  return vertices.reduce((s, v) => s + v.x, 0) / vertices.length
}

function labelY(vertices: Point[]): number {
  if (vertices.length === 0) return 0
  return vertices.reduce((s, v) => s + v.y, 0) / vertices.length
}

function startDrag(fenceId: string, vertexIndex: number, event: MouseEvent) {
  const svg = containerRef.value?.querySelector('svg')
  if (!svg) return

  const onMove = (e: MouseEvent) => {
    const rect = svg.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * 800
    const y = ((e.clientY - rect.top) / rect.height) * 600
    emit('vertex-move', fenceId, vertexIndex, Math.round(x), Math.round(y))
  }

  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}
</script>

<style scoped>
.fence-map-container {
  width: 100%;
  height: 600px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;
  background: #fafafa;
}
.fence-svg {
  width: 100%;
  height: 100%;
}
.fence-polygon {
  cursor: pointer;
}
.fence-polygon.selected {
  stroke-width: 4;
}
.vertex-circle {
  cursor: pointer;
}
.vertex-circle:hover {
  r: 6;
}
</style>
