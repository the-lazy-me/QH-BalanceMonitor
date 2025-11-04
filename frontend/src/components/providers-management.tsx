'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { getProviders, createProvider, updateProvider, deleteProvider } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { Plus, Pencil, Trash2, Server } from 'lucide-react'

interface Provider {
  id: number
  name: string
  upstream_url: string
  authorization: string
  new_api_user: string
  enabled: boolean
  alert_threshold: number
  created_at: string
  updated_at: string
}

export function ProvidersManagement() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    upstream_url: '',
    authorization: '',
    new_api_user: '1',
    enabled: true,
    alert_threshold: 10.0,
  })
  const { toast } = useToast()

  const loadProviders = async () => {
    try {
      const response = await getProviders()
      setProviders(response.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.message || '无法加载供应商列表',
      })
    }
  }

  useEffect(() => {
    loadProviders()
  }, [])

  const handleOpenDialog = (provider?: Provider) => {
    if (provider) {
      setEditingProvider(provider)
      setFormData({
        name: provider.name,
        upstream_url: provider.upstream_url,
        authorization: provider.authorization,
        new_api_user: provider.new_api_user,
        enabled: provider.enabled,
        alert_threshold: provider.alert_threshold,
      })
    } else {
      setEditingProvider(null)
      setFormData({
        name: '',
        upstream_url: '',
        authorization: '',
        new_api_user: '1',
        enabled: true,
        alert_threshold: 10.0,
      })
    }
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    try {
      if (editingProvider) {
        await updateProvider(editingProvider.id, formData)
        toast({
          title: '更新成功',
          description: '供应商信息已更新',
        })
      } else {
        await createProvider(formData)
        toast({
          title: '创建成功',
          description: '新供应商已添加',
        })
      }
      setDialogOpen(false)
      loadProviders()
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '操作失败',
        description: error.message || '无法保存供应商信息',
      })
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个供应商吗？')) return

    try {
      await deleteProvider(id)
      toast({
        title: '删除成功',
        description: '供应商已删除',
      })
      loadProviders()
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '删除失败',
        description: error.message || '无法删除供应商',
      })
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold">供应商管理</h2>
          <p className="text-muted-foreground">管理上游余额供应商</p>
        </div>
        <Button onClick={() => handleOpenDialog()}>
          <Plus className="mr-2 h-4 w-4" />
          添加供应商
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {providers.map((provider) => (
          <Card key={provider.id}>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-2">
                  <Server className="h-5 w-5 text-primary" />
                  <CardTitle className="text-lg">{provider.name}</CardTitle>
                </div>
                <div className={`px-2 py-1 rounded text-xs font-medium ${
                  provider.enabled
                    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                    : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
                }`}>
                  {provider.enabled ? '启用' : '禁用'}
                </div>
              </div>
              <CardDescription className="line-clamp-1">{provider.upstream_url}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-muted-foreground">Authorization:</span>
                  <div className="font-mono text-xs mt-1 truncate">{provider.authorization}</div>
                </div>
                <div>
                  <span className="text-muted-foreground">New-API-User:</span>
                  <span className="ml-2 font-mono">{provider.new_api_user}</span>
                </div>
              </div>
              <div className="flex gap-2 mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  className="flex-1"
                  onClick={() => handleOpenDialog(provider)}
                >
                  <Pencil className="mr-2 h-3 w-3" />
                  编辑
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  className="flex-1"
                  onClick={() => handleDelete(provider.id)}
                >
                  <Trash2 className="mr-2 h-3 w-3" />
                  删除
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {providers.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Server className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground mb-4">暂无供应商</p>
            <Button onClick={() => handleOpenDialog()}>
              <Plus className="mr-2 h-4 w-4" />
              添加第一个供应商
            </Button>
          </CardContent>
        </Card>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>{editingProvider ? '编辑供应商' : '添加供应商'}</DialogTitle>
            <DialogDescription>
              {editingProvider ? '修改供应商信息' : '添加新的上游余额供应商'}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">名称</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="供应商名称"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="upstream_url">上游地址</Label>
              <Input
                id="upstream_url"
                value={formData.upstream_url}
                onChange={(e) => setFormData({ ...formData, upstream_url: e.target.value })}
                placeholder="https://example.com"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="authorization">Authorization</Label>
              <Input
                id="authorization"
                value={formData.authorization}
                onChange={(e) => setFormData({ ...formData, authorization: e.target.value })}
                placeholder="Bearer token 或其他认证信息"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new_api_user">New-API-User</Label>
              <Input
                id="new_api_user"
                value={formData.new_api_user}
                onChange={(e) => setFormData({ ...formData, new_api_user: e.target.value })}
                placeholder="1"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="alert_threshold">告警阈值</Label>
              <Input
                id="alert_threshold"
                type="number"
                step="0.01"
                value={formData.alert_threshold}
                onChange={(e) => setFormData({ ...formData, alert_threshold: parseFloat(e.target.value) || 10.0 })}
                placeholder="10.00"
              />
              <p className="text-xs text-muted-foreground">
                当余额低于此值时发送告警邮件
              </p>
            </div>
            <div className="flex items-center justify-between">
              <Label htmlFor="enabled">启用状态</Label>
              <Switch
                id="enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) => setFormData({ ...formData, enabled: checked })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSubmit}>
              {editingProvider ? '保存' : '添加'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

