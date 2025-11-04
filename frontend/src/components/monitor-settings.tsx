'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getMonitorSettings, updateMonitorSettings } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { Save, Clock, Bell } from 'lucide-react'

interface MonitorSettings {
  id: number
  cron_expression: string
  alert_count: number
}

const CRON_EXAMPLES = [
  { label: '每小时', value: '@hourly', desc: '每小时执行一次' },
  { label: '每30分钟', value: '*/30 * * * *', desc: '每30分钟执行一次' },
  { label: '每15分钟', value: '*/15 * * * *', desc: '每15分钟执行一次' },
  { label: '每天', value: '@daily', desc: '每天午夜执行一次' },
  { label: '每12小时', value: '0 */12 * * *', desc: '每12小时执行一次' },
]

export function MonitorSettings() {
  const [settings, setSettings] = useState<MonitorSettings>({
    id: 1,
    cron_expression: '@hourly',
    alert_count: 3,
  })
  const [loading, setLoading] = useState(false)
  const { toast } = useToast()

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async () => {
    try {
      const response = await getMonitorSettings()
      if (response.data) {
        setSettings(response.data)
      }
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.message || '无法加载监控设置',
      })
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      await updateMonitorSettings(settings)
      toast({
        title: '保存成功',
        description: '监控设置已更新，将在下次检查时生效',
      })
      loadSettings()
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '保存失败',
        description: error.response?.data?.error || '无法保存监控设置',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleCronExampleClick = (value: string) => {
    setSettings({ ...settings, cron_expression: value })
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold">监控设置</h2>
        <p className="text-muted-foreground">配置余额检查频率和告警策略</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            检查时间设置
          </CardTitle>
          <CardDescription>
            使用 Cron 表达式配置余额检查的执行频率
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="cron_expression">Cron 表达式</Label>
              <Input
                id="cron_expression"
                value={settings.cron_expression}
                onChange={(e) =>
                  setSettings({ ...settings, cron_expression: e.target.value })
                }
                placeholder="@hourly"
                required
              />
              <p className="text-sm text-muted-foreground">
                格式：分 时 日 月 周 或使用预设值（如 @hourly、@daily）
              </p>
            </div>

            <div className="space-y-2">
              <Label>常用表达式</Label>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                {CRON_EXAMPLES.map((example) => (
                  <button
                    key={example.value}
                    type="button"
                    onClick={() => handleCronExampleClick(example.value)}
                    className={`p-3 border rounded-lg hover:bg-accent transition-colors text-left ${
                      settings.cron_expression === example.value ? 'border-primary bg-accent' : ''
                    }`}
                  >
                    <div className="font-medium text-sm">{example.label}</div>
                    <div className="text-xs text-muted-foreground">{example.value}</div>
                    <div className="text-xs text-muted-foreground mt-1">{example.desc}</div>
                  </button>
                ))}
              </div>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell className="h-5 w-5" />
            告警设置
          </CardTitle>
          <CardDescription>
            配置余额不足时的告警策略
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="alert_count">24小时内最大告警次数</Label>
              <Input
                id="alert_count"
                type="number"
                min="1"
                max="100"
                value={settings.alert_count}
                onChange={(e) =>
                  setSettings({ ...settings, alert_count: parseInt(e.target.value) || 1 })
                }
                required
              />
              <p className="text-sm text-muted-foreground">
                当余额低于阈值时，24小时内最多发送的告警邮件次数
              </p>
            </div>

            <div className="flex gap-2">
              <Button type="submit" disabled={loading}>
                <Save className="mr-2 h-4 w-4" />
                {loading ? '保存中...' : '保存设置'}
              </Button>
              <Button type="button" variant="outline" onClick={loadSettings}>
                重置
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Cron 表达式说明</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="text-sm space-y-2">
            <p className="font-medium">标准格式：</p>
            <code className="block bg-muted p-2 rounded">分 时 日 月 周</code>
            
            <p className="font-medium mt-4">示例：</p>
            <ul className="list-disc list-inside space-y-1 text-muted-foreground">
              <li><code>*/15 * * * *</code> - 每15分钟</li>
              <li><code>0 * * * *</code> - 每小时的整点</li>
              <li><code>0 */2 * * *</code> - 每2小时</li>
              <li><code>0 0 * * *</code> - 每天午夜</li>
              <li><code>0 9 * * *</code> - 每天早上9点</li>
            </ul>

            <p className="font-medium mt-4">预设值：</p>
            <ul className="list-disc list-inside space-y-1 text-muted-foreground">
              <li><code>@hourly</code> - 每小时</li>
              <li><code>@daily</code> - 每天</li>
              <li><code>@weekly</code> - 每周</li>
              <li><code>@monthly</code> - 每月</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

