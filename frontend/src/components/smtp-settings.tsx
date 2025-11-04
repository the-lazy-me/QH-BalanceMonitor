'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { getSMTPSettings, updateSMTPSettings, testSMTP } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { Mail, Send } from 'lucide-react'

interface SMTPSettingsData {
  id: number
  enabled: boolean
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  from_email: string
  to_emails: string
}

export function SMTPSettings() {
  const [settings, setSettings] = useState<SMTPSettingsData>({
    id: 1,
    enabled: false,
    smtp_host: '',
    smtp_port: 587,
    smtp_username: '',
    smtp_password: '',
    from_email: '',
    to_emails: '',
  })
  const [loading, setLoading] = useState(false)
  const [testing, setTesting] = useState(false)
  const { toast } = useToast()

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async () => {
    try {
      const response = await getSMTPSettings()
      setSettings(response.data)
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.message || '无法加载 SMTP 设置',
      })
    }
  }

  const handleSave = async () => {
    setLoading(true)
    try {
      await updateSMTPSettings(settings)
      toast({
        title: '保存成功',
        description: 'SMTP 设置已更新',
      })
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '保存失败',
        description: error.message || '无法保存 SMTP 设置',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    try {
      await testSMTP()
      toast({
        title: '测试成功',
        description: '测试邮件已发送，请检查收件箱',
      })
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '测试失败',
        description: error.message || '无法发送测试邮件',
      })
    } finally {
      setTesting(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold">SMTP 通知设置</h2>
        <p className="text-muted-foreground">配置邮件告警通知</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Mail className="h-5 w-5" />
                SMTP 配置
              </CardTitle>
              <CardDescription>配置 SMTP 服务器用于发送告警邮件</CardDescription>
            </div>
            <Switch
              checked={settings.enabled}
              onCheckedChange={(checked) => setSettings({ ...settings, enabled: checked })}
            />
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="smtp_host">SMTP 服务器</Label>
              <Input
                id="smtp_host"
                value={settings.smtp_host}
                onChange={(e) => setSettings({ ...settings, smtp_host: e.target.value })}
                placeholder="smtp.example.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp_port">端口</Label>
              <Input
                id="smtp_port"
                type="number"
                value={settings.smtp_port}
                onChange={(e) => setSettings({ ...settings, smtp_port: parseInt(e.target.value) })}
                placeholder="587"
              />
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="smtp_username">用户名</Label>
              <Input
                id="smtp_username"
                value={settings.smtp_username}
                onChange={(e) => setSettings({ ...settings, smtp_username: e.target.value })}
                placeholder="username@example.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp_password">密码</Label>
              <Input
                id="smtp_password"
                type="password"
                value={settings.smtp_password}
                onChange={(e) => setSettings({ ...settings, smtp_password: e.target.value })}
                placeholder="••••••••"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="from_email">发件人邮箱</Label>
            <Input
              id="from_email"
              type="email"
              value={settings.from_email}
              onChange={(e) => setSettings({ ...settings, from_email: e.target.value })}
              placeholder="noreply@example.com"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="to_emails">收件人邮箱</Label>
            <Input
              id="to_emails"
              value={settings.to_emails}
              onChange={(e) => setSettings({ ...settings, to_emails: e.target.value })}
              placeholder="user1@example.com, user2@example.com"
            />
            <p className="text-xs text-muted-foreground">
              多个邮箱请用逗号分隔
            </p>
          </div>

          <div className="flex gap-2 pt-4">
            <Button onClick={handleSave} disabled={loading}>
              {loading ? '保存中...' : '保存设置'}
            </Button>
            <Button
              variant="outline"
              onClick={handleTest}
              disabled={testing || !settings.enabled}
            >
              <Send className="mr-2 h-4 w-4" />
              {testing ? '发送中...' : '发送测试邮件'}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>使用说明</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>• 启用 SMTP 通知后，当供应商余额低于阈值时，系统会自动发送告警邮件</p>
          <p>• 告警阈值在每个供应商的设置中单独配置，可以为不同供应商设置不同的阈值</p>
          <p>• 为避免频繁发送邮件，每个供应商在 24 小时内最多发送一次告警</p>
          <p>• 建议使用专用的 SMTP 服务（如阿里云邮件推送、SendGrid 等）以确保送达率</p>
          <p>• 常见 SMTP 端口：25（明文）、465（SSL）、587（TLS）</p>
        </CardContent>
      </Card>
    </div>
  )
}

