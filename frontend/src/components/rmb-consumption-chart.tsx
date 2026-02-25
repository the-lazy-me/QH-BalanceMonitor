'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { getBalanceConsumptionRMB } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { formatRMB } from '@/lib/utils'
import {
  BarChart, Bar, Cell, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer,
} from 'recharts'

interface ConsumptionItem {
  provider_id: number
  provider_name: string
  consumption: number
  date: string
}

type ViewMode = 'total' | 'individual'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']

export function RMBConsumptionChart() {
  const [data, setData] = useState<ConsumptionItem[]>([])
  const [viewMode, setViewMode] = useState<ViewMode>('total')
  const [timeRange, setTimeRange] = useState(7)
  const [selectedProviders, setSelectedProviders] = useState<number[]>([])
  const { toast } = useToast()

  const loadData = useCallback(async () => {
    try {
      const res = await getBalanceConsumptionRMB(timeRange)
      setData(res.data || [])
    } catch (error: any) {
      toast({ variant: 'destructive', title: '加载失败', description: error.message })
    }
  }, [timeRange])

  useEffect(() => { loadData() }, [loadData])

  const allProviders = useMemo(() => {
    return Array.from(
      new Map(data.map(h => [h.provider_id, { id: h.provider_id, name: h.provider_name }])).values()
    ).sort((a, b) => a.name.localeCompare(b.name))
  }, [data])

  const providerNames = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? data.filter(h => selectedProviders.includes(h.provider_id))
      : data
    return Array.from(new Map(filtered.map(h => [h.provider_id, h.provider_name])).values()).sort()
  }, [data, selectedProviders])

  const chartData = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? data.filter(h => selectedProviders.includes(h.provider_id))
      : data

    const timeMap = new Map<string, Record<string, number>>()

    filtered.forEach(item => {
      // date 格式为 YYYY-MM-DD，转为 MM/DD 显示
      const parts = item.date.split('-')
      const dateKey = `${parts[1]}/${parts[2]}`
      if (!timeMap.has(dateKey)) timeMap.set(dateKey, {})
      const entry = timeMap.get(dateKey)!
      entry[item.provider_name] = (entry[item.provider_name] || 0) + item.consumption
    })

    return Array.from(timeMap.entries()).map(([time, values]) => {
      const total = providerNames.reduce((sum, name) => sum + (values[name] || 0), 0)
      return { time, ...values, __total__: total }
    })
  }, [data, selectedProviders, providerNames])

  const totalConsumption = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? data.filter(h => selectedProviders.includes(h.provider_id))
      : data
    return filtered.reduce((sum, item) => sum + item.consumption, 0)
  }, [data, selectedProviders])

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

  // 根据消耗值在 绿→黄→橙→红 之间插值
  const getConsumptionColor = useMemo(() => {
    const maxVal = Math.max(...chartData.map(d => d.__total__ || 0), 1)
    return (value: number) => {
      const ratio = Math.min(value / maxVal, 1)
      if (ratio < 0.25) return '#22c55e' // green
      if (ratio < 0.5) return '#eab308'  // yellow
      if (ratio < 0.75) return '#f97316' // orange
      return '#ef4444'                    // red
    }
  }, [chartData])

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
                人民币额度消耗
                <span className="text-lg font-normal text-muted-foreground">
                  累计消耗 {formatRMB(totalConsumption)}
                </span>
              </CardTitle>
              <CardDescription>
                每日余额下降量（折算人民币） · {chartData.length} 天
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
                  总览
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
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis tickFormatter={(v) => `¥${v.toFixed(0)}`} />
                <Tooltip content={<CustomTooltip />} />
                <Legend formatter={() => '总消耗 (¥)'} />
                <Bar dataKey="__total__" radius={[4, 4, 0, 0]} name="__total__">
                  {chartData.map((entry, index) => (
                    <Cell key={index} fill={getConsumptionColor(entry.__total__ || 0)} />
                  ))}
                </Bar>
              </BarChart>
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
            暂无消耗数据
          </div>
        )}
      </CardContent>
    </Card>
  )
}
