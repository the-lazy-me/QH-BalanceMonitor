'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DateRangeSelector } from '@/components/date-range-selector'
import { backfillBalanceConsumptionHistoryDay, getBalanceConsumptionRMB, getProviders } from '@/lib/api'
import { DateRangeValue, formatChartDate, formatDateRangeLabel, formatLongDate, getPresetDateRange } from '@/lib/date-range'
import { useToast } from '@/hooks/use-toast'
import { formatRMB } from '@/lib/utils'
import { Database, Loader2, StopCircle } from 'lucide-react'
import {
  BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer,
} from 'recharts'

interface ConsumptionItem {
  provider_id: number
  provider_name: string
  consumption: number
  date: string
}

interface ProviderItem {
  id: number
  name: string
  enabled: boolean
  query_mode: string
}

interface BackfillProgress {
  currentProviderName: string
  currentDate: string
  inserted: number
  skipped: number
  failed: number
  status: 'idle' | 'running' | 'stopped' | 'completed'
  logs: string[]
}

type ViewMode = 'total' | 'individual'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']
const MAX_BACKFILL_DAYS = 365
const DEFAULT_RANGE_DAYS = 7

const initialBackfillProgress: BackfillProgress = {
  currentProviderName: '',
  currentDate: '',
  inserted: 0,
  skipped: 0,
  failed: 0,
  status: 'idle',
  logs: [],
}

export function RMBConsumptionChart() {
  const [data, setData] = useState<ConsumptionItem[]>([])
  const [providers, setProviders] = useState<ProviderItem[]>([])
  const [viewMode, setViewMode] = useState<ViewMode>('total')
  const [dateRange, setDateRange] = useState<DateRangeValue>(() => getPresetDateRange(DEFAULT_RANGE_DAYS))
  const [selectedProviders, setSelectedProviders] = useState<number[]>([])
  const [historyDialogOpen, setHistoryDialogOpen] = useState(false)
  const [selectedHistoryProviders, setSelectedHistoryProviders] = useState<number[]>([])
  const [backfillProgress, setBackfillProgress] = useState<BackfillProgress>(initialBackfillProgress)
  const { toast } = useToast()
  const stopBackfillRef = useRef(false)

  const loadData = useCallback(async () => {
    try {
      const res = await getBalanceConsumptionRMB({
        startDate: dateRange.startDate,
        endDate: dateRange.endDate,
      })
      setData(res.data || [])
    } catch (error: any) {
      toast({ variant: 'destructive', title: '加载失败', description: error.message })
    }
  }, [dateRange.endDate, dateRange.startDate, toast])

  const loadProviders = useCallback(async () => {
    try {
      const res = await getProviders()
      setProviders(res.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载供应商失败',
        description: error.response?.data?.error || error.message || '无法加载供应商列表',
      })
    }
  }, [toast])

  useEffect(() => {
    loadData()
    loadProviders()
  }, [loadData, loadProviders])

  const allProviders = useMemo(() => {
    return Array.from(
      new Map(data.map(h => [h.provider_id, { id: h.provider_id, name: h.provider_name }])).values()
    ).sort((a, b) => a.name.localeCompare(b.name))
  }, [data])

  const availableHistoryProviders = useMemo(() => {
    return providers
      .filter(provider => provider.enabled && provider.query_mode === 'default')
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [providers])

  const activeProviders = useMemo(() => {
    if (selectedProviders.length === 0) return allProviders

    const selectedProviderSet = new Set(selectedProviders)
    return allProviders.filter(provider => selectedProviderSet.has(provider.id))
  }, [allProviders, selectedProviders])

  const providerSeries = useMemo(() => {
    return activeProviders.map((provider, index) => ({
      ...provider,
      key: `provider_${provider.id}`,
      color: COLORS[index % COLORS.length],
    }))
  }, [activeProviders])

  const showYearOnAxis = useMemo(() => {
    const years = new Set([
      dateRange.startDate.slice(0, 4),
      dateRange.endDate.slice(0, 4),
      ...data.map(item => item.date.slice(0, 4)),
    ])

    return years.size > 1
  }, [data, dateRange.endDate, dateRange.startDate])

  const chartData = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? data.filter(h => selectedProviders.includes(h.provider_id))
      : data

    const timeMap = new Map<string, Record<string, number>>()

    filtered.forEach(item => {
      if (!timeMap.has(item.date)) timeMap.set(item.date, {})
      const entry = timeMap.get(item.date)!
      const providerKey = `provider_${item.provider_id}`
      entry[providerKey] = (entry[providerKey] || 0) + item.consumption
    })

    return Array.from(timeMap.entries())
      .sort(([leftDate], [rightDate]) => leftDate.localeCompare(rightDate))
      .map(([date, values]) => {
      const total = providerSeries.reduce((sum, provider) => sum + (values[provider.key] || 0), 0)
      return { date, ...values, __total__: total }
    })
  }, [data, selectedProviders, providerSeries])

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

  const appendBackfillLog = (message: string) => {
    setBackfillProgress(prev => ({
      ...prev,
      logs: [message, ...prev.logs].slice(0, 12),
    }))
  }

  const handleHistoryProviderToggle = (id: number) => {
    if (backfillProgress.status === 'running') return

    setSelectedHistoryProviders(prev =>
      prev.includes(id) ? prev.filter(providerId => providerId !== id) : [...prev, id]
    )
  }

  const handleToggleAllHistoryProviders = () => {
    if (backfillProgress.status === 'running') return

    setSelectedHistoryProviders(prev =>
      prev.length === availableHistoryProviders.length ? [] : availableHistoryProviders.map(provider => provider.id)
    )
  }

  const resetBackfillProgress = () => {
    setBackfillProgress(initialBackfillProgress)
    stopBackfillRef.current = false
  }

  const handleDialogOpenChange = (open: boolean) => {
    if (backfillProgress.status === 'running') {
      setHistoryDialogOpen(true)
      return
    }

    setHistoryDialogOpen(open)
    if (open && selectedHistoryProviders.length === 0) {
      setSelectedHistoryProviders(availableHistoryProviders.map(provider => provider.id))
    }
    if (!open) {
      resetBackfillProgress()
    }
  }

  const formatBackfillDate = (date: Date) => {
    const year = date.getFullYear()
    const month = `${date.getMonth() + 1}`.padStart(2, '0')
    const day = `${date.getDate()}`.padStart(2, '0')
    return `${year}-${month}-${day}`
  }

  const handleStartBackfill = async () => {
    if (selectedHistoryProviders.length === 0) {
      toast({
        variant: 'destructive',
        title: '请选择供应商',
        description: '至少选择一个默认模式供应商后再开始获取历史数据',
      })
      return
    }

    const providersToBackfill = availableHistoryProviders.filter(provider => selectedHistoryProviders.includes(provider.id))
    if (providersToBackfill.length === 0) {
      toast({
        variant: 'destructive',
        title: '无可用供应商',
        description: '当前没有可回补历史数据的默认模式供应商',
      })
      return
    }

    stopBackfillRef.current = false
    setBackfillProgress({
      ...initialBackfillProgress,
      status: 'running',
    })
    const today = new Date()
    today.setHours(0, 0, 0, 0)

    for (const provider of providersToBackfill) {
      let providerFailed = false

      for (let offset = 0; offset < MAX_BACKFILL_DAYS; offset += 1) {
        if (stopBackfillRef.current) {
          setBackfillProgress(prev => ({
            ...prev,
            status: 'stopped',
            logs: [
              prev.currentProviderName
                ? `已手动停止，停在 ${prev.currentProviderName} ${prev.currentDate}`
                : '已手动停止',
              ...prev.logs,
            ].slice(0, 12),
          }))
          await loadData()
          toast({ title: '已停止', description: '历史回补已按要求停止' })
          return
        }

        const currentDate = new Date(today)
        currentDate.setDate(currentDate.getDate() - offset)
        const date = formatBackfillDate(currentDate)

        setBackfillProgress(prev => ({
          ...prev,
          currentProviderName: provider.name,
          currentDate: date,
        }))

        try {
          const res = await backfillBalanceConsumptionHistoryDay({
            provider_id: provider.id,
            date,
          })

          const status = res.data?.status === 'inserted' ? 'inserted' : 'skipped'
          const quotaConsumed = res.data?.quota_consumed ?? 0

          setBackfillProgress(prev => ({
            ...prev,
            inserted: prev.inserted + (status === 'inserted' ? 1 : 0),
            skipped: prev.skipped + (status === 'skipped' ? 1 : 0),
          }))

          appendBackfillLog(
            `${provider.name} ${date} ${status === 'inserted' ? '已写入' : '已跳过'} · quota ${quotaConsumed}`
          )
        } catch (error: any) {
          const message = error.response?.data?.error || error.message || '获取失败'
          providerFailed = true
          setBackfillProgress(prev => ({
            ...prev,
            failed: prev.failed + 1,
          }))
          appendBackfillLog(`${provider.name} ${date} 失败 · ${message}`)
          break
        }
      }

      if (providerFailed) {
        continue
      }
    }

    setBackfillProgress(prev => ({
      ...prev,
      status: 'completed',
    }))
    await loadData()
    toast({
      title: '历史回补完成',
      description: `已完成 ${providersToBackfill.length} 个供应商的历史回补检查`,
    })
  }

  const handleStopBackfill = () => {
    stopBackfillRef.current = true
  }

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload?.length) return null

    const visibleItems = payload
      .filter((entry: any) => typeof entry.value === 'number' && entry.value > 0)
      .sort((a: any, b: any) => b.value - a.value)

    const total = visibleItems.reduce((sum: number, entry: any) => sum + entry.value, 0)

    return (
      <div className="bg-popover border rounded-lg shadow-lg p-3 text-sm">
        <p className="font-medium mb-1.5">{formatLongDate(label)}</p>
        {visibleItems.map((entry: any) => (
          <div key={entry.dataKey} className="flex justify-between gap-4" style={{ color: entry.color }}>
            <span>{entry.name}</span>
            <span className="font-medium">{formatRMB(entry.value)}</span>
          </div>
        ))}
        {visibleItems.length > 1 && (
          <div className="flex justify-between gap-4 border-t mt-2 pt-2 font-medium">
            <span>总计</span>
            <span>{formatRMB(total)}</span>
          </div>
        )}
      </div>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                人民币额度消耗
                <span className="text-lg font-normal text-muted-foreground">
                  累计消耗 {formatRMB(totalConsumption)}
                </span>
              </CardTitle>
              <CardDescription>
                每日余额下降量（折算人民币） · {formatDateRangeLabel(dateRange)} · {chartData.length} 天
                {selectedProviders.length > 0 && ` · 已选 ${selectedProviders.length} 个供应商`}
              </CardDescription>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <Button
                variant="outline"
                onClick={() => handleDialogOpenChange(true)}
                disabled={availableHistoryProviders.length === 0}
              >
                <Database className="mr-2 h-4 w-4" />
                获取历史数据
              </Button>
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

          <DateRangeSelector value={dateRange} onChange={setDateRange} />

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
                <XAxis
                  dataKey="date"
                  tickFormatter={(value) => formatChartDate(String(value), showYearOnAxis)}
                  minTickGap={24}
                />
                <YAxis tickFormatter={(v) => `¥${v.toFixed(0)}`} />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                {providerSeries.map((provider, index) => (
                  <Bar
                    key={provider.key}
                    dataKey={provider.key}
                    name={provider.name}
                    stackId="consumption"
                    fill={provider.color}
                    radius={index === providerSeries.length - 1 ? [4, 4, 0, 0] : [0, 0, 0, 0]}
                  />
                ))}
              </BarChart>
            ) : (
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis
                  dataKey="date"
                  tickFormatter={(value) => formatChartDate(String(value), showYearOnAxis)}
                  minTickGap={24}
                />
                <YAxis tickFormatter={(v) => `¥${v.toFixed(0)}`} />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                {providerSeries.map((provider) => (
                  <Line
                    key={provider.key}
                    type="monotone"
                    dataKey={provider.key}
                    name={provider.name}
                    stroke={provider.color}
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

      <Dialog open={historyDialogOpen} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>获取历史数据</DialogTitle>
            <DialogDescription>
              先选择默认模式供应商，再逐个逐天向前回补最近 {MAX_BACKFILL_DAYS} 天。系统会自动跳过已记录的日期。
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                仅展示启用中的默认模式供应商
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleToggleAllHistoryProviders}
                disabled={availableHistoryProviders.length === 0 || backfillProgress.status === 'running'}
              >
                {selectedHistoryProviders.length === availableHistoryProviders.length ? '取消全选' : '全选'}
              </Button>
            </div>

            {availableHistoryProviders.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {availableHistoryProviders.map((provider) => (
                  <button
                    key={provider.id}
                    onClick={() => handleHistoryProviderToggle(provider.id)}
                    disabled={backfillProgress.status === 'running'}
                    className={`px-3 py-1.5 text-sm rounded-md border transition-colors ${
                      selectedHistoryProviders.includes(provider.id)
                        ? 'bg-primary text-primary-foreground border-primary'
                        : 'bg-background hover:bg-accent border-border'
                    } ${backfillProgress.status === 'running' ? 'cursor-not-allowed opacity-70' : ''}`}
                  >
                    {provider.name}
                  </button>
                ))}
              </div>
            ) : (
              <div className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
                暂无可回补历史数据的供应商。只有启用中的默认模式供应商会出现在这里。
              </div>
            )}

            <div className="grid gap-3 md:grid-cols-3">
              <div className="rounded-lg border p-3">
                <div className="text-sm text-muted-foreground">已写入</div>
                <div className="text-2xl font-bold">{backfillProgress.inserted}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-sm text-muted-foreground">已跳过</div>
                <div className="text-2xl font-bold">{backfillProgress.skipped}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-sm text-muted-foreground">失败供应商</div>
                <div className="text-2xl font-bold">{backfillProgress.failed}</div>
              </div>
            </div>

            <div className="rounded-lg border p-4 space-y-2">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-medium">当前进度</div>
                  <div className="text-sm text-muted-foreground">
                    {backfillProgress.currentProviderName
                      ? `${backfillProgress.currentProviderName} · ${backfillProgress.currentDate || '等待开始'}`
                      : '尚未开始'}
                  </div>
                </div>
                <div className="text-sm text-muted-foreground">
                  {backfillProgress.status === 'running' && (
                    <span className="inline-flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      运行中
                    </span>
                  )}
                  {backfillProgress.status === 'completed' && '已完成'}
                  {backfillProgress.status === 'stopped' && '已停止'}
                </div>
              </div>

              <div className="max-h-48 overflow-y-auto rounded-md bg-muted/40 p-3 text-sm">
                {backfillProgress.logs.length > 0 ? (
                  <div className="space-y-2">
                    {backfillProgress.logs.map((log, index) => (
                      <div key={`${log}-${index}`} className="font-mono text-xs leading-5">
                        {log}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-muted-foreground">
                    开始后会在这里显示最近的回补记录。
                  </div>
                )}
              </div>
            </div>
          </div>

          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => handleDialogOpenChange(false)}
              disabled={backfillProgress.status === 'running'}
            >
              关闭
            </Button>
            {backfillProgress.status === 'running' ? (
              <Button variant="destructive" onClick={handleStopBackfill}>
                <StopCircle className="mr-2 h-4 w-4" />
                停止
              </Button>
            ) : (
              <Button onClick={handleStartBackfill} disabled={availableHistoryProviders.length === 0}>
                <Database className="mr-2 h-4 w-4" />
                开始回补
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  )
}
