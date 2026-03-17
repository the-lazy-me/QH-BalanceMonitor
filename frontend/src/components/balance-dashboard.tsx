'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { getCurrentBalance, manualCheckBalance, getBalanceHistory } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { RefreshCw, TrendingDown, TrendingUp, Plus, X, GripVertical } from 'lucide-react'
import { formatBalance, formatDate, calculateRMB, formatRMB } from '@/lib/utils'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { RMBBalanceChart } from '@/components/rmb-balance-chart'
import { RMBConsumptionChart } from '@/components/rmb-consumption-chart'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  rectSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface BalanceData {
  provider_id: number
  provider_name: string
  quota: number | null
  balance: number | null
  price_ratio: number | null
  last_checked: string | null
}

interface HistoryData {
  provider_id: number
  provider_name: string
  balance: number
  checked_at: string
}

interface DashboardCard {
  id: string
  provider_id: number
}

function SortableCard({ 
  id, 
  balance, 
  onRemove 
}: { 
  id: string
  balance: BalanceData
  onRemove: () => void 
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div ref={setNodeRef} style={style} className="relative group">
      <Card className="h-full">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="flex items-center gap-2">
            <button
              className="cursor-grab active:cursor-grabbing touch-none opacity-0 group-hover:opacity-100 transition-opacity"
              {...attributes}
              {...listeners}
            >
              <GripVertical className="h-4 w-4 text-muted-foreground" />
            </button>
            <CardTitle className="text-sm font-medium">{balance.provider_name}</CardTitle>
          </div>
          <div className="flex items-center gap-2">
            {balance.balance !== null && balance.balance < 10 ? (
              <TrendingDown className="h-4 w-4 text-red-500" />
            ) : (
              <TrendingUp className="h-4 w-4 text-green-500" />
            )}
            <button
              onClick={onRemove}
              className="opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <X className="h-4 w-4 text-muted-foreground hover:text-red-500" />
            </button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {balance.balance !== null ? formatRMB(calculateRMB(balance.balance, balance.price_ratio)) : '—'}
          </div>
          <p className="text-xs text-muted-foreground">
            {balance.balance !== null ? `额度: ${formatBalance(balance.balance)}` : ''}
          </p>
          <p className="text-xs text-muted-foreground">
            {balance.last_checked ? formatDate(balance.last_checked) : '未检查'}
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

export function BalanceDashboard() {
  const [balances, setBalances] = useState<BalanceData[]>([])
  const [history, setHistory] = useState<HistoryData[]>([])
  const [loading, setLoading] = useState(false)
  const [dashboardCards, setDashboardCards] = useState<DashboardCard[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [timeRange, setTimeRange] = useState<number | undefined>(7) // 默认7天
  const [customDays, setCustomDays] = useState<string>('') // 自定义天数
  const [isCustom, setIsCustom] = useState(false) // 是否使用自定义
  const [selectedProviders, setSelectedProviders] = useState<number[]>([]) // 选中的供应商
  const { toast } = useToast()

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const loadData = async () => {
    try {
      const [balanceRes, historyRes] = await Promise.all([
        getCurrentBalance(),
        getBalanceHistory(undefined, undefined, timeRange)
      ])
      setBalances(balanceRes.data || [])
      setHistory(historyRes.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.message || '无法加载余额数据',
      })
    }
  }

  // 从 localStorage 加载卡片配置
  useEffect(() => {
    const saved = localStorage.getItem('dashboard-cards')
    if (saved) {
      try {
        setDashboardCards(JSON.parse(saved))
      } catch (e) {
        console.error('Failed to parse saved cards:', e)
      }
    }
  }, [])

  // 保存卡片配置到 localStorage
  useEffect(() => {
    if (dashboardCards && dashboardCards.length > 0) {
      localStorage.setItem('dashboard-cards', JSON.stringify(dashboardCards))
    }
  }, [dashboardCards])

  useEffect(() => {
    loadData()
  }, [timeRange])

  const handleManualCheck = async () => {
    setLoading(true)
    try {
      await manualCheckBalance()
      toast({
        title: '检查完成',
        description: '余额已更新',
      })
      loadData()
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '检查失败',
        description: error.message || '无法检查余额',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event

    if (over && active.id !== over.id) {
      setDashboardCards((items) => {
        const oldIndex = items.findIndex((item) => item.id === active.id)
        const newIndex = items.findIndex((item) => item.id === over.id)
        return arrayMove(items, oldIndex, newIndex)
      })
    }
  }

  const handleAddProvider = (providerId: number) => {
    const newCard: DashboardCard = {
      id: `card-${Date.now()}-${providerId}`,
      provider_id: providerId,
    }
    setDashboardCards([...(dashboardCards || []), newCard])
    setDialogOpen(false)
    toast({
      title: '添加成功',
      description: '供应商卡片已添加',
    })
  }

  const handleRemoveCard = (cardId: string) => {
    setDashboardCards((dashboardCards || []).filter((card) => card.id !== cardId))
    toast({
      title: '移除成功',
      description: '卡片已移除',
    })
  }

  // 获取可添加的供应商（未在仪表盘中的）
  const availableProviders = (balances || []).filter(
    (balance) => !(dashboardCards || []).some((card) => card.provider_id === balance.provider_id)
  )

  // 准备图表数据 - 显示所有筛选后的数据
  const filteredHistory = selectedProviders.length > 0
    ? (history || []).filter(h => selectedProviders.includes(h.provider_id))
    : (history || [])

  const chartData = filteredHistory
    .sort((a, b) => new Date(a.checked_at).getTime() - new Date(b.checked_at).getTime())
    .map(item => ({
      time: new Date(item.checked_at).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      }),
      [item.provider_name]: item.balance,
    }))

  // 合并相同时间点的数据
  const mergedChartData: any[] = []
  chartData.forEach(item => {
    const existing = mergedChartData.find(d => d.time === item.time)
    if (existing) {
      Object.assign(existing, item)
    } else {
      mergedChartData.push(item)
    }
  })

  // 获取所有供应商名称用于图表线条（根据筛选条件）
  const providerNames = Array.from(new Set(filteredHistory.map(h => h.provider_name)))
  
  // 获取所有唯一的供应商列表用于筛选器
  const allProviders = Array.from(
    new Map((history || []).map(h => [h.provider_id, { id: h.provider_id, name: h.provider_name }])).values()
  ).sort((a, b) => a.name.localeCompare(b.name))
  
  // 动态计算显示的天数和数据点数
  const getTimeRangeText = () => {
    if (isCustom && customDays) {
      return `最近 ${customDays} 天`
    }
    if (!timeRange) return '全部'
    return `最近 ${timeRange} 天`
  }
  
  const dataPointsCount = mergedChartData.length

  // 处理时间范围变化
  const handleTimeRangeChange = (value: string) => {
    if (value === 'custom') {
      setIsCustom(true)
      setTimeRange(undefined)
    } else if (value === 'all') {
      setIsCustom(false)
      setTimeRange(undefined)
      setCustomDays('')
    } else {
      setIsCustom(false)
      setTimeRange(Number(value))
      setCustomDays('')
    }
  }

  // 处理自定义天数应用
  const handleApplyCustomDays = () => {
    const days = parseInt(customDays)
    if (isNaN(days) || days <= 0) {
      toast({
        variant: 'destructive',
        title: '输入错误',
        description: '请输入有效的天数（大于0的整数）',
      })
      return
    }
    setTimeRange(days)
  }

  // 处理供应商选择
  const handleProviderToggle = (providerId: number) => {
    setSelectedProviders(prev => {
      if (prev.includes(providerId)) {
        return prev.filter(id => id !== providerId)
      } else {
        return [...prev, providerId]
      }
    })
  }

  // 全选/取消全选供应商
  const handleToggleAllProviders = () => {
    if (selectedProviders.length === allProviders.length) {
      setSelectedProviders([])
    } else {
      setSelectedProviders(allProviders.map(p => p.id))
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold">余额概览</h2>
          <p className="text-muted-foreground">实时监控所有供应商余额</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            添加卡片
          </Button>
          <Button onClick={handleManualCheck} disabled={loading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            立即检查
          </Button>
        </div>
      </div>

      {(dashboardCards || []).length > 0 ? (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={(dashboardCards || []).map(c => c.id)} strategy={rectSortingStrategy}>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {(dashboardCards || []).map((card) => {
                const balance = (balances || []).find((b) => b.provider_id === card.provider_id)
                if (!balance) return null
                return (
                  <SortableCard
                    key={card.id}
                    id={card.id}
                    balance={balance}
                    onRemove={() => handleRemoveCard(card.id)}
                  />
                )
              })}
            </div>
          </SortableContext>
        </DndContext>
      ) : (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <p className="text-muted-foreground mb-4">暂无卡片，点击"添加卡片"开始</p>
            <Button onClick={() => setDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              添加卡片
            </Button>
          </CardContent>
        </Card>
      )}

      <RMBConsumptionChart />

      <RMBBalanceChart />

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>余额趋势</CardTitle>
                <CardDescription>
                  {getTimeRangeText()}的余额变化 ({dataPointsCount} 个数据点)
                  {selectedProviders.length > 0 && ` · 已选 ${selectedProviders.length} 个供应商`}
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium text-muted-foreground">时间范围:</label>
                <select
                  value={isCustom ? 'custom' : (timeRange || 'all')}
                  onChange={(e) => handleTimeRangeChange(e.target.value)}
                  className="px-3 py-1.5 border rounded-md bg-background text-sm"
                >
                  <option value={1}>最近 1 天</option>
                  <option value={3}>最近 3 天</option>
                  <option value={7}>最近 7 天</option>
                  <option value={15}>最近 15 天</option>
                  <option value={30}>最近 30 天</option>
                  <option value={90}>最近 90 天</option>
                  <option value="all">全部</option>
                  <option value="custom">自定义天数</option>
                </select>
              </div>
            </div>
            {isCustom && (
              <div className="flex items-center gap-2 justify-end">
                <label className="text-sm font-medium text-muted-foreground">自定义天数:</label>
                <input
                  type="number"
                  min="1"
                  value={customDays}
                  onChange={(e) => setCustomDays(e.target.value)}
                  placeholder="输入天数"
                  className="px-3 py-1.5 border rounded-md bg-background text-sm w-32"
                />
                <Button size="sm" onClick={handleApplyCustomDays}>
                  应用
                </Button>
              </div>
            )}
            {allProviders.length > 0 && (
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-muted-foreground">筛选供应商:</label>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    onClick={handleToggleAllProviders}
                  >
                    {selectedProviders.length === allProviders.length ? '取消全选' : '全选'}
                  </Button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {allProviders.map((provider) => (
                    <button
                      key={provider.id}
                      onClick={() => handleProviderToggle(provider.id)}
                      className={`px-3 py-1.5 text-sm rounded-md border transition-colors ${
                        selectedProviders.includes(provider.id)
                          ? 'bg-primary text-primary-foreground border-primary'
                          : 'bg-background hover:bg-accent border-border'
                      }`}
                    >
                      {provider.name}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {mergedChartData.length > 0 ? (
            <ResponsiveContainer width="100%" height={350}>
              <LineChart data={mergedChartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                {providerNames.map((name, index) => {
                  const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6']
                  return (
                    <Line
                      key={name}
                      type="monotone"
                      dataKey={name}
                      stroke={colors[index % colors.length]}
                      strokeWidth={2}
                    />
                  )
                })}
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-[350px] text-muted-foreground">
              暂无历史数据
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>所有供应商</CardTitle>
          <CardDescription>当前所有供应商的余额状态</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {(balances || []).map((balance) => (
              <div
                key={balance.provider_id}
                className="flex items-center justify-between p-4 border rounded-lg"
              >
                <div>
                  <div className="font-medium">{balance.provider_name}</div>
                  <div className="text-sm text-muted-foreground">
                    {balance.last_checked ? formatDate(balance.last_checked) : '未检查'}
                  </div>
                </div>
                <div className="text-right">
                  <div className={`text-2xl font-bold ${
                    balance.balance !== null && balance.balance < 10
                      ? 'text-red-500'
                      : 'text-green-500'
                  }`}>
                    {balance.balance !== null ? formatRMB(calculateRMB(balance.balance, balance.price_ratio)) : '—'}
                  </div>
                  {balance.balance !== null && (
                    <div className="text-sm text-muted-foreground">
                      额度: {formatBalance(balance.balance)}
                    </div>
                  )}
                </div>
              </div>
            ))}
            {(balances || []).length === 0 && (
              <div className="text-center text-muted-foreground py-8">
                暂无供应商数据，请先添加供应商
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加供应商卡片</DialogTitle>
            <DialogDescription>
              选择要添加到仪表盘的供应商
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 py-4">
            {availableProviders.length > 0 ? (
              availableProviders.map((provider) => (
                <button
                  key={provider.provider_id}
                  onClick={() => handleAddProvider(provider.provider_id)}
                  className="flex items-center justify-between p-4 border rounded-lg hover:bg-accent transition-colors text-left"
                >
                  <div>
                    <div className="font-medium">{provider.provider_name}</div>
                    <div className="text-sm text-muted-foreground">
                      {provider.balance !== null ? formatBalance(provider.balance) : '未检查'}
                    </div>
                  </div>
                  <Plus className="h-5 w-5 text-muted-foreground" />
                </button>
              ))
            ) : (
              <div className="text-center text-muted-foreground py-8">
                所有供应商都已添加到仪表盘
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
