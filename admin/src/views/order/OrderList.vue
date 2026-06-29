<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>工单列表</span>
          <div style="display:flex;gap:8px">
            <el-upload
              v-if="auth.hasPermission('order:create')"
              :http-request="importExcel"
              :show-file-list="false"
              accept=".xlsx"
            >
              <el-button type="warning" :loading="importLoading">Excel导入</el-button>
            </el-upload>
            <el-button type="success" :disabled="selectedOrders.length === 0" @click="openOrderPrint(selectedOrders)">
              批量A4打印{{ selectedOrders.length > 0 ? `（${selectedOrders.length}）` : '' }}
            </el-button>
            <el-button v-if="auth.hasPermission('order:create')" type="primary" @click="openCreate">新增工单</el-button>
          </div>
        </div>
      </template>

      <!-- 搜索 -->
      <el-form :model="query" inline style="margin-bottom:16px">
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="单号/图纸编号" clearable style="width:200px" />
        </el-form-item>
        <el-form-item label="外部单号">
          <el-input v-model="query.external_no" placeholder="模糊查询" clearable style="width:180px" />
        </el-form-item>
        <el-form-item label="零件名称">
          <el-input v-model="query.part_name" placeholder="模糊查询" clearable style="width:180px" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="query.order_type" clearable placeholder="全部" style="width:120px">
            <el-option label="主单" value="normal" />
            <el-option label="报废单" value="scrap" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部" style="width:120px">
            <el-option label="草稿" value="draft" />
            <el-option label="已排工" value="ready" />
            <el-option label="进行中" value="active" />
            <el-option label="已完成" value="completed" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadOrders">查询</el-button>
        </el-form-item>
      </el-form>

      <!-- 表格 -->
      <el-table :data="orders" v-loading="loading" stripe @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="50" />
        <el-table-column prop="internal_no" label="内部单号" width="180">
          <template #default="{ row }">
            <el-link type="primary" @click="router.push(`/orders/${row.id}`)">{{ row.internal_no }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="external_no" label="外部单号" width="150" />
        <el-table-column prop="part_name" label="零件名称" width="160" />
        <el-table-column prop="drawing_no" label="图纸编号" width="150" />
        <el-table-column prop="total_qty" label="订单数量" width="100" align="center" />
        <el-table-column prop="total_completed" label="已完成" width="100" align="center" />
        <el-table-column label="进度" width="160">
          <template #default="{ row }">
            <el-progress
              :percentage="row.total_qty > 0 ? Math.round(row.total_completed / row.total_qty * 100) : 0"
              :status="row.status === 'completed' ? 'success' : ''"
            />
          </template>
        </el-table-column>
        <el-table-column prop="order_type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="row.order_type === 'scrap' ? 'danger' : 'primary'" size="small">
              {{ row.order_type === 'scrap' ? '报废单' : '主单' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="order_date" label="订单日期" width="120">
          <template #default="{ row }">{{ row.order_date?.slice(0, 10) }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="320">
          <template #default="{ row }">
            <el-button size="small" @click="router.push(`/orders/${row.id}`)">详情</el-button>
            <el-button size="small" type="success" @click="openOrderPrint([row])">A4打印</el-button>
            <el-button v-if="row.order_type === 'normal' && auth.hasPermission('order:create')" size="small" type="warning"
              @click="createScrap(row.id)">创建报废单</el-button>
            <el-button
              v-if="row.status === 'draft' && auth.hasPermission('order:delete')"
              size="small"
              type="danger"
              @click="deleteDraftOrder(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        :page-size="20"
        :total="total"
        layout="total, prev, pager, next"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="loadOrders"
      />
    </el-card>

    <!-- A4 工单打印弹窗 -->
    <el-dialog
      v-model="qrDialogVisible"
      title="A4工单打印"
      width="900px"
      @opened="renderQrCodes"
    >
      <div v-loading="printLoading" class="a4-preview-area" id="order-print-area">
        <section v-for="order in printOrders" :key="order.id" class="a4-order-page">
          <div class="print-header">
            <div>
              <div class="print-title">扫码报工工单</div>
              <div class="print-subtitle">{{ order.internal_no }}</div>
            </div>
            <div class="print-qr-block">
              <canvas :ref="el => setQrCanvasRef(el, order.internal_no)" class="qr-canvas" />
              <div class="print-qr-text">扫码报工</div>
            </div>
          </div>

          <div class="print-info-grid">
            <div><span>内部单号</span><strong>{{ order.internal_no || '—' }}</strong></div>
            <div><span>外部单号</span><strong>{{ order.external_no || '—' }}</strong></div>
            <div><span>零件名称</span><strong>{{ order.part_name || '—' }}</strong></div>
            <div><span>图纸编号</span><strong>{{ order.drawing_no || '—' }}</strong></div>
            <div><span>订单数量</span><strong>{{ order.total_qty ?? '—' }}</strong></div>
            <div><span>已完成</span><strong>{{ order.total_completed ?? 0 }}</strong></div>
            <div><span>单价</span><strong>{{ order.unit_price ?? 0 }}</strong></div>
            <div><span>总额</span><strong>{{ order.total_amount ?? 0 }}</strong></div>
            <div><span>订单日期</span><strong>{{ formatDate(order.order_date) }}</strong></div>
            <div><span>工单类型</span><strong>{{ order.order_type === 'scrap' ? '报废单' : '主单' }}</strong></div>
            <div><span>状态</span><strong>{{ statusLabel(order.status) }}</strong></div>
            <div><span>图纸链接</span><strong>{{ order.drawing_url || '—' }}</strong></div>
          </div>

          <table class="print-process-table">
            <thead>
              <tr>
                <th>序号</th>
                <th>工序名称</th>
                <th>显示名称</th>
                <th>单个工时(h)</th>
                <th>总工时(h)</th>
                <th>最后完成时间</th>
                <th>状态</th>
                <th>接收/完成/报废</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(process, index) in sortedProcesses(order.processes)" :key="process.id || index">
                <td>{{ index + 1 }}</td>
                <td>{{ process.process?.name || process.display_name || '—' }}</td>
                <td>{{ process.display_name || '—' }}</td>
                <td>{{ process.unit_hours ?? 0 }}</td>
                <td>{{ process.total_hours ?? 0 }}</td>
                <td>{{ formatDateTime(process.deadline) }}</td>
                <td>{{ processStatusLabel(process.status) }}</td>
                <td>
                  <span v-if="process.summary">
                    {{ process.summary.total_received }}/{{ process.summary.total_completed }}/{{ process.summary.total_scrap }}
                  </span>
                  <span v-else>0/0/0</span>
                </td>
              </tr>
              <tr v-if="!sortedProcesses(order.processes).length">
                <td colspan="8" class="empty-process">—</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
      <template #footer>
        <el-button @click="qrDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="printOrderSheets">
          <el-icon style="margin-right:4px"><Printer /></el-icon>打印
        </el-button>
      </template>
    </el-dialog>

    <!-- 新增工单弹窗 -->
    <el-dialog v-model="dialogVisible" title="新增工单" width="600px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="内部单号">
          <el-input value="保存后自动生成" disabled />
        </el-form-item>
        <el-form-item label="外部单号">
          <el-input v-model="form.external_no" />
        </el-form-item>
        <el-form-item label="零件名称">
          <el-input v-model="form.part_name" />
        </el-form-item>
        <el-form-item label="图纸编号">
          <el-input v-model="form.drawing_no" />
        </el-form-item>
        <el-form-item label="订单数量" prop="total_qty">
          <el-input-number v-model="form.total_qty" :min="1" />
        </el-form-item>
        <el-form-item label="单价">
          <el-input-number v-model="form.unit_price" :min="0" :precision="4" />
        </el-form-item>
        <el-form-item label="总额">
          <el-input-number v-model="form.total_amount" :min="0" :precision="4" />
        </el-form-item>
        <el-form-item label="订单日期" prop="order_date">
          <el-date-picker v-model="form.order_date" type="date" value-format="YYYY-MM-DDTHH:mm:ssZ" />
        </el-form-item>
        <el-form-item label="图纸上传">
          <el-upload :http-request="uploadDrawing" :show-file-list="true" accept=".pdf,.jpg,.png,.dwg">
            <el-button>选择文件（可选）</el-button>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveOrder">保存</el-button>
      </template>
    </el-dialog>

    <!-- Excel 导入结果 -->
    <el-dialog v-model="importResultVisible" title="Excel导入结果" width="720px">
      <el-alert
        :title="`成功 ${importSummary.success_count} 条，失败 ${importSummary.fail_count} 条`"
        :type="importSummary.fail_count > 0 ? 'warning' : 'success'"
        show-icon
        :closable="false"
        style="margin-bottom:12px"
      />
      <el-table :data="importResults" max-height="360" stripe>
        <el-table-column prop="row" label="行号" width="80" />
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small">
              {{ row.success ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="internal_no" label="内部单号" width="180" />
        <el-table-column prop="error" label="错误信息" />
      </el-table>
      <template #footer>
        <el-button type="primary" @click="importResultVisible = false">知道了</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Printer } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import http from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const printLoading = ref(false)
const importLoading = ref(false)
const orders = ref<any[]>([])
const total = ref(0)
const dialogVisible = ref(false)
const formRef = ref()
const importResultVisible = ref(false)
const importResults = ref<any[]>([])
const importSummary = reactive({ success_count: 0, fail_count: 0 })

// 多选
const selectedOrders = ref<any[]>([])
function handleSelectionChange(val: any[]) {
  selectedOrders.value = val
}

const qrDialogVisible = ref(false)
const printOrders = ref<any[]>([])
const qrCanvasMap = new Map<string, HTMLCanvasElement>()

function setQrCanvasRef(el: any, internalNo: string) {
  if (el) qrCanvasMap.set(internalNo, el as HTMLCanvasElement)
}

async function openOrderPrint(rows: any[]) {
  if (!rows.length) return
  printLoading.value = true
  try {
    const details = await Promise.all(rows.map(async (row) => {
      const res: any = await http.get(`/admin/orders/${row.id}`)
      return res.data || row
    }))
    printOrders.value = details
    qrCanvasMap.clear()
    qrDialogVisible.value = true
  } catch (e: any) {
    ElMessage.error(e)
  } finally {
    printLoading.value = false
  }
}

async function renderQrCodes() {
  await nextTick()
  for (const order of printOrders.value) {
    const canvas = qrCanvasMap.get(order.internal_no)
    if (canvas) {
      await QRCode.toCanvas(canvas, order.internal_no, {
        width: 132,
        margin: 2,
        color: { dark: '#000000', light: '#ffffff' }
      })
    }
  }
}

function printOrderSheets() {
  const printArea = document.getElementById('order-print-area')
  if (!printArea) return
  const printWindow = window.open('', '_blank', 'width=900,height=700')
  if (!printWindow) return
  const clonedArea = printArea.cloneNode(true) as HTMLElement
  const canvases = printArea.querySelectorAll('canvas')
  const imgs = clonedArea.querySelectorAll('canvas')
  canvases.forEach((canvas, i) => {
    const img = document.createElement('img')
    img.src = canvas.toDataURL('image/png')
    img.style.width = '132px'
    img.style.height = '132px'
    imgs[i].replaceWith(img)
  })
  printWindow.document.write(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>A4工单打印</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: Arial, "Microsoft YaHei", sans-serif; background: #fff; color: #111; }
    @page { size: A4; margin: 10mm; }
    .a4-order-page {
      width: 190mm;
      min-height: 277mm;
      padding: 0;
      page-break-after: always;
      break-after: page;
    }
    .a4-order-page:last-child { page-break-after: auto; break-after: auto; }
    .print-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      border-bottom: 2px solid #111;
      padding-bottom: 8mm;
      margin-bottom: 7mm;
    }
    .print-title { font-size: 24px; font-weight: 700; letter-spacing: 0; }
    .print-subtitle { font-size: 14px; margin-top: 4mm; }
    .print-qr-block { text-align: center; }
    .print-qr-block img { width: 132px; height: 132px; display: block; }
    .print-qr-text { font-size: 11px; margin-top: 2mm; }
    .print-info-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      border-top: 1px solid #333;
      border-left: 1px solid #333;
      margin-bottom: 8mm;
    }
    .print-info-grid div {
      min-height: 12mm;
      border-right: 1px solid #333;
      border-bottom: 1px solid #333;
      padding: 2mm 2.5mm;
      overflow-wrap: anywhere;
    }
    .print-info-grid span {
      display: block;
      color: #555;
      font-size: 10px;
      margin-bottom: 1.5mm;
    }
    .print-info-grid strong {
      display: block;
      font-size: 12px;
      font-weight: 600;
    }
    .print-process-table {
      width: 100%;
      border-collapse: collapse;
      table-layout: fixed;
      font-size: 11px;
    }
    .print-process-table th,
    .print-process-table td {
      border: 1px solid #333;
      padding: 2mm;
      text-align: center;
      vertical-align: middle;
      overflow-wrap: anywhere;
    }
    .print-process-table th {
      background: #f1f3f5;
      font-weight: 700;
    }
    @media print {
      body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    }
  </style>
</head>
<body>
  ${clonedArea.innerHTML}
  <script>window.onload = function() { window.print(); window.close(); }<\/script>
</body>
</html>`)
  printWindow.document.close()
}

function sortedProcesses(processes: any[] = []) {
  return [...processes].sort((a, b) => (a.sort || 0) - (b.sort || 0))
}

const query = reactive({ keyword: '', external_no: '', part_name: '', order_type: '', status: '', page: 1 })
const form = reactive({
  external_no: '', part_name: '', drawing_no: '', drawing_url: '',
  total_qty: 1, unit_price: 0, total_amount: 0, order_date: ''
})
const rules = {
  total_qty: [{ required: true }],
  order_date: [{ required: true, message: '请选择订单日期' }]
}

async function loadOrders() {
  loading.value = true
  try {
    const res: any = await http.get('/admin/orders', { params: query })
    orders.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { external_no: '', part_name: '', drawing_no: '', drawing_url: '', total_qty: 1, unit_price: 0, total_amount: 0, order_date: '' })
  dialogVisible.value = true
}

async function uploadDrawing(opts: any) {
  // 1. 获取预签名 URL
  const presign: any = await http.post('/admin/oss/presign', { filename: opts.file.name, prefix: 'drawings' })
  // 2. 直传 OSS
  await fetch(presign.upload_url, { method: 'PUT', body: opts.file })
  form.drawing_url = presign.access_url
  ElMessage.success('图纸上传成功')
}

async function importExcel(opts: any) {
  importLoading.value = true
  try {
    const formData = new FormData()
    formData.append('file', opts.file)
    const res: any = await http.post('/admin/orders/import-excel', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    importSummary.success_count = res.success_count || 0
    importSummary.fail_count = res.fail_count || 0
    importResults.value = res.results || []
    importResultVisible.value = true
    ElMessage.success(`导入完成：成功 ${importSummary.success_count} 条`)
    opts.onSuccess?.(res)
    loadOrders()
  } catch (e: any) {
    opts.onError?.(e)
    ElMessage.error(e)
  } finally {
    importLoading.value = false
  }
}

async function saveOrder() {
  await formRef.value.validate()
  saving.value = true
  try {
    await http.post('/admin/orders', form)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    loadOrders()
  } catch (e: any) {
    ElMessage.error(e)
  } finally {
    saving.value = false
  }
}

async function createScrap(id: number) {
  await ElMessageBox.confirm('确定为该工单创建报废单？', '提示')
  try {
    const res: any = await http.post(`/admin/orders/${id}/scrap`)
    ElMessage.success(`报废单 ${res.data.internal_no} 已创建`)
    loadOrders()
  } catch (e: any) {
    ElMessage.error(e)
  }
}

async function deleteDraftOrder(row: any) {
  await ElMessageBox.confirm(`确定删除草稿工单 ${row.internal_no}？`, '删除确认', {
    type: 'warning'
  })
  try {
    await http.delete(`/admin/orders/${row.id}`)
    ElMessage.success('删除成功')
    loadOrders()
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
function processStatusLabel(s: string) {
  return { pending: '待开始', in_progress: '进行中', completed: '已完成' }[s] || s
}
function formatDate(value: string) {
  return value ? value.slice(0, 10) : '—'
}
function formatDateTime(value: string) {
  if (!value) return '—'
  return value.replace('T', ' ').slice(0, 16)
}

onMounted(loadOrders)
</script>

<style scoped>
.a4-preview-area {
  max-height: 70vh;
  overflow: auto;
  padding: 12px;
  background: #f5f7fa;
}

.a4-order-page {
  width: 794px;
  min-height: 1123px;
  margin: 0 auto 16px;
  padding: 38px;
  background: #fff;
  color: #111;
  box-shadow: 0 2px 12px rgb(0 0 0 / 10%);
}

.print-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  border-bottom: 2px solid #111;
  padding-bottom: 28px;
  margin-bottom: 26px;
}

.print-title {
  font-size: 24px;
  font-weight: 700;
}

.print-subtitle {
  margin-top: 12px;
  font-size: 15px;
  color: #303133;
}

.print-qr-block {
  text-align: center;
}

.qr-canvas {
  display: block;
}

.print-qr-text {
  margin-top: 6px;
  font-size: 12px;
  color: #606266;
}

.print-info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  border-top: 1px solid #303133;
  border-left: 1px solid #303133;
  margin-bottom: 28px;
}

.print-info-grid div {
  min-height: 52px;
  border-right: 1px solid #303133;
  border-bottom: 1px solid #303133;
  padding: 8px 10px;
  overflow-wrap: anywhere;
}

.print-info-grid span {
  display: block;
  margin-bottom: 6px;
  color: #606266;
  font-size: 12px;
}

.print-info-grid strong {
  display: block;
  color: #111;
  font-size: 14px;
}

.print-process-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 13px;
}

.print-process-table th,
.print-process-table td {
  border: 1px solid #303133;
  padding: 8px 6px;
  text-align: center;
  vertical-align: middle;
  overflow-wrap: anywhere;
}

.print-process-table th {
  background: #f1f3f5;
  font-weight: 700;
}

.empty-process {
  color: #909399;
}
</style>
