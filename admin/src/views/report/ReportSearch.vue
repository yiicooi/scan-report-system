<template>
  <el-card>
    <template #header>报工查询</template>
    <el-form :model="query" inline>
      <el-form-item label="操作人">
        <el-select v-model="query.user_id" clearable placeholder="全部人员" style="width:160px">
          <el-option v-for="u in users" :key="u.id" :label="u.name" :value="u.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="日期">
        <el-date-picker v-model="query.dateRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始" end-placeholder="结束" />
      </el-form-item>
      <el-form-item label="内部单号">
        <el-input v-model="query.internal_no" clearable style="width:180px" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="search">查询</el-button>
      </el-form-item>
    </el-form>

    <el-table :data="records" v-loading="loading" stripe style="margin-top:16px">
      <el-table-column prop="reported_at" label="报工时间" width="160">
        <template #default="{ row }">{{ row.reported_at?.replace('T',' ').slice(0,19) }}</template>
      </el-table-column>
      <el-table-column label="操作人" prop="user.name" width="100" />
      <el-table-column label="工序" prop="order_process.display_name" />
      <el-table-column prop="received_qty" label="接收" width="80" align="center" />
      <el-table-column prop="completed_qty" label="完成" width="80" align="center" />
      <el-table-column prop="scrap_qty" label="报废" width="80" align="center" />
      <el-table-column prop="note" label="备注" />
    </el-table>
  </el-card>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import http from '@/api/http'

const loading = ref(false)
const records = ref<any[]>([])
const users = ref<any[]>([])
const query = reactive({ user_id: null, dateRange: [], internal_no: '' })

async function search() {
  loading.value = true
  // TODO: 调用报工明细查询 API
  loading.value = false
}

onMounted(async () => {
  const res: any = await http.get('/admin/users')
  users.value = res.data || []
})
</script>
