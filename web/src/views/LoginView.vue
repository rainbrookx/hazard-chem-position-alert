<template>
  <div class="login-page">
    <div class="login-panel">
      <div class="form-header">
        <h2 class="form-title">人员定位告警系统</h2>
        <p class="form-desc">Hazardous‑Chemical Work‑Safety Personnel Positioning Alarm System</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="请输入用户名"
            size="large"
            :prefix-icon="User"
            class="custom-input"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            size="large"
            :prefix-icon="Lock"
            show-password
            class="custom-input"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="login-btn"
            @click="handleLogin"
          >
            {{ loading ? '登录中...' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>

      <transition name="fade">
        <div v-if="error" class="login-error">
          <el-icon><WarningFilled /></el-icon>
          <span>{{ error }}</span>
        </div>
      </transition>
    </div>

    <footer class="login-footer">
      Hazardous‑Chemical Work‑Safety Personnel Positioning Alarm System &copy; 2026
    </footer>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { User, Lock, WarningFilled } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()

const form = reactive({ username: '', password: '' })
const formRef = ref<FormInstance>()
const loading = ref(false)
const error = ref('')

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  error.value = ''
  try {
    await auth.login(form.username, form.password)
    router.push('/terminals')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #e8e8e8;
}

.login-panel {
  width: 400px;
  padding: 48px 40px 36px;
  background: rgba(255, 255, 255, 0.65);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.form-header {
  text-align: center;
  margin-bottom: 32px;
}

.form-title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  letter-spacing: 2px;
}

.form-desc {
  margin: 8px 0 0;
  font-size: 12px;
  color: #909399;
  letter-spacing: 1px;
}

.login-form {
  margin-top: 8px;
}

.custom-input :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.8);
  box-shadow: 0 0 0 1px #dcdfe6 inset;
}

.custom-input :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #c0c4cc inset;
}

.custom-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #409eff inset;
}

.login-btn {
  width: 100%;
  margin-top: 4px;
}

.login-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 16px;
  padding: 10px 12px;
  font-size: 13px;
  color: #f56c6c;
  background: rgba(245, 108, 108, 0.08);
  border-radius: 6px;
}

.login-footer {
  position: fixed;
  bottom: 16px;
  font-size: 12px;
  color: #b0b0b0;
  letter-spacing: 1px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
