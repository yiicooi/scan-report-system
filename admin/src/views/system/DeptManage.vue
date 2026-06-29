<template>
  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between">
        <span>部门管理</span>
        <el-button v-if="auth.hasPermission('department:create')" type="primary" @click="openForm()">新增部门</el-button>
      </div>
    </template>
    <el-table :data="depts" v-loading="loading" row-key="id" default-expand-all>
      <el-table-column prop="name" label="部门名称" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button v-if="auth.hasPermission('department:create')" size="small" @click="openForm(row)">添加子部门</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="visible" title="新增部门" width="400px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="部门名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="上级部门">
          <el-input :value="parentName" disabled />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const loading = ref(false)
const auth = useAuthStore()
const depts = ref<any[]>([])
const visible = ref(false)
const formRef = ref()
const form = reactive({ name: '', parent_id: null as null|number })
const rules = { name: [{ required: true, message: '请输入部门名称' }] }

const parentName = computed(() => {
  if (!form.parent_id) return '顶级'
  const find = (list: any[]): any => {
    for (const d of list) {
      if (d.id === form.parent_id) return d
      if (d.children) { const r = find(d.children); if (r) return r }
    }
  }
  return find(depts.value)?.name || ''
})

async function load() {
  loading.value = true
  const res: any = await http.get('/admin/departments')
  depts.value = res.data || []
  loading.value = false
}
function openForm(parent?: any) {
  form.name = ''; form.parent_id = parent?.id || null
  visible.value = true
}
async function save() {
  await formRef.value.validate()
  await http.post('/admin/departments', form)
  ElMessage.success('创建成功')
  visible.value = false
  load()
}
onMounted(load)
</script>
