'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { BalanceDashboard } from '@/components/balance-dashboard'
import { ProvidersManagement } from '@/components/providers-management'
import { SMTPSettings } from '@/components/smtp-settings'
import { MonitorSettings } from '@/components/monitor-settings'
import { NotificationHistory } from '@/components/notification-history'
import { BalanceHistory } from '@/components/balance-history'
import { Accounting } from '@/components/accounting'
import { Activity, Settings, Bell, Server, Mail, Clock, LogOut, Loader2, History, BookOpen } from 'lucide-react'
import { checkInitStatus, getToken, removeToken } from '@/lib/api'

export default function Home() {
  const router = useRouter()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const checkAuth = async () => {
      try {
        // 检查是否有 token
        const token = getToken()
        if (!token) {
          // 没有 token，检查系统是否已初始化
          const response = await checkInitStatus()
          if (response.data.initialized) {
            // 已初始化，跳转到登录页
            router.push('/login')
          } else {
            // 未初始化，跳转到初始化页
            router.push('/initialize')
          }
          return
        }

        // 有 token，允许访问
        setLoading(false)
      } catch (error) {
        console.error('认证检查失败:', error)
        // 出错时也跳转到登录页
        router.push('/login')
      }
    }

    checkAuth()
  }, [router])

  const handleLogout = () => {
    removeToken()
    router.push('/login')
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-slate-600 dark:text-slate-400">加载中...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
      <div className="container mx-auto py-8 px-4">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-4xl font-bold text-slate-900 dark:text-white mb-2">
              余额监控系统
            </h1>
            <p className="text-slate-600 dark:text-slate-400">
              实时监控供应商余额，智能告警通知
            </p>
          </div>
          <Button
            variant="outline"
            onClick={handleLogout}
            className="flex items-center gap-2"
          >
            <LogOut className="w-4 h-4" />
            退出登录
          </Button>
        </div>

        <Tabs defaultValue="dashboard" className="space-y-6">
          <TabsList className="grid w-full grid-cols-6 lg:w-auto">
            <TabsTrigger value="dashboard" className="flex items-center gap-2">
              <Activity className="w-4 h-4" />
              <span className="hidden sm:inline">仪表盘</span>
            </TabsTrigger>
            <TabsTrigger value="providers" className="flex items-center gap-2">
              <Server className="w-4 h-4" />
              <span className="hidden sm:inline">供应商</span>
            </TabsTrigger>
            <TabsTrigger value="history" className="flex items-center gap-2">
              <History className="w-4 h-4" />
              <span className="hidden sm:inline">历史记录</span>
            </TabsTrigger>
            <TabsTrigger value="notifications" className="flex items-center gap-2">
              <Bell className="w-4 h-4" />
              <span className="hidden sm:inline">通知</span>
            </TabsTrigger>
            <TabsTrigger value="settings" className="flex items-center gap-2">
              <Settings className="w-4 h-4" />
              <span className="hidden sm:inline">设置</span>
            </TabsTrigger>
            <TabsTrigger value="accounting" className="flex items-center gap-2">
              <BookOpen className="w-4 h-4" />
              <span className="hidden sm:inline">记账</span>
            </TabsTrigger>
          </TabsList>

          <TabsContent value="dashboard" className="space-y-6">
            <BalanceDashboard />
          </TabsContent>

          <TabsContent value="providers" className="space-y-6">
            <ProvidersManagement />
          </TabsContent>

          <TabsContent value="history" className="space-y-6">
            <BalanceHistory />
          </TabsContent>

          <TabsContent value="notifications" className="space-y-6">
            <NotificationHistory />
          </TabsContent>

          <TabsContent value="accounting" className="space-y-6">
            <Accounting />
          </TabsContent>

          <TabsContent value="settings" className="space-y-6">
            <Tabs defaultValue="monitor" className="space-y-6">
              <TabsList>
                <TabsTrigger value="monitor" className="flex items-center gap-2">
                  <Clock className="w-4 h-4" />
                  监控设置
                </TabsTrigger>
                <TabsTrigger value="smtp" className="flex items-center gap-2">
                  <Mail className="w-4 h-4" />
                  邮件设置
                </TabsTrigger>
              </TabsList>

              <TabsContent value="monitor">
                <MonitorSettings />
              </TabsContent>

              <TabsContent value="smtp">
                <SMTPSettings />
              </TabsContent>
            </Tabs>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

