<template>
  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between">
        <span>流程模板（别名）</span>
        <el-button v-if="auth.hasPermission('process:create')" type="primary" @click="openCreate">新增模板</el-button>
      </div>
    </template>
    <el-table :data="aliases" v-loading="loading" stripe>
      <el-table-column prop="alias_name" label="模板名称" />
      <el-table-column label="所属部门">
        <template #default="{ row }">{{ row.department?.name || '—' }}</template>
      </el-table-column>
      <el-table-column label="工序数">
        <template #default="{ row }">{{ row.items?.length || 0 }}</template>
      </el-table-column>
      <el-table-column prop="note" label="备注" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-popconfirm v-if="auth.hasPermission('process:delete')" title="确定删除此模板？" @confirm="del(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="visible" title="新增流程模板" width="700px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="模板名称" prop="alias_name">
          <el-input v-model="form.alias_name" />
        </el-form-item>
        <el-form-item label="所属部门">
          <el-tree-select
            v-model="form.department_id"
            :data="depts"
            :props="{ label: 'name', children: 'children', value: 'id' }"
            node-key="id"
            clearable
            check-strictly
            :render-after-expand="false"
            default-expand-all
            placeholder="请选择部门"
            style="width:100%"
          />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.note" type="textarea" :rows="2" />
        </el-form-item>
        <el-divider>工序列表（工时在使用时填写）</el-divider>
        <div v-for="(item, idx) in form.items" :key="idx" style="display:flex;gap:8px;margin-bottom:8px;align-items:center">
          <span style="width:30px;text-align:right">{{ idx+1 }}.</span>
          <el-select v-model="item.process_id" placeholder="选择工序" filterable style="width:180px" @change="(v: number) => autoFillName(item, v)">
            <el-option v-for="p in processList" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
          <el-input v-model="item.display_name" placeholder="显示别名（可选）" style="flex:1" />
          <el-button size="small" type="danger" circle @click="form.items.splice(idx,1)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button @click="form.items.push({ process_id: null, display_name: '', sort: form.items.length+1 })">
          + 添加工序
        </el-button>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
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
const aliases = ref<any[]>([])
const depts = ref<any[]>([])
const processList = ref<any[]>([])
const visible = ref(false)
const formRef = ref()
const form = reactive<any>({ alias_name: '', department_id: null, note: '', items: [] })
const rules = { alias_name: [{ required: true, message: '请输入模板名称' }] }

async function load() {
  loading.value = true
  const [a, d, p]: any[] = await Promise.all([
    http.get('/admin/process-aliases'),
    http.get('/admin/departments'),
    http.get('/admin/processes')
  ])
  aliases.value = a.data || []
  depts.value = d.data || []
  processList.value = p.data || []
  loading.value = false
}
function openCreate() {
  form.alias_name = ''; form.department_id = null; form.note = ''; form.items = []
  visible.value = true
}
function autoFillName(item: any, processId: number) {
  const p = processList.value.find(x => x.id === processId)
  if (p && !item.display_name) item.display_name = p.name
}
async function save() {
  await formRef.value.validate()
  saving.value = true
  const payload = { ...form, items: form.items.map((item: any, idx: number) => ({ ...item, sort: idx+1 })) }
  try {
    await http.post('/admin/process-aliases', payload)
    ElMessage.success('创建成功')
    visible.value = false
    load()
  } catch (e: any) {
    ElMessage.error(e)
  } finally {
    saving.value = false
  }
}
async function del(id: number) {
  await http.delete(`/admin/process-aliases/${id}`)
  ElMessage.success('已删除')
  load()
}
onMounted(load)
</script>
