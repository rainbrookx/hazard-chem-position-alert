<template>
  <el-container class="app-container" v-if="showLayout">
    <el-header height="60px" class="app-header">
      <el-menu :default-active="activeRoute" mode="horizontal" router class="app-menu">
        <el-menu-item index="/terminals">定位终端</el-menu-item>
        <el-menu-item index="/fences">电子围栏</el-menu-item>
        <el-menu-item index="/alerts">预警报警</el-menu-item>
      </el-menu>
      <el-button class="logout-btn" @click="handleLogout" type="danger" plain size="small">
        退出登录
      </el-button>
    </el-header>
    <el-main>
      <router-view />
    </el-main>
  </el-container>
  <router-view v-else />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()

const showLayout = computed(() => route.path !== '/login')
const activeRoute = computed(() => route.path)

function handleLogout() {
  auth.logout()
}
</script>

<style scoped>
.app-container {
  min-height: 100vh;
}
.app-header {
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--el-border-color-light);
  padding: 0 20px;
}
.app-menu {
  flex: 1;
  border-bottom: none !important;
}
.logout-btn {
  margin-left: auto;
}
</style>
