<template>
  <div>
    <el-row :gutter="16">
      <el-col :span="8">
        <div>
          <div>
            <el-button type="primary" size="small" @click="showAddDialog">新增围栏</el-button>
          </div>
          <el-table :data="fences" stripe size="small" @row-click="selectFence" highlight-current-row>
            <el-table-column prop="name" label="名称" min-width="100" />
            <el-table-column prop="type" label="类型" width="80">
              <template #default="{ row }">
                <el-tag :type="typeTag(row.type)" size="small">{{ typeLabel(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="is_active" label="启用" width="60">
              <template #default="{ row }">
                <el-text>{{ row.is_active ? '是' : '否' }}</el-text>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button size="small" text type="primary" @click.stop="editFence(row)">编辑</el-button>
                <el-button size="small" text type="danger" @click.stop="deleteFence(row.zone_id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="16">
        <FenceMap
          :fences="fences"
          :selected-id="selectedId"
          @select="selectFenceById"
          @vertex-move="handleVertexMove"
        />
      </el-col>
    </el-row>

    <el-dialog v-model="dialogVisible" :title="editingFence ? '编辑围栏' : '新增围栏'" width="500px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="围栏名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="围栏类型">
          <el-select v-model="form.type">
            <el-option :value="1" label="禁止进入" />
            <el-option :value="2" label="受限区域" />
            <el-option :value="3" label="安全区域" />
          </el-select>
        </el-form-item>
        <el-form-item label="最大人数">
          <el-input-number v-model="form.max_people" :min="0" />
        </el-form-item>
        <el-form-item label="最少人数">
          <el-input-number v-model="form.min_people" :min="0" />
        </el-form-item>
        <el-form-item label="滞留时限(秒)">
          <el-input-number v-model="form.max_stay_seconds" :min="0" />
        </el-form-item>
        <el-form-item label="静止时限(秒)">
          <el-input-number v-model="form.stationary_seconds" :min="0" />
        </el-form-item>
        <el-form-item label="静止位移阈值(米)">
          <el-input-number v-model="form.stationary_threshold_meters" :min="0" :precision="2" :step="0.5" />
        </el-form-item>
        <el-form-item label="静止恢复去抖(秒)">
          <el-input-number v-model="form.stationary_recovery_seconds" :min="0" />
        </el-form-item>
        <el-form-item label="指定人员ID">
          <el-input v-model="form.required_person_ids" placeholder="逗号分隔的人员ID列表" />
        </el-form-item>
        <el-form-item label="聚集网格(米)">
          <el-input-number v-model="form.grid_cell_meters" :min="0" />
        </el-form-item>
        <el-form-item label="是否启用">
          <el-switch v-model="form.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveFence">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import FenceMap from '@/components/FenceMap.vue'
import { authHeader } from '@/api/auth'
import type { Point, FenceInfo } from '@/types'

const fences = ref<FenceInfo[]>([])
const selectedId = ref<string | null>(null)
const dialogVisible = ref(false)
const editingFence = ref<FenceInfo | null>(null)

const form = reactive({
  name: '',
  type: 1,
  max_people: 0,
  min_people: 0,
  max_stay_seconds: 0,
  stationary_seconds: 0,
  stationary_threshold_meters: 2.0,
  stationary_recovery_seconds: 3,
  required_person_ids: '' as string,
  grid_cell_meters: 0,
  is_active: true,
  vertices: [
    { x: 100, y: 100 },
    { x: 200, y: 100 },
    { x: 200, y: 200 },
    { x: 100, y: 200 },
  ] as Point[],
})

function typeTag(type: number): string {
  switch (type) {
    case 1: return 'danger'
    case 2: return 'warning'
    case 3: return 'success'
    default: return 'info'
  }
}

function typeLabel(type: number): string {
  switch (type) {
    case 1: return '禁止'
    case 2: return '受限'
    case 3: return '安全'
    default: return '未知'
  }
}

async function loadFences() {
  const resp = await fetch('/api/fences', { headers: authHeader() })
  if (resp.ok) {
    const data = await resp.json()
    fences.value = data.fences || []
  }
}

function selectFence(row: FenceInfo) { selectedId.value = row.zone_id }
function selectFenceById(id: string) { selectedId.value = id }

function editFence(row: FenceInfo) {
  editingFence.value = row
  Object.assign(form, {
    name: row.name,
    type: row.type,
    max_people: row.max_people,
    min_people: row.min_people,
    max_stay_seconds: row.max_stay_seconds,
    stationary_seconds: row.stationary_seconds,
    stationary_threshold_meters: row.stationary_threshold_meters ?? 2.0,
    stationary_recovery_seconds: row.stationary_recovery_seconds ?? 3,
    required_person_ids: (row.required_person_ids || []).join(','),
    grid_cell_meters: row.grid_cell_meters ?? 0,
    is_active: row.is_active,
    vertices: [...row.vertices],
  })
  dialogVisible.value = true
}

function showAddDialog() {
  editingFence.value = null
  Object.assign(form, {
    name: '', type: 1, max_people: 0, min_people: 0,
    max_stay_seconds: 0, stationary_seconds: 0,
    stationary_threshold_meters: 2.0, stationary_recovery_seconds: 3,
    required_person_ids: '', grid_cell_meters: 0, is_active: true,
    vertices: [
      { x: 100, y: 100 },
      { x: 200, y: 100 },
      { x: 200, y: 200 },
      { x: 100, y: 200 },
    ],
  })
  dialogVisible.value = true
}

async function saveFence() {
  const body = {
    ...form,
    zone_id: editingFence.value?.zone_id || `local-${Date.now()}`,
    required_person_ids: form.required_person_ids
      ? form.required_person_ids.split(',').map((s: string) => s.trim()).filter(Boolean)
      : [],
  }
  const url = editingFence.value ? `/api/fences/${editingFence.value.zone_id}` : '/api/fences'
  const method = editingFence.value ? 'PUT' : 'POST'
  const resp = await fetch(url, {
    method,
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  })
  if (resp.ok) {
    dialogVisible.value = false
    await loadFences()
  }
}

async function deleteFence(zoneId: string) {
  const resp = await fetch(`/api/fences/${zoneId}`, { method: 'DELETE', headers: authHeader() })
  if (resp.ok) await loadFences()
}

function handleVertexMove(fenceId: string, idx: number, x: number, y: number) {
  const fence = fences.value.find((f) => f.zone_id === fenceId)
  if (fence && fence.vertices[idx]) {
    fence.vertices[idx] = { x, y }
  }
}

onMounted(loadFences)
</script>
