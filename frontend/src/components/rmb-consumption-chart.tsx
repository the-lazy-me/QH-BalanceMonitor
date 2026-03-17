'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DateRangeSelector } from '@/components/date-range-selector'
import {
  cancelConsumptionBackfillJob,
  createConsumptionBackfillJob,
  getActiveConsumptionBackfillJob,
  getBalanceConsumptionRMB,
  getConsumptionBackfillJob,
  getProviders,
} from '@/lib/api'
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

interface BackfillJobProvider {
  provider_id: number
  provider_name: string
  status: string
  current_date: string | null
  inserted_days: number
  skipped_days: number
  error_message: string | null
  started_at: string | null
  finished_at: string | null
}

interface BackfillJob {
  job_id: number
  status: string
  days_limit: number
  created_by: number
  created_at: string
  started_at: string | null
  finished_at: string | null
  total_providers: number
  pending_providers: number
  running_providers: number
  completed_providers: number
  failed_providers: number
  cancelled_providers: number
  interrupted_providers: number
  inserted_days: number
  skipped_days: number
  providers: BackfillJobProvider[]
}

type ViewMode = 'total' | 'individual'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']
const DEFAULT_RANGE_DAYS = 7

function isActiveBackfillJob(status: string | null | undefined) {
  return status === 'pending' || status === 'running' || status === 'cancelling'
}

function getJobStatusLabel(status: string | null | undefined) {
  switch (status) {
    case 'pending':
      return '排队中'
    case 'running':
      return '运行中'
    case 'cancelling':
      return '停止中'
    case 'cancelled':
      return '已取消'
    case 'completed':
      return '已完成'
    case 'completed_with_errors':
      return '部分失败'
    case 'interrupted':
      return '已中断'
    default:
      return '未知状态'
  }
}

function getProviderStatusLabel(status: string | null | undefined) {
  switch (status) {
    case 'pending':
      return '等待中'
    case 'running':
      return '运行中'
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
    case 'cancelled':
      return '已取消'
    case 'interrupted':
      return '已中断'
    default:
      return '未知状态'
  }
}

function getProviderStatusClassName(status: string | null | undefined) {
  switch (status) {
    case 'running':
      return 'bg-blue-100 text-blue-700'
    case 'completed':
      return 'bg-green-100 text-green-700'
    case 'failed':
      return 'bg-red-100 text-red-700'
    case 'cancelled':
    case 'interrupted':
      return 'bg-amber-100 text-amber-700'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

export function RMBConsumptionChart() {
  const [data, setData] = useState<ConsumptionItem[]>([])
  const [providers, setProviders] = useState<ProviderItem[]>([])
  const [viewMode, setViewMode] = useState<ViewMode>('total')
  const [dateRange, setDateRange] = useState<DateRangeValue>(() => getPresetDateRange(DEFAULT_RANGE_DAYS))
  const [selectedProviders, setSelectedProviders] = useState<number[]>([])
  const [historyDialogOpen, setHistoryDialogOpen] = useState(false)
  const [selectedHistoryProviders, setSelectedHistoryProviders] = useState<number[]>([])
  const [backfillJob, setBackfillJob] = useState<BackfillJob | null>(null)
  const [creatingBackfillJob, setCreatingBackfillJob] = useState(false)
  const [cancellingBackfillJob, setCancellingBackfillJob] = useState(false)
  const { toast } = useToast()

  const loadData = useCallback(async () => {
    try {
      const res = await getBalanceConsumptionRMB({
        startDate: dateRange.startDate,
        endDate: dateRange.endDate,
      })
      setData(res.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.response?.data?.error || error.message || '无法加载人民币额度消耗',
      })
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

  const loadActiveBackfillJob = useCallback(async (silent = false) => {
    try {
      const res = await getActiveConsumptionBackfillJob()
      const activeJob = res.data?.job ?? null
      setBackfillJob(prev => {
        if (activeJob) return activeJob
        if (prev && !isActiveBackfillJob(prev.status)) return prev
        return null
      })
      return activeJob
    } catch (error: any) {
      if (!silent) {
        toast({
          variant: 'destructive',
          title: '加载回补任务失败',
          description: error.response?.data?.error || error.message || '无法加载当前回补任务',
        })
      }
      return null
    }
  }, [toast])

  const loadBackfillJobByID = useCallback(async (jobID: number, silent = false) => {
    try {
      const res = await getConsumptionBackfillJob(jobID)
      const job = res.data as BackfillJob
      setBackfillJob(job)
      return job
    } catch (error: any) {
      if (!silent) {
        toast({
          variant: 'destructive',
          title: '加载回补进度失败',
          description: error.response?.data?.error || error.message || '无法加载回补任务详情',
        })
      }
      return null
    }
  }, [toast])

  useEffect(() => {
    void loadData()
    void loadProviders()
    void loadActiveBackfillJob(true)
  }, [loadActiveBackfillJob, loadData, loadProviders])

  useEffect(() => {
    if (!historyDialogOpen) return
    void loadActiveBackfillJob(true)
  }, [historyDialogOpen, loadActiveBackfillJob])

  useEffect(() => {
    if (!historyDialogOpen || !backfillJob || !isActiveBackfillJob(backfillJob.status)) {
      return
    }

    const intervalID = window.setInterval(() => {
      void loadBackfillJobByID(backfillJob.job_id, true).then((job) => {
        if (job && !isActiveBackfillJob(job.status)) {
          void loadData()
        }
      })
    }, 3000)

    return () => {
      window.clearInterval(intervalID)
    }
  }, [backfillJob, historyDialogOpen, loadBackfillJobByID, loadData])

  useEffect(() => {
    if (!historyDialogOpen || isActiveBackfillJob(backfillJob?.status) || selectedHistoryProviders.length > 0) {
      return
    }

    const defaultSelection = providers
      .filter(provider => provider.enabled && provider.query_mode === 'default')
      .map(provider => provider.id)

    if (defaultSelection.length > 0) {
      setSelectedHistoryProviders(defaultSelection)
    }
  }, [backfillJob?.status, historyDialogOpen, providers, selectedHistoryProviders.length])

  const allProviders = useMemo(() => {
    return Array.from(
      new Map(data.map(item => [item.provider_id, { id: item.provider_id, name: item.provider_name }])).values()
    ).sort((left, right) => left.name.localeCompare(right.name))
  }, [data])

  const availableHistoryProviders = useMemo(() => {
    return providers
      .filter(provider => provider.enabled && provider.query_mode === 'default')
      .sort((left, right) => left.name.localeCompare(right.name))
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
      ? data.filter(item => selectedProviders.includes(item.provider_id))
      : data

    const timeMap = new Map<string, Record<string, number>>()

    filtered.forEach(item => {
      if (!timeMap.has(item.date)) {
        timeMap.set(item.date, {})
      }
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
  }, [data, providerSeries, selectedProviders])

  const totalConsumption = useMemo(() => {
    const filtered = selectedProviders.length > 0
      ? data.filter(item => selectedProviders.includes(item.provider_id))
      : data
    return filtered.reduce((sum, item) => sum + item.consumption, 0)
  }, [data, selectedProviders])

  const handleProviderToggle = (providerID: number) => {
    setSelectedProviders(prev =>
      prev.includes(providerID) ? prev.filter(id => id !== providerID) : [...prev, providerID]
    )
  }

  const handleToggleAll = () => {
    setSelectedProviders(prev =>
      prev.length === allProviders.length ? [] : allProviders.map(provider => provider.id)
    )
  }

  const handleHistoryProviderToggle = (providerID: number) => {
    if (isActiveBackfillJob(backfillJob?.status)) return

    setSelectedHistoryProviders(prev =>
      prev.includes(providerID) ? prev.filter(id => id !== providerID) : [...prev, providerID]
    )
  }

  const handleToggleAllHistoryProviders = () => {
    if (isActiveBackfillJob(backfillJob?.status)) return

    setSelectedHistoryProviders(prev =>
      prev.length === availableHistoryProviders.length
        ? []
        : availableHistoryProviders.map(provider => provider.id)
    )
  }

  const handleDialogOpenChange = (open: boolean) => {
    setHistoryDialogOpen(open)
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

    setCreatingBackfillJob(true)
    try {
      const res = await createConsumptionBackfillJob({
        provider_ids: selectedHistoryProviders,
      })
      const job = (res.data?.job ?? null) as BackfillJob | null
      setBackfillJob(job)
      toast({
        title: '回补任务已创建',
        description: `已提交 ${selectedHistoryProviders.length} 个供应商的后台回补任务`,
      })
    } catch (error: any) {
      if (error.response?.status === 409) {
        const job = (error.response?.data?.job ?? null) as BackfillJob | null
        setBackfillJob(job)
        toast({
          variant: 'destructive',
          title: '已有任务在运行',
          description: '当前只允许同时运行一个历史回补任务，请等待现有任务结束',
        })
      } else {
        toast({
          variant: 'destructive',
          title: '创建任务失败',
          description: error.response?.data?.error || error.message || '无法创建历史回补任务',
        })
      }
    } finally {
      setCreatingBackfillJob(false)
    }
  }

  const handleCancelBackfill = async () => {
    if (!backfillJob) return

    setCancellingBackfillJob(true)
    try {
      await cancelConsumptionBackfillJob(backfillJob.job_id)
      toast({
        title: '停止请求已提交',
        description: '任务会在当前正在执行的日期完成后停止',
      })
      await loadBackfillJobByID(backfillJob.job_id, true)
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '停止任务失败',
        description: error.response?.data?.error || error.message || '无法停止回补任务',
      })
    } finally {
      setCancellingBackfillJob(false)
    }
  }

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload?.length) return null

    const visibleItems = payload
      .filter((entry: any) => typeof entry.value === 'number' && entry.value > 0)
      .sort((left: any, right: any) => right.value - left.value)

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

  const activeJobRunning = isActiveBackfillJob(backfillJob?.status)
  const historyButtonDisabled = availableHistoryProviders.length === 0 && !backfillJob
  const historyButtonLabel = activeJobRunning ? '查看回补进度' : '获取历史数据'

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
                disabled={historyButtonDisabled}
              >
                <Database className="mr-2 h-4 w-4" />
                {historyButtonLabel}
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
                <YAxis tickFormatter={(value) => `¥${value.toFixed(0)}`} />
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
                <YAxis tickFormatter={(value) => `¥${value.toFixed(0)}`} />
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
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>获取历史数据</DialogTitle>
            <DialogDescription>
              回补任务会在后端后台运行。不同供应商并行处理，单个供应商按天串行向前回补最近 365 天，前端关闭后仍会继续。
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5">
            <div className="flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                仅展示启用中的默认模式供应商
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleToggleAllHistoryProviders}
                disabled={activeJobRunning || availableHistoryProviders.length === 0}
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
                    disabled={activeJobRunning}
                    className={`px-3 py-1.5 text-sm rounded-md border transition-colors ${
                      selectedHistoryProviders.includes(provider.id)
                        ? 'bg-primary text-primary-foreground border-primary'
                        : 'bg-background hover:bg-accent border-border'
                    } ${activeJobRunning ? 'cursor-not-allowed opacity-70' : ''}`}
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

            {backfillJob && (
              <div className="rounded-lg border p-4 space-y-4">
                <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                  <div>
                    <div className="font-medium">当前任务</div>
                    <div className="text-sm text-muted-foreground">
                      任务 #{backfillJob.job_id} · {getJobStatusLabel(backfillJob.status)} · 共 {backfillJob.total_providers} 个供应商
                    </div>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {activeJobRunning && (
                      <span className="inline-flex items-center gap-2">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        后台运行中
                      </span>
                    )}
                    {!activeJobRunning && backfillJob.finished_at && `结束于 ${formatLongDate(backfillJob.finished_at.slice(0, 10))}`}
                  </div>
                </div>

                <div className="grid gap-3 md:grid-cols-4">
                  <div className="rounded-lg border p-3">
                    <div className="text-sm text-muted-foreground">已写入天数</div>
                    <div className="text-2xl font-bold">{backfillJob.inserted_days}</div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-sm text-muted-foreground">已跳过天数</div>
                    <div className="text-2xl font-bold">{backfillJob.skipped_days}</div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-sm text-muted-foreground">已完成供应商</div>
                    <div className="text-2xl font-bold">{backfillJob.completed_providers}/{backfillJob.total_providers}</div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-sm text-muted-foreground">异常供应商</div>
                    <div className="text-2xl font-bold">
                      {backfillJob.failed_providers + backfillJob.cancelled_providers + backfillJob.interrupted_providers}
                    </div>
                  </div>
                </div>

                <div className="rounded-md bg-muted/40 p-3 text-sm text-muted-foreground">
                  运行中 {backfillJob.running_providers} 个，等待中 {backfillJob.pending_providers} 个，失败 {backfillJob.failed_providers} 个，已取消 {backfillJob.cancelled_providers} 个，中断 {backfillJob.interrupted_providers} 个。
                </div>

                <div className="space-y-2">
                  <div className="text-sm font-medium">供应商进度</div>
                  <div className="max-h-64 overflow-y-auto space-y-2">
                    {backfillJob.providers.map((provider) => (
                      <div key={provider.provider_id} className="rounded-lg border p-3">
                        <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                          <div>
                            <div className="font-medium">{provider.provider_name}</div>
                            <div className="text-sm text-muted-foreground">
                              当前日期 {provider.current_date || '尚未开始'} · 已写入 {provider.inserted_days} 天 · 已跳过 {provider.skipped_days} 天
                            </div>
                            {provider.error_message && (
                              <div className="text-sm text-red-600 mt-1">{provider.error_message}</div>
                            )}
                          </div>
                          <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${getProviderStatusClassName(provider.status)}`}>
                            {getProviderStatusLabel(provider.status)}
                          </span>
                        </div>
                      </div>
                    ))}
                    {backfillJob.providers.length === 0 && (
                      <div className="text-sm text-muted-foreground">暂无供应商进度数据</div>
                    )}
                  </div>
                </div>
              </div>
            )}
          </div>

          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => handleDialogOpenChange(false)}>
              关闭
            </Button>
            {activeJobRunning ? (
              <Button
                variant="destructive"
                onClick={handleCancelBackfill}
                disabled={cancellingBackfillJob || backfillJob?.status === 'cancelling'}
              >
                {cancellingBackfillJob || backfillJob?.status === 'cancelling' ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <StopCircle className="mr-2 h-4 w-4" />
                )}
                停止任务
              </Button>
            ) : (
              <Button
                onClick={handleStartBackfill}
                disabled={creatingBackfillJob || availableHistoryProviders.length === 0 || selectedHistoryProviders.length === 0}
              >
                {creatingBackfillJob ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Database className="mr-2 h-4 w-4" />
                )}
                开始回补
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  )
}
