<template>
  <div class="terminal-view">
    <div class="toolbar">
      <el-switch v-model="autoRefresh" active-text="实时刷新" />
      <el-select
        v-model="refreshInterval"
        :disabled="!autoRefresh"
        style="width: 140px; margin-left: 12px"
      >
        <el-option label="1 秒" :value="1000" />
        <el-option label="5 秒" :value="5000" />
        <el-option label="10 秒" :value="10000" />
        <el-option label="30 秒" :value="30000" />
        <el-option label="60 秒" :value="60000" />
        <el-option label="暂停" :value="0" />
      </el-select>
      <span class="terminal-count">共 {{ terminals.length }} 台终端</span>
    </div>

    <el-table :data="terminals" stripe v-loading="loading" class="data-table">
      <el-table-column prop="terminal_id" label="终端ID" width="140">
        <template #default="{ row }">
          <span>{{ row.terminal_id }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="person_id" label="绑定对象" width="140">
        <template #default="{ row }">
          <span>{{ row.person_id }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="battery" label="电量" width="120">
        <template #default="{ row }">
          <el-progress
            :percentage="row.battery"
            :stroke-width="8"
            :color="batteryColor(row.battery)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="online" label="在线状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.online ? 'success' : 'danger'" size="small">
            {{ row.online ? '在线' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="当前位置 (X, Y)" width="180">
        <template #default="{ row }">
          <div class="coord-cell">
            <span class="coord-current">{{ row.x.toFixed(1) }}, {{ row.y.toFixed(1) }}</span>
            <span v-if="row.last_x || row.last_y" class="coord-last"
              >{{ row.last_x.toFixed(1) }}, {{ row.last_y.toFixed(1) }}</span
            >
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { fetchTerminals, type TerminalInfo } from '@/api/terminal'

const terminals = ref<TerminalInfo[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
const refreshInterval = ref(5000)
let timer: ReturnType<typeof setInterval> | null = null

function batteryColor(level: number): string {
  if (level <= 20) return '#F56C6C'
  if (level <= 50) return '#E6A23C'
  return '#67C23A'
}

async function load() {
  loading.value = true
  try {
    const data = await fetchTerminals()
    terminals.value = data.terminals || []
  } catch (e) {
    console.error('Failed to load terminals', e)
  } finally {
    loading.value = false
  }
}

function startTimer() {
  stopTimer()
  if (!autoRefresh.value || refreshInterval.value === 0) return
  timer = setInterval(load, refreshInterval.value)
}

function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

watch([autoRefresh, refreshInterval], startTimer, { immediate: false })
onUnmounted(stopTimer)

load()
startTimer()
</script>

<style scoped>
.terminal-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.toolbar {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  margin-bottom: 12px;
}
.terminal-count {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.data-table {
  flex: 1;
  width: 100%;
}
.coord-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.coord-current {
  color: #67c23a;
  font-family: monospace;
}
.coord-last {
  color: #909399;
  font-family: monospace;
  font-size: 12px;
}
</style>
