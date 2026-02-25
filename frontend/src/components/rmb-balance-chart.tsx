'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { getBalanceHistoryRMB, getCurrentBalance } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { formatRMB, calculateRMB } from '@/lib/utils'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer, AreaChart, Area,
} from 'recharts'

interface RMBHistoryItem {
  provider_id: number
  provider_name: string
  rmb_balance: number
  checked_at: string
}

interface BalanceData {
  provider_id: number
  provider_name: string
  balance: number | null
  price_ratio: number | null
}

type ViewMode = 'total' | 'individual'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']

export function RMBBalanceChart() {
  const [history, setHistory] = useState<RMBHistoryItem[]>([])
  const [balances, setBalances] = useState<BalanceData[]>([])
  const [viewMode, setViewMode] = useState<ViewMode>('total')
  const [timeRange, setTimeRange] = useState(7)
  const [selectedProviders, setSelectedProviders] = useState<number[]>([])
  const { toast } = useToast()

  const loadData = useCallback(async () => {
    try {
      const [historyRes, balanceRes] = await Promise.all([
        getBalanceHistoryRMB(timeRange),
        getCurrentBalance(),
      ])
      setHistory(historyRes.data || [])
      setBalances(balanceRes.data || [])
    } catch (error: any) {
      toast({ variant: 'destructive', title: '加载失败', description: error.message })
    }
  }, [timeRange])

  useEffect(() => { loadData() }, [loadData])

  // 所有供应商列表（从历史数据中提取）
  const allProviders = useMemo(() => {
    return Array.from(
      new Map(history.map(h => [h.provider_id, { id: h.provider_id, name: h.provider_name }])).values()
    ).sort((a, b) => a.name.localeCompare(b.name))
  }, [history])

  const providerNames = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? history.filter(h => selectedProviders.includes(h.provider_id))
      : history
    return Array.from(new Map(filtered.map(h => [h.provider_id, h.provider_name])).values()).sort()
  }, [history, selectedProviders])

  // 按时间点聚合，后端已算好人民币
  const chartData = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? history.filter(h => selectedProviders.includes(h.provider_id))
      : history

    const timeMap = new Map<string, Record<string, number>>()

    filtered.forEach(item => {
      const timeKey = new Date(item.checked_at).toLocaleString('zh-CN', {
        month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
      })

      if (!timeMap.has(timeKey)) {
        timeMap.set(timeKey, {})
      }
      const entry = timeMap.get(timeKey)!
      entry[item.provider_name] = item.rmb_balance
    })

    return Array.from(timeMap.entries()).map(([time, values]) => {
      const total = providerNames.reduce((sum, name) => sum + (values[name] || 0), 0)
      return { time, ...values, __total__: total }
    })
  }, [history, selectedProviders, providerNames])

  // 当前总人民币余额
  const currentTotal = useMemo(() => {
    return balances.reduce((sum, b) => {
      if (b.balance === null) return sum
      return sum + calculateRMB(b.balance, b.price_ratio)
    }, 0)
  }, [balances])

  const handleProviderToggle = (id: number) => {
    setSelectedProviders(prev =>
      prev.includes(id) ? prev.filter(p => p !== id) : [...prev, id]
    )
  }

  const handleToggleAll = () => {
    setSelectedProviders(prev =>
      prev.length === allProviders.length ? [] : allProviders.map(p => p.id)
    )
  }

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload?.length) return null
    return (
      <div className="bg-popover border rounded-lg shadow-lg p-3 text-sm">
        <p className="font-medium mb-1.5">{label}</p>
        {payload.map((entry: any) => (
          <div key={entry.dataKey} className="flex justify-between gap-4" style={{ color: entry.color }}>
            <span>{entry.dataKey === '__total__' ? '总计' : entry.dataKey}</span>
            <span className="font-medium">{formatRMB(entry.value)}</span>
          </div>
        ))}
      </div>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                人民币余额走势
                <span className="text-lg font-normal text-muted-foreground">
                  当前合计 {formatRMB(currentTotal)}
                </span>
              </CardTitle>
              <CardDescription>
                各供应商余额按倍率折算为人民币 · {chartData.length} 个数据点
                {selectedProviders.length > 0 && ` · 已选 ${selectedProviders.length} 个供应商`}
              </CardDescription>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium text-muted-foreground">时间:</label>
                <select
                  value={timeRange}
                  onChange={(e) => setTimeRange(Number(e.target.value))}
                  className="px-3 py-1.5 border rounded-md bg-background text-sm"
                >
                  <option value={1}>1 天</option>
                  <option value={3}>3 天</option>
                  <option value={7}>7 天</option>
                  <option value={15}>15 天</option>
                  <option value={30}>30 天</option>
                  <option value={90}>90 天</option>
                </select>
              </div>
              <div className="flex gap-1 bg-muted rounded-lg p-1">
                <Button
                  variant={viewMode === 'total' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setViewMode('total')}
                >
                  总计
                </Button>
                <Button
                  variant={viewMode === 'individual' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setViewMode('individual')}
                >
                  各家
                </Button>
              </div>
            </div>
          </div>
          {allProviders.length > 0 && (
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium text-muted-foreground">筛选供应商:</label>
                <Button variant="outline" size="sm" onClick={handleToggleAll}>
                  {selectedProviders.length === allProviders.length ? '取消全选' : '全选'}
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {allProviders.map((provider) => (
                  <button
                    key={provider.id}
                    onClick={() => handleProviderToggle(provider.id)}
                    className={`px-3 py-1.5 text-sm rounded-md border transition-colors cursor-pointer ${
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
        {chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={350}>
            {viewMode === 'total' ? (
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="rmbTotalGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis tickFormatter={(v) => `¥${v.toFixed(0)}`} />
                <Tooltip content={<CustomTooltip />} />
                <Legend formatter={() => '总计 (¥)'} />
                <Area
                  type="monotone"
                  dataKey="__total__"
                  stroke="#3b82f6"
                  fill="url(#rmbTotalGradient)"
                  strokeWidth={2}
                  name="__total__"
                />
              </AreaChart>
            ) : (
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis tickFormatter={(v) => `¥${v.toFixed(0)}`} />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                {providerNames.map((name, i) => (
                  <Line
                    key={name}
                    type="monotone"
                    dataKey={name}
                    stroke={COLORS[i % COLORS.length]}
                    strokeWidth={2}
                    dot={false}
                    connectNulls
                  />
                ))}
              </LineChart>
            )}
          </ResponsiveContainer>
        ) : (
          <div className="flex items-center justify-center h-[350px] text-muted-foreground">
            暂无历史数据
          </div>
        )}
      </CardContent>
    </Card>
  )
}
