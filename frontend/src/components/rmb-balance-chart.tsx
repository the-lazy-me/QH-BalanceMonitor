'use client'

import { useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { formatRMB, calculateRMB } from '@/lib/utils'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer, AreaChart, Area,
} from 'recharts'

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

type ViewMode = 'total' | 'individual'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']

interface RMBBalanceChartProps {
  history: HistoryData[]
  balances: BalanceData[]
}

export function RMBBalanceChart({ history, balances }: RMBBalanceChartProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('total')

  // 构建 provider_id -> price_ratio 映射
  const ratioMap = useMemo(() => {
    const map = new Map<number, number>()
    balances.forEach(b => {
      map.set(b.provider_id, b.price_ratio && b.price_ratio > 0 ? b.price_ratio : 1)
    })
    return map
  }, [balances])

  // 获取所有供应商名称
  const providerNames = useMemo(() => {
    return Array.from(
      new Map(history.map(h => [h.provider_id, h.provider_name])).values()
    ).sort()
  }, [history])

  // 按时间点聚合数据，折算为人民币
  const chartData = useMemo(() => {
    const sorted = [...history].sort(
      (a, b) => new Date(a.checked_at).getTime() - new Date(b.checked_at).getTime()
    )

    const timeMap = new Map<string, Record<string, number>>()

    sorted.forEach(item => {
      const timeKey = new Date(item.checked_at).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      })
      const ratio = ratioMap.get(item.provider_id) ?? 1
      const rmbValue = calculateRMB(item.balance, ratio)

      if (!timeMap.has(timeKey)) {
        timeMap.set(timeKey, { time: 0 } as any)
      }
      const entry = timeMap.get(timeKey)!
      entry[item.provider_name] = rmbValue
    })

    // 计算总计并设置 time 字段
    return Array.from(timeMap.entries()).map(([time, values]) => {
      const total = providerNames.reduce((sum, name) => sum + (values[name] || 0), 0)
      return { time, ...values, __total__: total }
    })
  }, [history, ratioMap, providerNames])

  // 当前总人民币余额
  const currentTotal = useMemo(() => {
    return balances.reduce((sum, b) => {
      if (b.balance === null) return sum
      return sum + calculateRMB(b.balance, b.price_ratio)
    }, 0)
  }, [balances])

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
            </CardDescription>
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
      </CardHeader>
      <CardContent>
        {chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={350}>
            {viewMode === 'total' ? (
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="totalGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis tickFormatter={(v) => `¥${(v / 1).toFixed(0)}`} />
                <Tooltip content={<CustomTooltip />} />
                <Legend formatter={() => '总计 (¥)'} />
                <Area
                  type="monotone"
                  dataKey="__total__"
                  stroke="#3b82f6"
                  fill="url(#totalGradient)"
                  strokeWidth={2}
                  name="__total__"
                />
              </AreaChart>
            ) : (
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis tickFormatter={(v) => `¥${(v / 1).toFixed(0)}`} />
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
