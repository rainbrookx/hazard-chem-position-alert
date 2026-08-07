<template>
  <div>
    <div>
      <el-switch v-model="autoRefresh" active-text="实时刷新" />
      <el-select v-model="refreshInterval" :disabled="!autoRefresh">
        <el-option label="1 秒" :value="1000" />
        <el-option label="5 秒" :value="5000" />
        <el-option label="10 秒" :value="10000" />
        <el-option label="30 秒" :value="30000" />
        <el-option label="60 秒" :value="60000" />
        <el-option label="暂停" :value="0" />
      </el-select>
      <el-text>共 {{ terminals.length }} 台终端</el-text>
    </div>

    <el-table :data="terminals" stripe v-loading="loading">
      <el-table-column prop="terminal_id" label="终端ID" width="140">
        <template #default="{ row }">
          <el-text>{{ row.terminal_id }}</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="person_id" label="绑定对象" width="140">
        <template #default="{ row }">
          <el-text>{{ row.person_id }}</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="battery" label="电量" width="120">
        <template #default="{ row }">
          <el-progress :percentage="row.battery" :stroke-width="8" :color="batteryColor(row.battery)" />
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
          <div>
            <el-text>{{ row.x.toFixed(1) }}, {{ row.y.toFixed(1) }}</el-text>
            <el-text v-if="row.last_x || row.last_y">{{ row.last_x.toFixed(1) }}, {{ row.last_y.toFixed(1) }}</el-text>
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
