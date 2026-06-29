<template>
  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between">
        <span>角色权限管理</span>
        <el-button v-if="auth.hasPermission('role:create')" type="primary" @click="openRoleForm">新增角色</el-button>
      </div>
    </template>

    <el-row :gutter="16">
      <!-- 角色列表 -->
      <el-col :span="8">
        <el-card shadow="never">
          <template #header>角色列表</template>
          <el-menu :default-active="String(currentRole?.id)" @select="onRoleSelect">
            <el-menu-item v-for="r in roles" :key="r.id" :index="String(r.id)">
              {{ r.name }}
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>

      <!-- 权限分配 -->
      <el-col :span="16">
        <el-card shadow="never" v-if="currentRole">
          <template #header>
            <div style="display:flex;justify-content:space-between;align-items:center">
              <span>{{ currentRole.name }} — 权限配置</span>
              <el-button v-if="auth.hasPermission('role:update')" type="primary" size="small" @click="savePermissions">保存</el-button>
            </div>
          </template>

          <el-table :data="groupedPerms" row-key="resource" default-expand-all>
            <el-table-column prop="resource" label="资源" width="150">
              <template #default="{ row }">
                <strong>{{ resourceLabel(row.resource) }}</strong>
              </template>
            </el-table-column>
            <el-table-column label="权限">
              <template #default="{ row }">
                <el-checkbox-group v-model="row.checked">
                  <el-checkbox v-for="p in row.actions" :key="p.id" :label="p.id">
                    {{ p.action }}
                  </el-checkbox>
                </el-checkbox-group>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-empty v-else description="请选择左侧角色" />
      </el-col>
    </el-row>

    <!-- 新增角色弹窗 -->
    <el-dialog v-model="roleFormVisible" title="新增角色" width="400px">
      <el-form :model="roleForm" :rules="rules" ref="roleFormRef" label-width="80px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="roleForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="roleForm.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleFormVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRole">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const roles = ref<any[]>([])
const auth = useAuthStore()
const permissions = ref<any[]>([])
const currentRole = ref<any>(null)
const roleFormVisible = ref(false)
const roleFormRef = ref()
const roleForm = reactive({ name: '', description: '' })
const rules = { name: [{ required: true }] }

// 按 resource 分组权限
const groupedPerms = computed(() => {
  const map: Record<string, any> = {}
  for (const p of permissions.value) {
    if (!map[p.resource]) map[p.resource] = { resource: p.resource, actions: [], checked: [] }
    map[p.resource].actions.push(p)
  }
  // 设置当前角色已有权限
  if (currentRole.value) {
    const rolePermIds = new Set((currentRole.value.permissions || []).map((p: any) => p.id))
    for (const g of Object.values(map) as any[]) {
      g.checked = g.actions.filter((a: any) => rolePermIds.has(a.id)).map((a: any) => a.id)
    }
  }
  return Object.values(map)
})

async function load() {
  const [r, p]: any[] = await Promise.all([http.get('/admin/roles'), http.get('/admin/permissions')])
  roles.value = r.data || []
  permissions.value = p.data || []
}
function onRoleSelect(id: string) {
  currentRole.value = roles.value.find(r => String(r.id) === id)
}
async function savePermissions() {
  const permIds = groupedPerms.value.flatMap((g: any) => g.checked)
  await http.put(`/admin/roles/${currentRole.value.id}/permissions`, { permission_ids: permIds })
  ElMessage.success('权限已更新')
  load()
}
function openRoleForm() {
  roleForm.name = ''; roleForm.description = ''
  roleFormVisible.value = true
}
async function saveRole() {
  await roleFormRef.value.validate()
  await http.post('/admin/roles', roleForm)
  ElMessage.success('角色创建成功')
  roleFormVisible.value = false
  load()
}
function resourceLabel(r: string): string {
  const map: Record<string, string> = { order: '工单', user: '用户', role: '角色', department: '部门', process: '工序', report: '报工' }
  return map[r] || r
}
onMounted(load)
</script>
