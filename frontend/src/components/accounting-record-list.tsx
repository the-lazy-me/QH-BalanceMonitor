'use client'

import { useEffect, useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  getAccountingRecords,
  deleteAccountingRecord,
  getProviders,
} from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { Plus, Pencil, Trash2, Loader2, FileText } from 'lucide-react'
import { AccountingRecordForm } from './accounting-record-form'

interface Provider {
  id: number
  name: string
}

interface AccountingRecord {
  id: number
  provider_id: number
  provider_name: string
  amount: number
  date: string
  note: string
  created_at: string
  updated_at: string
}

export function AccountingRecordList() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [records, setRecords] = useState<AccountingRecord[]>([])
  const [loading, setLoading] = useState(false)

  // 筛选状态
  const [filterProviderId, setFilterProviderId] = useState<number | undefined>()
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  // 表单弹窗状态
  const [formOpen, setFormOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<AccountingRecord | null>(null)

  // 删除确认弹窗状态
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deletingRecord, setDeletingRecord] = useState<AccountingRecord | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  const { toast } = useToast()

  // 加载供应商列表
  const loadProviders = useCallback(async () => {
    try {
      const res = await getProviders()
      setProviders(res.data || [])
    } catch {
      // 静默处理，供应商加载失败不阻塞主流程
    }
  }, [])

  // 加载记账记录
  const loadRecords = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, unknown> = {}
      if (filterProviderId) params.provider_id = filterProviderId
      if (startDate) params.start_date = startDate
      if (endDate) params.end_date = endDate
      const res = await getAccountingRecords(params as any)
      setRecords(res.data || [])
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '加载失败',
        description: error.response?.data?.error || '无法加载记账记录',
      })
    } finally {
      setLoading(false)
    }
  }, [filterProviderId, startDate, endDate, toast])

  useEffect(() => {
    loadProviders()
  }, [loadProviders])

  useEffect(() => {
    loadRecords()
  }, [loadRecords])

  // 刷新列表（创建/编辑成功后回调）
  const handleRefresh = () => {
    loadRecords()
  }

  // 打开新增表单
  const handleCreate = () => {
    setEditRecord(null)
    setFormOpen(true)
  }

  // 打开编辑表单
  const handleEdit = (record: AccountingRecord) => {
    setEditRecord(record)
    setFormOpen(true)
  }

  // 打开删除确认
  const handleDeleteClick = (record: AccountingRecord) => {
    setDeletingRecord(record)
    setDeleteDialogOpen(true)
  }

  // 确认删除
  const handleDeleteConfirm = async () => {
    if (!deletingRecord) return
    setDeleteLoading(true)
    try {
      await deleteAccountingRecord(deletingRecord.id)
      toast({ title: '删除成功', description: '记账记录已删除' })
      setDeleteDialogOpen(false)
      setDeletingRecord(null)
      loadRecords()
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: '删除失败',
        description: error.response?.data?.error || '无法删除记账记录',
      })
    } finally {
      setDeleteLoading(false)
    }
  }

  // 格式化金额
  const formatAmount = (amount: number) => {
    return amount.toLocaleString('zh-CN', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
  }

  return (
    <div className="space-y-4">
      {/* 筛选栏 */}
      <Card className="border rounded-lg shadow-none">
        <CardContent className="pt-4 pb-4">
          <div className="flex flex-wrap gap-4 items-end">
            {/* 供应商筛选 */}
            <div className="flex-1 min-w-[160px]">
              <label className="text-sm font-medium mb-1.5 block">供应商</label>
              <select
                value={filterProviderId ?? ''}
                onChange={(e) =>
                  setFilterProviderId(
                    e.target.value ? Number(e.target.value) : undefined
                  )
                }
                className="w-full h-10 px-3 py-2 border rounded-md bg-background text-sm cursor-pointer"
              >
                <option value="">全部</option>
                {providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>

            {/* 开始日期 */}
            <div className="flex-1 min-w-[160px]">
              <label className="text-sm font-medium mb-1.5 block">开始日期</label>
              <input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="w-full h-10 px-3 py-2 border rounded-md bg-background text-sm cursor-pointer"
              />
            </div>

            {/* 结束日期 */}
            <div className="flex-1 min-w-[160px]">
              <label className="text-sm font-medium mb-1.5 block">结束日期</label>
              <input
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="w-full h-10 px-3 py-2 border rounded-md bg-background text-sm cursor-pointer"
              />
            </div>

            {/* 新增按钮 */}
            <Button onClick={handleCreate} className="cursor-pointer">
              <Plus className="mr-2 h-4 w-4" />
              新增
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 记录表格 */}
      <Card className="border rounded-lg shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-lg">记账记录</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <SkeletonTable />
          ) : records.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg font-medium">暂无记账记录</p>
              <p className="text-sm mt-2">点击"新增"按钮添加第一条记录</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="pb-3 pr-4 font-medium text-muted-foreground">供应商</th>
                    <th className="pb-3 pr-4 font-medium text-muted-foreground text-right">金额</th>
                    <th className="pb-3 pr-4 font-medium text-muted-foreground">日期</th>
                    <th className="pb-3 pr-4 font-medium text-muted-foreground">备注</th>
                    <th className="pb-3 font-medium text-muted-foreground text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {records.map((record) => (
                    <tr
                      key={record.id}
                      className="border-b last:border-b-0 hover:bg-accent/50 transition-colors duration-150"
                    >
                      <td className="py-3 pr-4 font-medium">{record.provider_name}</td>
                      <td className="py-3 pr-4 text-right tabular-nums">
                        ¥{formatAmount(record.amount)}
                      </td>
                      <td className="py-3 pr-4">{record.date}</td>
                      <td className="py-3 pr-4 text-muted-foreground max-w-[200px] truncate">
                        {record.note || '-'}
                      </td>
                      <td className="py-3 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 cursor-pointer"
                            onClick={() => handleEdit(record)}
                            title="编辑"
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 cursor-pointer text-destructive hover:text-destructive"
                            onClick={() => handleDeleteClick(record)}
                            title="删除"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 新增/编辑表单弹窗 */}
      <AccountingRecordForm
        open={formOpen}
        onOpenChange={setFormOpen}
        onSuccess={handleRefresh}
        providers={providers}
        editRecord={editRecord}
      />

      {/* 删除确认弹窗 */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除这条记账记录吗？此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          {deletingRecord && (
            <div className="text-sm space-y-1 py-2">
              <p>
                <span className="text-muted-foreground">供应商：</span>
                {deletingRecord.provider_name}
              </p>
              <p>
                <span className="text-muted-foreground">金额：</span>
                ¥{formatAmount(deletingRecord.amount)}
              </p>
              <p>
                <span className="text-muted-foreground">日期：</span>
                {deletingRecord.date}
              </p>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleteLoading}
              className="cursor-pointer"
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteConfirm}
              disabled={deleteLoading}
              className="cursor-pointer"
            >
              {deleteLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="mr-2 h-4 w-4" />
              )}
              {deleteLoading ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

/** Skeleton 加载占位 */
function SkeletonTable() {
  return (
    <div className="space-y-3">
      {/* 表头骨架 */}
      <div className="flex gap-4 pb-3 border-b">
        <div className="h-4 w-20 bg-muted animate-pulse rounded" />
        <div className="h-4 w-16 bg-muted animate-pulse rounded ml-auto" />
        <div className="h-4 w-20 bg-muted animate-pulse rounded" />
        <div className="h-4 w-24 bg-muted animate-pulse rounded" />
        <div className="h-4 w-16 bg-muted animate-pulse rounded" />
      </div>
      {/* 行骨架 */}
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 py-3 border-b last:border-b-0">
          <div className="h-4 w-24 bg-muted animate-pulse rounded" />
          <div className="h-4 w-16 bg-muted animate-pulse rounded ml-auto" />
          <div className="h-4 w-20 bg-muted animate-pulse rounded" />
          <div className="h-4 w-32 bg-muted animate-pulse rounded" />
          <div className="flex gap-1">
            <div className="h-8 w-8 bg-muted animate-pulse rounded" />
            <div className="h-8 w-8 bg-muted animate-pulse rounded" />
          </div>
        </div>
      ))}
    </div>
  )
}
