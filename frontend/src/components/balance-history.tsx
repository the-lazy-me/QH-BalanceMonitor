'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { getBalanceHistory, getProviders } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { RefreshCw, Filter, History } from 'lucide-react'
import { formatBalance, formatDate } from '@/lib/utils'

interface HistoryRecord {
  id: number
  provider_id: number
  provider_name: string
  quota: number
  balance: number
  checked_at: string
}

interface Provider {
  id: number
  name: string
}

export function BalanceHistory() {
  const [history, setHistory] = useState<HistoryRecord[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState<number | undefined>()
  const [limit, setLimit] = useState(100)
  const { toast } = useToast()

  const loadData = async () => {
    setLoading(true)
    try {
      const [historyRes, providersRes] = await Promise.all([
        getBalanceHistory(selectedProvider, limit),
        getProviders()
      ])
      setHistory(historyRes.data || [])
      setProviders(providersRes.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.message || '无法加载历史记录',
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [selectedProvider, limit])

  const handleRefresh = () => {
    loadData()
  }

  const handleClearFilter = () => {
    setSelectedProvider(undefined)
  }

  // 按日期分组历史记录
  const groupedHistory = history.reduce((groups, record) => {
    const date = new Date(record.checked_at).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    })
    if (!groups[date]) {
      groups[date] = []
    }
    groups[date].push(record)
    return groups
  }, {} as Record<string, HistoryRecord[]>)

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold flex items-center gap-2">
            <History className="w-8 h-8" />
            历史记录
          </h2>
          <p className="text-muted-foreground">查看所有余额检查历史记录</p>
        </div>
        <Button onClick={handleRefresh} disabled={loading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      {/* 筛选器 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Filter className="w-5 h-5" />
            筛选选项
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-[200px]">
              <label className="text-sm font-medium mb-2 block">供应商</label>
              <select
                value={selectedProvider || ''}
                onChange={(e) => setSelectedProvider(e.target.value ? Number(e.target.value) : undefined)}
                className="w-full px-3 py-2 border rounded-md bg-background"
              >
                <option value="">全部供应商</option>
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.id}>
                    {provider.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex-1 min-w-[200px]">
              <label className="text-sm font-medium mb-2 block">显示数量</label>
              <select
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
                className="w-full px-3 py-2 border rounded-md bg-background"
              >
                <option value={50}>最近 50 条</option>
                <option value={100}>最近 100 条</option>
                <option value={200}>最近 200 条</option>
                <option value={500}>最近 500 条</option>
              </select>
            </div>

            {selectedProvider && (
              <Button variant="outline" onClick={handleClearFilter}>
                清除筛选
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* 统计信息 */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              记录总数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{history.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              供应商数量
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {new Set(history.map(h => h.provider_id)).size}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              检查天数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Object.keys(groupedHistory).length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 历史记录列表 */}
      <Card>
        <CardHeader>
          <CardTitle>检查记录</CardTitle>
          <CardDescription>
            {selectedProvider
              ? `显示 ${providers.find(p => p.id === selectedProvider)?.name || ''} 的历史记录`
              : '显示所有供应商的历史记录'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {Object.keys(groupedHistory).length > 0 ? (
            <div className="space-y-6">
              {Object.entries(groupedHistory)
                .sort(([dateA], [dateB]) => dateB.localeCompare(dateA))
                .map(([date, records]) => (
                  <div key={date}>
                    <h3 className="text-lg font-semibold mb-3 text-primary sticky top-0 bg-background py-2">
                      {date}
                    </h3>
                    <div className="space-y-2">
                      {records.map((record) => (
                        <div
                          key={record.id}
                          className="flex items-center justify-between p-4 border rounded-lg hover:bg-accent/50 transition-colors"
                        >
                          <div className="flex-1">
                            <div className="flex items-center gap-3">
                              <span className="font-medium text-base">
                                {record.provider_name}
                              </span>
                              <span className="text-sm text-muted-foreground">
                                {new Date(record.checked_at).toLocaleTimeString('zh-CN', {
                                  hour: '2-digit',
                                  minute: '2-digit',
                                  second: '2-digit',
                                })}
                              </span>
                            </div>
                          </div>
                          <div className="flex items-center gap-6">
                            <div className="text-right">
                              <div className="text-sm text-muted-foreground">配额</div>
                              <div className="font-medium">
                                {record.quota.toLocaleString()}
                              </div>
                            </div>
                            <div className="text-right">
                              <div className="text-sm text-muted-foreground">余额</div>
                              <div className={`text-xl font-bold ${
                                record.balance < 10
                                  ? 'text-red-500'
                                  : record.balance < 50
                                  ? 'text-orange-500'
                                  : 'text-green-500'
                              }`}>
                                {formatBalance(record.balance)}
                              </div>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
            </div>
          ) : (
            <div className="text-center text-muted-foreground py-12">
              <History className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg font-medium">暂无历史记录</p>
              <p className="text-sm mt-2">
                {selectedProvider
                  ? '该供应商暂无检查记录'
                  : '系统尚未进行余额检查，请先手动检查或等待定时任务执行'}
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

