'use client'

import { useEffect, useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  getProviderCostStats,
  getMonthlyCostStats,
  getProviders,
} from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import {
  DollarSign,
  TrendingUp,
  BarChart3,
  Table as TableIcon,
} from 'lucide-react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts'

// 设计系统色板
const CHART_COLORS = [
  '#1E3A8A', // Secondary
  '#CA8A04', // CTA
  '#0F172A', // Primary
  '#3B82F6', // Blue
  '#10B981', // Emerald
  '#8B5CF6', // Violet
  '#F59E0B', // Amber
  '#EF4444', // Red
]

const CTA_COLOR = '#CA8A04'
const AREA_FILL = '#1E3A8A'
const AREA_STROKE = '#1E3A8A'

interface Provider {
  id: number
  name: string
}

interface ProviderCostSummary {
  provider_id: number
  provider_name: string
  total_amount: number
  record_count: number
}

interface MonthlyCostSummary {
  month: string
  total_amount: number
  record_count: number
}

export function AccountingCostStats() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [providerStats, setProviderStats] = useState<ProviderCostSummary[]>([])
  const [monthlyStats, setMonthlyStats] = useState<MonthlyCostSummary[]>([])
  const [totalAmount, setTotalAmount] = useState(0)
  const [loading, setLoading] = useState(true)

  // 按供应商统计筛选
  const [providerStartDate, setProviderStartDate] = useState('')
  const [providerEndDate, setProviderEndDate] = useState('')

  // 按月趋势筛选
  const [monthlyProviderId, setMonthlyProviderId] = useState<number | undefined>()

  const { toast } = useToast()

  const loadProviders = useCallback(async () => {
    try {
      const res = await getProviders()
      setProviders(res.data || [])
    } catch {
      // 静默处理
    }
  }, [])

  const loadProviderStats = useCallback(async () => {
    try {
      const params: Record<string, string> = {}
      if (providerStartDate) params.start_date = providerStartDate
      if (providerEndDate) params.end_date = providerEndDate
      const res = await getProviderCostStats(params as any)
      const data = res.data
      setProviderStats(data?.summaries || [])
      setTotalAmount(data?.total_amount || 0)
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.response?.data?.error || '无法加载供应商统计',
      })
    }
  }, [providerStartDate, providerEndDate, toast])

  const loadMonthlyStats = useCallback(async () => {
    try {
      const params: Record<string, unknown> = {}
      if (monthlyProviderId) params.provider_id = monthlyProviderId
      const res = await getMonthlyCostStats(params as any)
      setMonthlyStats(res.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.response?.data?.error || '无法加载月度统计',
      })
    }
  }, [monthlyProviderId, toast])

  useEffect(() => {
    loadProviders()
  }, [loadProviders])

  useEffect(() => {
    setLoading(true)
    Promise.all([loadProviderStats(), loadMonthlyStats()]).finally(() =>
      setLoading(false)
    )
  }, [loadProviderStats, loadMonthlyStats])

  const formatAmount = (value: number) =>
    value.toLocaleString('zh-CN', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })

  if (loading) {
    return <StatsSkeletons />
  }

  return (
    <div className="space-y-6">
      {/* 汇总卡片 */}
      <Card className="border rounded-lg shadow-none">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
            <DollarSign className="h-4 w-4" />
            总进货金额
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold" style={{ color: CTA_COLOR }}>
            ¥{formatAmount(totalAmount)}
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            共 {providerStats.length} 个供应商，
            {providerStats.reduce((sum, s) => sum + s.record_count, 0)} 条记录
          </p>
        </CardContent>
      </Card>

      {/* 图表区域 */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* 按供应商统计 - 水平 Bar Chart */}
        <Card className="border rounded-lg shadow-none">
          <CardHeader className="pb-3">
            <CardTitle className="text-base font-medium flex items-center gap-2">
              <BarChart3 className="h-4 w-4" />
              按供应商统计
            </CardTitle>
            {/* 日期范围筛选 */}
            <div className="flex gap-3 mt-3">
              <input
                type="date"
                value={providerStartDate}
                onChange={(e) => setProviderStartDate(e.target.value)}
                className="h-9 px-3 py-1 border rounded-md bg-background text-sm cursor-pointer"
                placeholder="开始日期"
              />
              <input
                type="date"
                value={providerEndDate}
                onChange={(e) => setProviderEndDate(e.target.value)}
                className="h-9 px-3 py-1 border rounded-md bg-background text-sm cursor-pointer"
                placeholder="结束日期"
              />
            </div>
          </CardHeader>
          <CardContent>
            {providerStats.length === 0 ? (
              <EmptyChart message="暂无供应商统计数据" />
            ) : (
              <ResponsiveContainer width="100%" height={Math.max(200, providerStats.length * 48)}>
                <BarChart
                  data={providerStats}
                  layout="vertical"
                  margin={{ top: 0, right: 20, left: 0, bottom: 0 }}
                >
                  <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                  <XAxis type="number" tickFormatter={(v) => `¥${v.toLocaleString()}`} />
                  <YAxis
                    type="category"
                    dataKey="provider_name"
                    width={100}
                    tick={{ fontSize: 13 }}
                  />
                  <Tooltip
                    formatter={(value: number) => [`¥${formatAmount(value)}`, '金额']}
                    contentStyle={{ borderRadius: 8, border: '1px solid #e2e8f0' }}
                  />
                  <Bar
                    dataKey="total_amount"
                    fill={CHART_COLORS[0]}
                    radius={[0, 4, 4, 0]}
                    barSize={28}
                  />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        {/* 按月趋势 - Area Chart */}
        <Card className="border rounded-lg shadow-none">
          <CardHeader className="pb-3">
            <CardTitle className="text-base font-medium flex items-center gap-2">
              <TrendingUp className="h-4 w-4" />
              月度趋势
            </CardTitle>
            {/* 供应商筛选 */}
            <div className="mt-3">
              <select
                value={monthlyProviderId ?? ''}
                onChange={(e) =>
                  setMonthlyProviderId(
                    e.target.value ? Number(e.target.value) : undefined
                  )
                }
                className="h-9 px-3 py-1 border rounded-md bg-background text-sm cursor-pointer"
              >
                <option value="">全部供应商</option>
                {providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
          </CardHeader>
          <CardContent>
            {monthlyStats.length === 0 ? (
              <EmptyChart message="暂无月度统计数据" />
            ) : (
              <ResponsiveContainer width="100%" height={280}>
                <AreaChart
                  data={monthlyStats}
                  margin={{ top: 10, right: 20, left: 0, bottom: 0 }}
                >
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                  <YAxis tickFormatter={(v) => `¥${v.toLocaleString()}`} />
                  <Tooltip
                    formatter={(value: number) => [`¥${formatAmount(value)}`, '金额']}
                    contentStyle={{ borderRadius: 8, border: '1px solid #e2e8f0' }}
                  />
                  <Area
                    type="monotone"
                    dataKey="total_amount"
                    stroke={AREA_STROKE}
                    fill={AREA_FILL}
                    fillOpacity={0.15}
                    strokeWidth={2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* 统计数据表格 */}
      <Card className="border rounded-lg shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-base font-medium flex items-center gap-2">
            <TableIcon className="h-4 w-4" />
            供应商统计明细
          </CardTitle>
        </CardHeader>
        <CardContent>
          {providerStats.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <TableIcon className="h-10 w-10 mx-auto mb-3 opacity-50" />
              <p className="font-medium">暂无统计数据</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="pb-3 pr-4 font-medium text-muted-foreground">供应商</th>
                    <th className="pb-3 pr-4 font-medium text-muted-foreground text-right">总金额</th>
                    <th className="pb-3 font-medium text-muted-foreground text-right">记录条数</th>
                  </tr>
                </thead>
                <tbody>
                  {providerStats.map((stat) => (
                    <tr
                      key={stat.provider_id}
                      className="border-b last:border-b-0 hover:bg-accent/50 transition-colors duration-150"
                    >
                      <td className="py-3 pr-4 font-medium">{stat.provider_name}</td>
                      <td className="py-3 pr-4 text-right tabular-nums">
                        ¥{formatAmount(stat.total_amount)}
                      </td>
                      <td className="py-3 text-right tabular-nums">{stat.record_count}</td>
                    </tr>
                  ))}
                  {/* 合计行 */}
                  <tr className="border-t-2 font-semibold">
                    <td className="py-3 pr-4">合计</td>
                    <td className="py-3 pr-4 text-right tabular-nums" style={{ color: CTA_COLOR }}>
                      ¥{formatAmount(totalAmount)}
                    </td>
                    <td className="py-3 text-right tabular-nums">
                      {providerStats.reduce((sum, s) => sum + s.record_count, 0)}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

/** 空图表占位 */
function EmptyChart({ message }: { message: string }) {
  return (
    <div className="flex items-center justify-center h-[200px] text-muted-foreground">
      <div className="text-center">
        <BarChart3 className="h-10 w-10 mx-auto mb-3 opacity-50" />
        <p className="text-sm">{message}</p>
      </div>
    </div>
  )
}

/** Skeleton 加载占位 */
function StatsSkeletons() {
  return (
    <div className="space-y-6">
      {/* 汇总卡片骨架 */}
      <Card className="border rounded-lg shadow-none">
        <CardHeader className="pb-2">
          <div className="h-4 w-24 bg-muted animate-pulse rounded" />
        </CardHeader>
        <CardContent>
          <div className="h-9 w-40 bg-muted animate-pulse rounded mb-2" />
          <div className="h-4 w-48 bg-muted animate-pulse rounded" />
        </CardContent>
      </Card>

      {/* 图表骨架 */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="border rounded-lg shadow-none">
          <CardHeader className="pb-3">
            <div className="h-5 w-32 bg-muted animate-pulse rounded" />
            <div className="flex gap-3 mt-3">
              <div className="h-9 w-36 bg-muted animate-pulse rounded" />
              <div className="h-9 w-36 bg-muted animate-pulse rounded" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="h-[200px] bg-muted animate-pulse rounded" />
          </CardContent>
        </Card>
        <Card className="border rounded-lg shadow-none">
          <CardHeader className="pb-3">
            <div className="h-5 w-24 bg-muted animate-pulse rounded" />
            <div className="mt-3">
              <div className="h-9 w-40 bg-muted animate-pulse rounded" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="h-[280px] bg-muted animate-pulse rounded" />
          </CardContent>
        </Card>
      </div>

      {/* 表格骨架 */}
      <Card className="border rounded-lg shadow-none">
        <CardHeader className="pb-3">
          <div className="h-5 w-32 bg-muted animate-pulse rounded" />
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex gap-4 pb-3 border-b">
              <div className="h-4 w-20 bg-muted animate-pulse rounded" />
              <div className="h-4 w-20 bg-muted animate-pulse rounded ml-auto" />
              <div className="h-4 w-16 bg-muted animate-pulse rounded" />
            </div>
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex gap-4 py-3 border-b last:border-b-0">
                <div className="h-4 w-24 bg-muted animate-pulse rounded" />
                <div className="h-4 w-20 bg-muted animate-pulse rounded ml-auto" />
                <div className="h-4 w-12 bg-muted animate-pulse rounded" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
