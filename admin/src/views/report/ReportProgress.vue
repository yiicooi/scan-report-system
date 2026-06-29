<template>
  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between;align-items:center">
        <span>报工进度</span>
      </div>
    </template>

    <el-form :model="query" inline style="margin-bottom:16px">
      <el-form-item label="内部单号">
        <el-input v-model="query.internal_no" placeholder="模糊查询" clearable style="width:180px" @keyup.enter="search" />
      </el-form-item>
      <el-form-item label="外部单号">
        <el-input v-model="query.external_no" placeholder="模糊查询" clearable style="width:180px" @keyup.enter="search" />
      </el-form-item>
      <el-form-item label="零件名称">
        <el-input v-model="query.part_name" placeholder="模糊查询" clearable style="width:180px" @keyup.enter="search" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="search">查询</el-button>
      </el-form-item>
    </el-form>

    <el-table
      v-if="orders.length"
      :data="orders"
      v-loading="loading"
      stripe
      highlight-current-row
      style="margin-bottom:16px"
      @row-click="selectOrder"
    >
      <el-table-column prop="internal_no" label="内部单号" width="180" />
      <el-table-column prop="external_no" label="外部单号" width="160" />
      <el-table-column prop="part_name" label="零件名称" min-width="160" />
      <el-table-column prop="drawing_no" label="图纸编号" width="150" />
      <el-table-column prop="total_qty" label="订单数量" width="100" align="center" />
      <el-table-column prop="total_completed" label="最终完成" width="100" align="center" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="orderStatusType(row.status)" size="small">{{ orderStatusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="90">
        <template #default="{ row }">
          <el-button size="small" text type="primary" @click.stop="selectOrder(row)">查看</el-button>
        </template>
      </el-table-column>
    </el-table>

    <template v-if="order">
      <!-- 工单基本信息 -->
      <el-descriptions :column="4" border style="margin-bottom:16px">
        <el-descriptions-item label="内部单号">{{ order.internal_no }}</el-descriptions-item>
        <el-descriptions-item label="外部单号">{{ order.external_no }}</el-descriptions-item>
        <el-descriptions-item label="零件名称">{{ order.part_name || '—' }}</el-descriptions-item>
        <el-descriptions-item label="订单数量">{{ order.total_qty }}</el-descriptions-item>
        <el-descriptions-item label="最终完成">{{ order.total_completed }}</el-descriptions-item>
      </el-descriptions>

      <!-- 各工序进度 -->
      <el-table :data="progress" stripe>
        <el-table-column prop="sort" label="序号" width="60" align="center" />
        <el-table-column prop="display_name" label="工序名称" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="processStatusType(row.status)" size="small">{{ processStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="180">
          <template #default="{ row }">
            <el-progress :percentage="Math.round(row.progress_pct || 0)" :status="row.is_completed ? 'success' : ''" />
          </template>
        </el-table-column>
        <el-table-column label="总接收" width="90" align="center" prop="total_received" />
        <el-table-column label="总完成" width="90" align="center" prop="total_completed" />
        <el-table-column label="总报废" width="90" align="center" prop="total_scrap" />
        <el-table-column label="工时(h)" width="130">
          <template #default="{ row }">
            {{ row.unit_hours }} / {{ row.total_hours }}
          </template>
        </el-table-column>
        <el-table-column label="截止日期" width="120">
          <template #default="{ row }">{{ row.deadline?.slice(0,10) || '—' }}</template>
        </el-table-column>
        <el-table-column label="明细" width="80">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="showDetail(row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 报工明细弹窗 -->
    <el-dialog v-model="detailVisible" :title="`${currentProcess?.display_name} — 报工明细`" width="800px">
      <el-table :data="details" v-loading="detailLoading">
        <el-table-column prop="reported_at" label="报工时间" width="160">
          <template #default="{ row }">{{ row.reported_at?.replace('T', ' ').slice(0,19) }}</template>
        </el-table-column>
        <el-table-column label="操作人">
          <template #default="{ row }">{{ row.user?.name }}</template>
        </el-table-column>
        <el-table-column prop="received_qty" label="本次接收" width="90" align="center" />
        <el-table-column prop="completed_qty" label="本次完成" width="90" align="center" />
        <el-table-column prop="scrap_qty" label="本次报废" width="90" align="center" />
        <el-table-column label="图片" width="150">
          <template #default="{ row }">
            <div class="report-images">
              <el-link
                v-for="(url, index) in reportImages(row)"
                :key="`${url}-${index}`"
                :href="url"
                target="_blank"
                type="primary"
              >
                图片{{ index + 1 }}
              </el-link>
              <span v-if="reportImages(row).length === 0">—</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="备注" />
      </el-table>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import http from '@/api/http'

const query = reactive({ internal_no: '', external_no: '', part_name: '' })
const loading = ref(false)
const orders = ref<any[]>([])
const order = ref<any>(null)
const progress = ref<any[]>([])
const detailVisible = ref(false)
const detailLoading = ref(false)
const details = ref<any[]>([])
const currentProcess = ref<any>(null)

async function search() {
  if (!query.internal_no && !query.external_no && !query.part_name) return
  loading.value = true
  try {
    const res: any = await http.get('/admin/orders', {
      params: {
        internal_no: query.internal_no,
        external_no: query.external_no,
        part_name: query.part_name,
        page: 1,
        page_size: 50
      }
    })
    orders.value = res.data || []
    if (orders.value.length === 0) {
      ElMessage.warning('未找到工单')
      order.value = null
      progress.value = []
      return
    }
    await selectOrder(orders.value[0])
  } catch (e: any) {
    ElMessage.error(e || '未找到工单')
    orders.value = []
    order.value = null
    progress.value = []
  } finally {
    loading.value = false
  }
}

async function selectOrder(row: any) {
  try {
    const [detail, prog]: any[] = await Promise.all([
      http.get(`/admin/orders/${row.id}`),
      http.get(`/admin/orders/${row.id}/report-progress`)
    ])
    order.value = detail.data || row
    progress.value = prog.data || []
  } catch (e: any) {
    ElMessage.error(e || '加载进度失败')
  }
}

async function showDetail(row: any) {
  currentProcess.value = row
  detailVisible.value = true
  detailLoading.value = true
  try {
    const res: any = await http.get(`/admin/order-processes/${row.id}/report-details`)
    details.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e || '加载明细失败')
    details.value = []
  } finally {
    detailLoading.value = false
  }
}

function processStatusType(s: string) {
  return { pending: 'info', in_progress: 'warning', completed: 'success' }[s] || ''
}
function processStatusLabel(s: string) {
  return { pending: '待开始', in_progress: '进行中', completed: '已完成' }[s] || s
}
function orderStatusType(s: string) {
  return { draft: 'info', ready: 'primary', active: 'warning', completed: 'success' }[s] || ''
}
function orderStatusLabel(s: string) {
  return { draft: '草稿', ready: '已排工', active: '进行中', completed: '已完成' }[s] || s
}

function reportImages(row: any) {
  return [
    ...(row.receive_images || []),
    ...(row.complete_images || []),
    ...(row.scrap_images || [])
  ].filter(Boolean)
}
</script>

<style scoped>
.report-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
