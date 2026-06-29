<template>
  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between;align-items:center">
        <span>用户管理</span>
        <el-button v-if="auth.hasPermission('user:create')" type="primary" @click="openForm()">新增用户</el-button>
      </div>
    </template>
    <el-table :data="users" v-loading="loading" stripe>
      <el-table-column prop="name" label="用户名" />
      <el-table-column label="部门">
        <template #default="{ row }">{{ row.department?.name || '—' }}</template>
      </el-table-column>
      <el-table-column label="角色">
        <template #default="{ row }">{{ row.role?.name || '—' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.is_active ? 'success' : 'danger'" size="small">
            {{ row.is_active ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button v-if="auth.hasPermission('user:update')" size="small" @click="openForm(row)">编辑</el-button>
          <el-popconfirm v-if="auth.hasPermission('user:delete')" title="确定删除？" @confirm="deleteUser(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑用户' : '新增用户'" width="480px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="用户名" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="form.id ? '新密码' : '密码'" :prop="form.id ? '' : 'password'">
          <el-input v-model="form.password" type="password" show-password :placeholder="form.id ? '不填则不修改' : ''" />
        </el-form-item>
        <el-form-item label="部门">
          <el-tree-select
            v-model="form.department_id"
            :data="depts"
            :props="{ label: 'name', children: 'children', value: 'id' }"
            node-key="id"
            clearable
            check-strictly
            default-expand-all
            placeholder="请选择"
          />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role_id" clearable placeholder="请选择">
            <el-option v-for="r in roles" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" v-if="form.id">
          <el-switch v-model="form.is_active" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveUser">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const loading = ref(false)
const auth = useAuthStore()
const saving = ref(false)
const users = ref<any[]>([])
const depts = ref<any[]>([])
const roles = ref<any[]>([])
const dialogVisible = ref(false)
const formRef = ref()
const form = reactive<any>({ id: null, name: '', password: '', department_id: null, role_id: null, is_active: true })
const rules = {
  name: [{ required: true, message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码', min: 6 }]
}

async function load() {
  loading.value = true
  const [u, d, r]: any[] = await Promise.all([
    http.get('/admin/users'),
    http.get('/admin/departments'),
    http.get('/admin/roles')
  ])
  users.value = u.data || []
  depts.value = d.data || []
  roles.value = r.data || []
  loading.value = false
}

function openForm(row?: any) {
  if (row) {
    Object.assign(form, { id: row.id, name: row.name, password: '', department_id: row.department_id, role_id: row.role_id, is_active: row.is_active })
  } else {
    Object.assign(form, { id: null, name: '', password: '', department_id: null, role_id: null, is_active: true })
  }
  dialogVisible.value = true
}

async function saveUser() {
  await formRef.value.validate()
  saving.value = true
  try {
    if (form.id) {
      await http.put(`/admin/users/${form.id}`, form)
    } else {
      await http.post('/admin/users', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    load()
  } catch (e: any) {
    ElMessage.error(e)
  } finally {
    saving.value = false
  }
}

async function deleteUser(id: number) {
  await http.delete(`/admin/users/${id}`)
  ElMessage.success('已删除')
  load()
}

onMounted(load)
</script>
