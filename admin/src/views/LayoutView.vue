<template>
  <el-container style="height:100vh">
    <!-- 侧边栏 -->
    <el-aside width="220px" style="background:#001529">
      <div style="color:#fff;font-size:16px;font-weight:bold;padding:20px 16px;border-bottom:1px solid #1a3550">
        扫码报工系统
      </div>
      <el-menu router :default-active="route.path" background-color="#001529" text-color="#aaa" active-text-color="#fff">
        <el-menu-item-group title="单据管理">
          <el-menu-item index="/orders"><el-icon><Document /></el-icon>工单列表</el-menu-item>
          <el-menu-item index="/report-progress"><el-icon><DataAnalysis /></el-icon>报工进度</el-menu-item>
          <el-menu-item index="/report-search"><el-icon><Search /></el-icon>报工查询</el-menu-item>
        </el-menu-item-group>
        <el-menu-item-group title="工艺管理">
          <el-menu-item index="/processes"><el-icon><Tools /></el-icon>工序模板</el-menu-item>
          <el-menu-item index="/process-aliases"><el-icon><List /></el-icon>流程模板</el-menu-item>
        </el-menu-item-group>
        <el-menu-item-group title="系统管理">
          <el-menu-item index="/users"><el-icon><User /></el-icon>用户管理</el-menu-item>
          <el-menu-item index="/departments"><el-icon><OfficeBuilding /></el-icon>部门管理</el-menu-item>
          <el-menu-item index="/roles"><el-icon><Lock /></el-icon>角色权限</el-menu-item>
        </el-menu-item-group>
      </el-menu>
    </el-aside>

    <el-container>
      <!-- 顶部 -->
      <el-header style="display:flex;align-items:center;justify-content:flex-end;background:#fff;border-bottom:1px solid #eee">
        <el-dropdown @command="handleCommand">
          <span style="cursor:pointer">{{ auth.userInfo?.name }} <el-icon><ArrowDown /></el-icon></span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>

      <!-- 主内容 -->
      <el-main style="background:#f5f7fa;padding:20px">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    auth.logout()
    router.push('/login')
  }
}
</script>

