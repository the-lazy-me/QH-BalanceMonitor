'use client'

import { useEffect, useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createAccountingRecord, updateAccountingRecord } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'
import { Save, Loader2 } from 'lucide-react'

interface AccountingRecordFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
  providers: Array<{ id: number; name: string }>
  editRecord?: {
    id: number
    provider_id: number
    amount: number
    date: string
    note: string
  } | null
}

interface FormErrors {
  provider_id?: string
  amount?: string
  date?: string
}

export function AccountingRecordForm({
  open,
  onOpenChange,
  onSuccess,
  providers,
  editRecord,
}: AccountingRecordFormProps) {
  const [providerId, setProviderId] = useState<number>(0)
  const [amount, setAmount] = useState('')
  const [date, setDate] = useState('')
  const [note, setNote] = useState('')
  const [errors, setErrors] = useState<FormErrors>({})
  const [loading, setLoading] = useState(false)
  const { toast } = useToast()

  const isEdit = !!editRecord

  // 打开弹窗时初始化表单
  useEffect(() => {
    if (open) {
      if (editRecord) {
        setProviderId(editRecord.provider_id)
        setAmount(String(editRecord.amount))
        setDate(editRecord.date)
        setNote(editRecord.note || '')
      } else {
        setProviderId(0)
        setAmount('')
        setDate('')
        setNote('')
      }
      setErrors({})
    }
  }, [open, editRecord])

  const validate = (): boolean => {
    const newErrors: FormErrors = {}

    if (!providerId) {
      newErrors.provider_id = '请选择供应商'
    }

    const amountNum = parseFloat(amount)
    if (!amount || isNaN(amountNum)) {
      newErrors.amount = '请输入金额'
    } else if (amountNum <= 0) {
      newErrors.amount = '金额必须为正数'
    }

    if (!date) {
      newErrors.date = '请选择日期'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validate()) return

    setLoading(true)
    try {
      if (isEdit && editRecord) {
        await updateAccountingRecord(editRecord.id, {
          amount: parseFloat(amount),
          date,
          note,
        })
        toast({ title: '更新成功', description: '记账记录已更新' })
      } else {
        await createAccountingRecord({
          provider_id: providerId,
          amount: parseFloat(amount),
          date,
          note: note || undefined,
        })
        toast({ title: '创建成功', description: '记账记录已创建' })
      }
      onOpenChange(false)
      onSuccess()
    } catch (error: any) {
      const message =
        error.response?.data?.error || error.message || '操作失败'
      toast({
        variant: 'destructive',
        title: isEdit ? '更新失败' : '创建失败',
        description: message,
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? '编辑记账记录' : '新增记账记录'}</DialogTitle>
          <DialogDescription>
            {isEdit ? '修改记账记录信息' : '为供应商添加一条进货记账记录'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* 供应商选择 */}
          <div className="space-y-2">
            <Label htmlFor="provider_id">
              供应商 <span className="text-destructive">*</span>
            </Label>
            <select
              id="provider_id"
              value={providerId}
              onChange={(e) => setProviderId(Number(e.target.value))}
              disabled={isEdit}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
            >
              <option value={0}>请选择供应商</option>
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
            {errors.provider_id && (
              <p className="text-sm text-destructive">{errors.provider_id}</p>
            )}
          </div>

          {/* 金额输入 */}
          <div className="space-y-2">
            <Label htmlFor="amount">
              金额 <span className="text-destructive">*</span>
            </Label>
            <Input
              id="amount"
              type="number"
              step="0.01"
              min="0.01"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="请输入金额"
            />
            {errors.amount && (
              <p className="text-sm text-destructive">{errors.amount}</p>
            )}
          </div>

          {/* 日期选择 */}
          <div className="space-y-2">
            <Label htmlFor="date">
              日期 <span className="text-destructive">*</span>
            </Label>
            <Input
              id="date"
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
            {errors.date && (
              <p className="text-sm text-destructive">{errors.date}</p>
            )}
          </div>

          {/* 备注输入 */}
          <div className="space-y-2">
            <Label htmlFor="note">备注</Label>
            <textarea
              id="note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="可选备注信息"
              rows={3}
              className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-none"
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              取消
            </Button>
            <Button type="submit" disabled={loading} className="cursor-pointer">
              {loading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Save className="mr-2 h-4 w-4" />
              )}
              {loading ? '提交中...' : isEdit ? '保存修改' : '创建记录'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
