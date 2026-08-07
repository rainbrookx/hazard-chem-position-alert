<template>
  <div>
    <div>
      <el-select v-model="typeFilter" placeholder="告警类型" clearable>
        <el-option label="全部类型" :value="null" />
        <el-option label="越界报警" :value="1" />
        <el-option label="静止报警" :value="2" />
        <el-option label="超员报警" :value="3" />
        <el-option label="缺员报警" :value="4" />
        <el-option label="滞留报警" :value="5" />
        <el-option label="一键报警" :value="6" />
        <el-option label="人员聚集" :value="7" />
      </el-select>
      <el-date-picker
        v-model="timeRange"
        type="datetimerange"
        range-separator="至"
        start-placeholder="开始时间"
        end-placeholder="结束时间"
      />
      <el-text>共 {{ alerts.length }} 条记录</el-text>
    </div>

    <el-table :data="filteredAlerts" stripe v-loading="loading">
      <el-table-column prop="trigger_time_ms" label="触发时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.trigger_time_ms) }}
        </template>
      </el-table-column>
      <el-table-column label="类型" width="120">
        <template #default="{ row }">
          <AlertBadge :severity="row.severity" :alert-type="row.alert_type">
            {{ alertTypeLabel(row.alert_type) }}
          </AlertBadge>
        </template>
      </el-table-column>
      <el-table-column label="等级" width="80">
        <template #default="{ row }">
          <el-tag :type="severityTag(row.severity)" size="small">{{ severityLabel(row.severity) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="person_ids" label="涉及人员" min-width="150">
        <template #default="{ row }">
          {{ (row.person_ids || []).join(', ') || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="zone_id" label="区域" width="100">
        <template #default="{ row }">
          {{ row.zone_id || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="200" />
    </el-table>

    <el-pagination
      v-if="total > pageSize"
      v-model:current-page="currentPage"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="loadHistory"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useAlertsStore } from '@/stores/alerts'
import AlertBadge from '@/components/AlertBadge.vue'
import type { AlertQuery } from '@/types'

const store = useAlertsStore()
const loading = ref(false)
const typeFilter = ref<number | null>(null)
const timeRange = ref<[Date, Date] | null>(null)
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

const alerts = computed(() => store.alerts)

const filteredAlerts = computed(() => {
  let list = store.alerts
  if (typeFilter.value !== null) {
    list = list.filter((a) => a.alert_type === typeFilter.value)
  }
  return list
})

function formatTime(ms: number): string {
  return new Date(ms).toLocaleString('zh-CN')
}

function alertTypeLabel(type: number): string {
  switch (type) {
    case 1: return '越界'
    case 2: return '静止'
    case 3: return '超员'
    case 4: return '缺员'
    case 5: return '滞留'
    case 6: return '一键'
    case 7: return '聚集'
    default: return '未知'
  }
}

function severityLabel(sev: number): string {
  switch (sev) {
    case 1: return '黄'
    case 2: return '橙'
    case 3: return '红'
    default: return '-'
  }
}

function severityTag(sev: number): string {
  switch (sev) {
    case 1: return 'warning'
    case 2: return 'warning'
    case 3: return 'danger'
    default: return 'info'
  }
}

async function loadHistory() {
  loading.value = true
  try {
    const query: AlertQuery = { limit: pageSize, offset: (currentPage.value - 1) * pageSize }
    if (typeFilter.value !== null) query.types = [String(typeFilter.value)]
    if (timeRange.value) {
      query.start_time_ms = timeRange.value[0].getTime()
      query.end_time_ms = timeRange.value[1].getTime()
    }
    await store.loadHistory(query)
    total.value = store.total
  } catch (e) {
    console.error('Failed to load alerts', e)
  } finally {
    loading.value = false
  }
}

watch(typeFilter, () => {
  currentPage.value = 1
  loadHistory()
})
watch(timeRange, () => {
  currentPage.value = 1
  loadHistory()
})

onMounted(loadHistory)
</script>
