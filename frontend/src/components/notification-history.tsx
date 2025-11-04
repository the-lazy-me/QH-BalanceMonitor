'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { getNotifications } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { Bell, AlertTriangle } from 'lucide-react'
import { formatDate } from '@/lib/utils'

interface Notification {
  id: number
  provider_id: number
  provider_name: string
  balance: number
  message: string
  sent_at: string
}

export function NotificationHistory() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const { toast } = useToast()

  useEffect(() => {
    loadNotifications()
  }, [])

  const loadNotifications = async () => {
    try {
      const response = await getNotifications(100)
      setNotifications(response.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.message || '无法加载通知历史',
      })
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold">通知历史</h2>
        <p className="text-muted-foreground">查看所有告警通知记录</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell className="h-5 w-5" />
            告警记录
          </CardTitle>
          <CardDescription>
            共 {notifications.length} 条通知记录
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {notifications.map((notification) => (
              <div
                key={notification.id}
                className="flex items-start gap-4 p-4 border rounded-lg hover:bg-accent/50 transition-colors"
              >
                <div className="mt-1">
                  <AlertTriangle className="h-5 w-5 text-orange-500" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <div className="font-medium">{notification.provider_name}</div>
                      <div className="text-sm text-muted-foreground mt-1">
                        {notification.message}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <div className="text-lg font-bold text-orange-500">
                        {notification.balance.toFixed(2)}
                      </div>
                      <div className="text-xs text-muted-foreground mt-1">
                        {formatDate(notification.sent_at)}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}

            {notifications.length === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Bell className="h-12 w-12 mb-4 opacity-50" />
                <p>暂无通知记录</p>
                <p className="text-sm mt-1">当余额低于阈值时会自动发送告警</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

