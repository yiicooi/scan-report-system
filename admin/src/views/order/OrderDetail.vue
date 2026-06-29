<template>
  <div v-loading="loading">
    <!-- 工单基本信息 -->
    <el-card style="margin-bottom:16px">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>工单详情 — {{ order.internal_no }}</span>
          <div>
            <el-tag :type="order.order_type === 'scrap' ? 'danger' : 'primary'" style="margin-right:8px">
              {{ order.order_type === 'scrap' ? '报废单' : '主单' }}
            </el-tag>
            <el-tag :type="statusType(order.status)">{{ statusLabel(order.status) }}</el-tag>
          </div>
        </div>
      </template>
      <el-descriptions :column="3" border>
        <el-descriptions-item label="内部单号">{{ order.internal_no }}</el-descriptions-item>
        <el-descriptions-item label="外部单号">{{ order.external_no }}</el-descriptions-item>
        <el-descriptions-item label="零件名称">{{ order.part_name || '—' }}</el-descriptions-item>
        <el-descriptions-item label="图纸编号">{{ order.drawing_no }}</el-descriptions-item>
        <el-descriptions-item label="订单数量">{{ order.total_qty }}</el-descriptions-item>
        <el-descriptions-item label="已完成">{{ order.total_completed }}</el-descriptions-item>
        <el-descriptions-item label="单价">{{ order.unit_price }}</el-descriptions-item>
        <el-descriptions-item label="总额">{{ order.total_amount }}</el-descriptions-item>
        <el-descriptions-item label="订单日期">{{ order.order_date?.slice(0,10) }}</el-descriptions-item>
        <el-descriptions-item label="图纸">
          <el-link v-if="order.drawing_url" :href="order.drawing_url" target="_blank" type="primary">查看图纸</el-link>
          <span v-else>—</span>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 工序明细编辑器 -->
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>工序流程</span>
          <div>
            <el-button v-if="auth.hasPermission('order:update')" @click="showAliasDialog = true">从流程模板导入</el-button>
            <el-button v-if="auth.hasPermission('order:update')" type="primary" @click="addProcess">添加工序</el-button>
            <el-button v-if="auth.hasPermission('order:update')" type="success" @click="saveProcesses">保存工序</el-button>
          </div>
        </div>
      </template>

      <el-table :data="processes" row-key="id">
        <el-table-column label="排序" width="70" align="center">
          <template #default="{ $index }">{{ $index + 1 }}</template>
        </el-table-column>
        <el-table-column label="工序名称" min-width="160">
          <template #default="{ row }">
            <el-select v-model="row.process_id" placeholder="选择工序" filterable @change="(v: number) => onProcessChange(row, v)">
              <el-option v-for="p in processList" :key="p.id" :label="p.name" :value="p.id" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="显示名称" min-width="140">
          <template #default="{ row }">
            <el-input v-model="row.display_name" placeholder="别名（可选）" />
          </template>
        </el-table-column>
        <el-table-column label="单个工时(h)" width="130">
          <template #default="{ row }">
            <el-input-number v-model="row.unit_hours" :min="0" :precision="2" :step="0.5" style="width:120px" />
          </template>
        </el-table-column>
        <el-table-column label="总工时(h)" width="130">
          <template #default="{ row }">
            <el-input-number v-model="row.total_hours" :min="0" :precision="2" style="width:120px" />
          </template>
        </el-table-column>
        <el-table-column label="最后完成时间" width="200">
          <template #default="{ row }">
            <el-date-picker v-model="row.deadline" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width:180px" />
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="processStatusType(row.status)" size="small">{{ processStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="130">
          <template #default="{ row }">
            <el-progress v-if="row.summary" :percentage="Math.round(row.summary.progress_pct || 0)" :stroke-width="8" />
          </template>
        </el-table-column>
        <el-table-column label="接收/完成/报废" width="160" align="center">
          <template #default="{ row }">
            <span v-if="row.summary">
              {{ row.summary.total_received }}/{{ row.summary.total_completed }}/{{ row.summary.total_scrap }}
            </span>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column v-if="auth.hasPermission('order:update')" label="操作" width="80" fixed="right">
          <template #default="{ $index }">
            <el-button size="small" type="danger" text @click="processes.splice($index, 1)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 流程模板导入弹窗 -->
    <el-dialog v-model="showAliasDialog" title="选择流程模板" width="500px">
      <el-table :data="aliases" @row-click="importAlias" highlight-current-row style="cursor:pointer">
        <el-table-column prop="alias_name" label="模板名称" />
        <el-table-column prop="note" label="备注" />
        <el-table-column label="工序数" width="80">
          <template #default="{ row }">{{ row.items?.length }}</template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="showAliasDialog = false">取消</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const orderId = route.params.id
const loading = ref(false)
const order = ref<any>({})
const processes = ref<any[]>([])
const processList = ref<any[]>([])
const aliases = ref<any[]>([])
const showAliasDialog = ref(false)

async function loadAll() {
  loading.value = true
  try {
    const [o, pl, al]: any[] = await Promise.all([
      http.get(`/admin/orders/${orderId}`),
      http.get('/admin/processes'),
      http.get('/admin/process-aliases')
    ])
    order.value = o.data
    processes.value = o.data.processes || []
    processList.value = pl.data || []
    aliases.value = al.data || []
  } finally {
    loading.value = false
  }
}

function addProcess() {
  processes.value.push({
    order_id: Number(orderId),
    process_id: null,
    display_name: '',
    sort: processes.value.length + 1,
    unit_hours: 0,
    total_hours: 0,
    deadline: null,
    status: 'pending',
    _isNew: true
  })
}

function onProcessChange(row: any, processId: number) {
  const p = processList.value.find(x => x.id === processId)
  if (p && !row.display_name) {
    row.display_name = p.name
  }
}

function importAlias(alias: any) {
  const items = alias.items || []
  const newProcesses = items.map((item: any, idx: number) => ({
    order_id: Number(orderId),
    process_id: item.process_id,
    display_name: item.display_name || item.process?.name || '',
    sort: processes.value.length + idx + 1,
    unit_hours: 0,  // 工时必须手动填写
    total_hours: 0,
    deadline: null,
    status: 'pending',
    _isNew: true
  }))
  processes.value.push(...newProcesses)
  showAliasDialog.value = false
  ElMessage.success(`已导入 ${items.length} 道工序，请填写各工序工时`)
}

async function saveProcesses() {
  // 只保存新增的工序，已有工序使用 PUT
  const newItems = processes.value.filter(p => p._isNew)
  const updates = processes.value.filter(p => !p._isNew && p.id)

  try {
    // 保存新工序
    for (let i = 0; i < newItems.length; i++) {
      const item = { ...newItems[i], sort: processes.value.indexOf(newItems[i]) + 1 }
      delete item._isNew
      await http.post(`/admin/orders/${orderId}/processes`, item)
    }
    // 更新已有工序（工时/截止日期）
    for (const item of updates) {
      await http.put(`/admin/orders/${orderId}/processes/${item.id}`, item)
    }
    ElMessage.success('工序保存成功')
    loadAll()
  } catch (e: any) {
    ElMessage.error(e)
  }
}

function statusType(s: string) {
  return { draft: 'info', ready: 'primary', active: 'warning', completed: 'success' }[s] || ''
}
function statusLabel(s: string) {
  return { draft: '草稿', ready: '已排工', active: '进行中', completed: '已完成' }[s] || s
}
function processStatusType(s: string) {
  return { pending: 'info', in_progress: 'warning', completed: 'success' }[s] || ''
}
function processStatusLabel(s: string) {
  return { pending: '待开始', in_progress: '进行中', completed: '已完成' }[s] || s
}

onMounted(loadAll)
</script>
