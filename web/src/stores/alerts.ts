import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { fetchActiveAlerts, fetchHistory, type AlertRecord, type AlertQuery } from '@/api/alerts'

export const useAlertsStore = defineStore('alerts', () => {
  const alerts = ref<AlertRecord[]>([])
  const activeFilter = ref<number | null>(null) // alert_type filter
  const total = ref(0)
  const polling = ref(false)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  const filteredAlerts = computed(() => {
    if (activeFilter.value === null) return alerts.value
    return alerts.value.filter((a) => a.alert_type === activeFilter.value)
  })

  async function loadActiveAlerts(types?: number[]) {
    const typeStrs = types?.map(String)
    const result = await fetchActiveAlerts(typeStrs)
    alerts.value = result
  }

  async function loadHistory(query: AlertQuery) {
    const result = await fetchHistory(query)
    alerts.value = result.records
    total.value = result.total
  }

  function startPolling(intervalMs: number = 5000) {
    if (pollTimer) return
    polling.value = true
    loadActiveAlerts()
    pollTimer = setInterval(() => loadActiveAlerts(), intervalMs)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    polling.value = false
  }

  return {
    alerts,
    activeFilter,
    total,
    polling,
    filteredAlerts,
    loadActiveAlerts,
    loadHistory,
    startPolling,
    stopPolling,
  }
})
