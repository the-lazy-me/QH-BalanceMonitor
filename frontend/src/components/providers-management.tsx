'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { getProviders, createProvider, updateProvider, deleteProvider, saveCustomQuery, getCustomQuery, testProvider } from '@/lib/api'
import { CustomQueryForm, CustomQueryData } from '@/components/custom-query-form'
import { useToast } from '@/hooks/use-toast'
import { Loader2, Pencil, Play, Plus, Server, Trash2 } from 'lucide-react'

interface Provider {
  id: number
  name: string
  upstream_url: string
  authorization: string
  new_api_user: string
  enabled: boolean
  alert_threshold: number
  query_mode: string
  price_ratio: number
  created_at: string
  updated_at: string
}

interface ProviderTestResult {
  success: boolean
  provider_id: number
  provider_name: string
  query_mode: string
  quota?: number
  balance?: number
  checked_at?: string
  error?: string
}

export function ProvidersManagement() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [testingProviderId, setTestingProviderId] = useState<number | null>(null)
  const [testResults, setTestResults] = useState<Record<number, ProviderTestResult>>({})
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    upstream_url: '',
    authorization: '',
    new_api_user: '1',
    enabled: true,
    alert_threshold: 10.0,
    query_mode: 'default',
    price_ratio: 1,
  })
  const [customQueryData, setCustomQueryData] = useState<CustomQueryData | null>(null)
  const customQueryLatestRef = useRef<CustomQueryData | null>(null)
  const handleCustomQueryChange = useCallback((data: CustomQueryData) => {
    customQueryLatestRef.current = data
  }, [])
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

  const handleOpenDialog = async (provider?: Provider) => {
    if (provider) {
      setEditingProvider(provider)
      setFormData({
        name: provider.name,
        upstream_url: provider.upstream_url,
        authorization: provider.authorization,
        new_api_user: provider.new_api_user,
        enabled: provider.enabled,
        alert_threshold: provider.alert_threshold,
        query_mode: provider.query_mode || 'default',
        price_ratio: provider.price_ratio ?? 1,
      })

      // Load custom query config if in custom mode
      if (provider.query_mode === 'custom') {
        try {
          const response = await getCustomQuery(provider.id)
          setCustomQueryData(response.data)
        } catch {
          setCustomQueryData(null)
        }
      } else {
        setCustomQueryData(null)
      }
    } else {
      setEditingProvider(null)
      setFormData({
        name: '',
        upstream_url: '',
        authorization: '',
        new_api_user: '1',
        enabled: true,
        alert_threshold: 10.0,
        query_mode: 'default',
        price_ratio: 1,
      })
      setCustomQueryData(null)
    }
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!formData.price_ratio || formData.price_ratio <= 0) {
      toast({
        variant: 'destructive',
        title: '校验失败',
        description: '价格比例必须为正数',
      })
      return
    }

    try {
      let providerId: number

      if (editingProvider) {
        await updateProvider(editingProvider.id, formData)
        providerId = editingProvider.id
        toast({ title: '更新成功', description: '供应商信息已更新' })
      } else {
        const response = await createProvider(formData)
        providerId = response.data.id
        toast({ title: '创建成功', description: '新供应商已添加' })
      }

      // Save custom query config if in custom mode
      if (formData.query_mode === 'custom' && (customQueryLatestRef.current || customQueryData)) {
        try {
          await saveCustomQuery(providerId, (customQueryLatestRef.current || customQueryData)!)
        } catch (error: any) {
          toast({
            variant: 'destructive',
            title: '自定义查询配置保存失败',
            description: error.response?.data?.error || error.message,
          })
        }
      }

      setDialogOpen(false)
      setCustomQueryData(null)
      customQueryLatestRef.current = null
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
      setTestResults((prev) => {
        const next = { ...prev }
        delete next[id]
        return next
      })
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

  const handleTestProvider = async (provider: Provider) => {
    setTestingProviderId(provider.id)

    try {
      const response = await testProvider(provider.id)
      const result = response.data as ProviderTestResult

      setTestResults((prev) => ({
        ...prev,
        [provider.id]: result,
      }))

      if (result.success) {
        toast({
          title: '测试成功',
          description: `${provider.name} 当前余额 ${result.balance ?? 0}`,
        })
      } else {
        toast({
          variant: 'destructive',
          title: '测试失败',
          description: result.error || `${provider.name} 测试失败`,
        })
      }
    } catch (error: any) {
      const message = error.response?.data?.error || error.message || '无法测试供应商'

      setTestResults((prev) => ({
        ...prev,
        [provider.id]: {
          success: false,
          provider_id: provider.id,
          provider_name: provider.name,
          query_mode: provider.query_mode,
          checked_at: new Date().toISOString(),
          error: message,
        },
      }))

      toast({
        variant: 'destructive',
        title: '测试失败',
        description: message,
      })
    } finally {
      setTestingProviderId(null)
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
        {providers.map((provider) => {
          const testResult = testResults[provider.id]
          const testedAt = testResult?.checked_at
            ? new Date(testResult.checked_at).toLocaleString('zh-CN')
            : null

          return (
            <Card key={provider.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <Server className="h-5 w-5 text-primary" />
                    <CardTitle className="text-lg">{provider.name}</CardTitle>
                  </div>
                  <div className="flex items-center gap-1">
                    <div className={`px-2 py-1 rounded text-xs font-medium ${
                      provider.query_mode === 'custom'
                        ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                        : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                    }`}>
                      {provider.query_mode === 'custom' ? '自定义' : '默认'}
                    </div>
                    <div className={`px-2 py-1 rounded text-xs font-medium ${
                      provider.enabled
                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
                    }`}>
                      {provider.enabled ? '启用' : '禁用'}
                    </div>
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
                  <div>
                    <span className="text-muted-foreground">兑换比例:</span>
                    <span className="ml-2 font-mono">{provider.price_ratio ?? 1} 元/额度</span>
                  </div>
                </div>
                <div className="grid grid-cols-3 gap-2 mt-4">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTestProvider(provider)}
                    disabled={testingProviderId === provider.id}
                  >
                    {testingProviderId === provider.id ? (
                      <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                    ) : (
                      <Play className="mr-2 h-3 w-3" />
                    )}
                    测试
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleOpenDialog(provider)}
                  >
                    <Pencil className="mr-2 h-3 w-3" />
                    编辑
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => handleDelete(provider.id)}
                  >
                    <Trash2 className="mr-2 h-3 w-3" />
                    删除
                  </Button>
                </div>
                {testResult && (
                  <div
                    className={`mt-3 rounded-md border px-3 py-2 text-sm ${
                      testResult.success
                        ? 'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-300'
                        : 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-300'
                    }`}
                  >
                    <div className="font-medium">{testResult.success ? '测试成功' : '测试失败'}</div>
                    {testResult.success ? (
                      <div className="mt-1 space-y-1">
                        <div>余额：<span className="font-mono">{testResult.balance ?? 0}</span></div>
                        {provider.query_mode === 'default' && (
                          <div>Quota：<span className="font-mono">{testResult.quota ?? 0}</span></div>
                        )}
                      </div>
                    ) : (
                      <div className="mt-1 break-all">{testResult.error}</div>
                    )}
                    {testedAt && (
                      <div className="mt-1 text-xs opacity-80">测试时间：{testedAt}</div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )
        })}
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

      <Dialog open={dialogOpen} onOpenChange={(open) => {
        setDialogOpen(open)
        if (!open) {
          setCustomQueryData(null)
          customQueryLatestRef.current = null
        }
      }}>
        <DialogContent className="sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>{editingProvider ? '编辑供应商' : '添加供应商'}</DialogTitle>
            <DialogDescription>
              {editingProvider ? '修改供应商信息' : '添加新的上游余额供应商'}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4 max-h-[70vh] overflow-y-auto">
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
            <div className="grid gap-2">
              <Label htmlFor="price_ratio">兑换比例（元/额度）</Label>
              <Input
                id="price_ratio"
                type="number"
                step="any"
                min="0"
                value={formData.price_ratio}
                onChange={(e) => {
                  const val = parseFloat(e.target.value)
                  setFormData({ ...formData, price_ratio: isNaN(val) ? 0 : val })
                }}
                placeholder="1"
              />
              <p className="text-xs text-muted-foreground">
                1 站内额度需要多少人民币（例如：1 额度 = 0.5 元，则填 0.5）
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
            <div className="grid gap-2">
              <Label>查询模式</Label>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant={formData.query_mode === 'default' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setFormData({ ...formData, query_mode: 'default' })}
                >
                  默认
                </Button>
                <Button
                  type="button"
                  variant={formData.query_mode === 'custom' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setFormData({ ...formData, query_mode: 'custom' })}
                >
                  自定义
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                默认模式使用 New API 查询，自定义模式使用自定义 curl 查询
              </p>
            </div>
            {formData.query_mode === 'default' && (
              <>
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
              </>
            )}
            {formData.query_mode === 'custom' && (
              <CustomQueryForm
                providerId={editingProvider?.id}
                visible={formData.query_mode === 'custom'}
                onChange={handleCustomQueryChange}
                initialData={customQueryData}
              />
            )}
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
