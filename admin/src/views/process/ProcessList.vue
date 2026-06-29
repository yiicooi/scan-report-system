<template>
  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between">
        <span>工序模板</span>
        <el-button v-if="auth.hasPermission('process:create')" type="primary" @click="openForm()">新增</el-button>
      </div>
    </template>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="name" label="工序名称" />
      <el-table-column label="所属部门">
        <template #default="{ row }">{{ row.department?.name || '—' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button v-if="auth.hasPermission('process:update')" size="small" @click="openForm(row)">编辑</el-button>
          <el-popconfirm v-if="auth.hasPermission('process:delete')" title="确定删除？" @confirm="del(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="visible" :title="editId ? '编辑工序' : '新增工序'" width="400px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="工序名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="所属部门">
          <el-tree-select
            v-model="form.department_id"
            :data="depts"
            :props="{ label: 'name', children: 'children', value: 'id' }"
            node-key="id"
            clearable
            check-strictly
            default-expand-all
            placeholder="请选择部门"
            style="width:100%"
          />
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const loading = ref(false)
const auth = useAuthStore()
const list = ref<any[]>([])
const depts = ref<any[]>([])
const visible = ref(false)
const editId = ref<number|null>(null)
const formRef = ref()
const form = reactive({ name: '', department_id: null as null|number })
const rules = { name: [{ required: true, message: '请输入工序名称' }] }

async function load() {
  loading.value = true
  const [p, d]: any[] = await Promise.all([http.get('/admin/processes'), http.get('/admin/departments')])
  list.value = p.data || []
  depts.value = d.data || []
  loading.value = false
}
function openForm(row?: any) {
  if (row) {
    editId.value = row.id
    form.name = row.name
    form.department_id = row.department_id
  } else {
    editId.value = null
    form.name = ''
    form.department_id = null
  }
  visible.value = true
}
async function save() {
  await formRef.value.validate()
  if (editId.value) {
    await http.put(`/admin/processes/${editId.value}`, form)
    ElMessage.success('更新成功')
  } else {
    await http.post('/admin/processes', form)
    ElMessage.success('创建成功')
  }
  visible.value = false
  load()
}
async function del(id: number) {
  await http.delete(`/admin/processes/${id}`)
  ElMessage.success('已删除')
  load()
}
onMounted(load)
</script>
