'use client'

import { useEffect, useMemo, useState } from 'react'
import { CalendarRange, RotateCcw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  DateRangeValue,
  formatDateRangeLabel,
  getDateRangeError,
  getPresetDateRange,
  getTodayDateString,
} from '@/lib/date-range'

interface DateRangePreset {
  label: string
  days: number
}

interface DateRangeSelectorProps {
  value: DateRangeValue
  onChange: (value: DateRangeValue) => void
  presets?: DateRangePreset[]
  className?: string
  disabled?: boolean
}

const DEFAULT_PRESETS: DateRangePreset[] = [
  { label: '最近 7 天', days: 7 },
  { label: '最近 15 天', days: 15 },
  { label: '最近 30 天', days: 30 },
  { label: '最近 90 天', days: 90 },
]

export function DateRangeSelector({
  value,
  onChange,
  presets = DEFAULT_PRESETS,
  className,
  disabled = false,
}: DateRangeSelectorProps) {
  const [draft, setDraft] = useState<DateRangeValue>(value)
  const maxDate = useMemo(() => getTodayDateString(), [])

  useEffect(() => {
    setDraft(value)
  }, [value.endDate, value.startDate])

  const validationError = getDateRangeError(draft)

  const activePresetDays = useMemo(() => {
    const matchedPreset = presets.find((preset) => {
      const presetRange = getPresetDateRange(preset.days)
      return (
        presetRange.startDate === value.startDate &&
        presetRange.endDate === value.endDate
      )
    })

    return matchedPreset?.days ?? null
  }, [presets, value.endDate, value.startDate])

  const handleDraftChange = (field: keyof DateRangeValue, nextValue: string) => {
    setDraft((prev) => ({
      ...prev,
      [field]: nextValue,
    }))
  }

  const handleApply = () => {
    if (validationError) return
    if (
      draft.startDate === value.startDate &&
      draft.endDate === value.endDate
    ) {
      return
    }

    onChange(draft)
  }

  const handlePresetClick = (days: number) => {
    const presetRange = getPresetDateRange(days)
    setDraft(presetRange)
    onChange(presetRange)
  }

  const handleReset = () => {
    const defaultRange = getPresetDateRange(presets[0]?.days ?? 7)
    setDraft(defaultRange)
    onChange(defaultRange)
  }

  const applyDisabled =
    disabled ||
    Boolean(validationError) ||
    (draft.startDate === value.startDate && draft.endDate === value.endDate)

  return (
    <div className={cn('rounded-xl border bg-muted/30 p-3', className)}>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div className="flex items-center gap-2 text-sm font-medium">
            <CalendarRange className="h-4 w-4 text-muted-foreground" />
            <span>日期范围</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {presets.map((preset) => (
              <Button
                key={preset.days}
                type="button"
                size="sm"
                variant={activePresetDays === preset.days ? 'default' : 'outline'}
                onClick={() => handlePresetClick(preset.days)}
                disabled={disabled}
              >
                {preset.label}
              </Button>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-3 xl:flex-row xl:items-end">
          <div className="grid flex-1 gap-3 sm:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] sm:items-center">
            <div className="space-y-1">
              <div className="text-xs text-muted-foreground">开始日期</div>
              <Input
                type="date"
                value={draft.startDate}
                max={draft.endDate || maxDate}
                onChange={(event) => handleDraftChange('startDate', event.target.value)}
                className="h-9"
                disabled={disabled}
              />
            </div>

            <div className="hidden text-center text-sm text-muted-foreground sm:block">
              至
            </div>

            <div className="space-y-1">
              <div className="text-xs text-muted-foreground">结束日期</div>
              <Input
                type="date"
                value={draft.endDate}
                min={draft.startDate || undefined}
                max={maxDate}
                onChange={(event) => handleDraftChange('endDate', event.target.value)}
                className="h-9"
                disabled={disabled}
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              type="button"
              size="sm"
              onClick={handleApply}
              disabled={applyDisabled}
            >
              应用
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={handleReset}
              disabled={disabled}
            >
              <RotateCcw className="mr-2 h-4 w-4" />
              重置
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          <span className="text-muted-foreground">
            当前 {formatDateRangeLabel(value)}
          </span>
          {validationError && (
            <span className="text-destructive">{validationError}</span>
          )}
        </div>
      </div>
    </div>
  )
}
