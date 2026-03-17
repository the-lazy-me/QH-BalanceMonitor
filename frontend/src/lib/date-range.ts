export interface DateRangeValue {
  startDate: string
  endDate: string
}

const SHANGHAI_TIME_ZONE = 'Asia/Shanghai'

const shanghaiDateFormatter = new Intl.DateTimeFormat('zh-CN', {
  timeZone: SHANGHAI_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})

function formatDateFromParts(year: number, month: number, day: number) {
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

export function getTodayDateString() {
  const parts = shanghaiDateFormatter.formatToParts(new Date())
  const year = Number(parts.find(part => part.type === 'year')?.value || '0')
  const month = Number(parts.find(part => part.type === 'month')?.value || '0')
  const day = Number(parts.find(part => part.type === 'day')?.value || '0')

  return formatDateFromParts(year, month, day)
}

export function shiftDate(dateString: string, daysDelta: number) {
  const [year, month, day] = dateString.split('-').map(Number)
  const date = new Date(Date.UTC(year, month - 1, day))
  date.setUTCDate(date.getUTCDate() + daysDelta)

  return formatDateFromParts(
    date.getUTCFullYear(),
    date.getUTCMonth() + 1,
    date.getUTCDate()
  )
}

export function getPresetDateRange(days: number): DateRangeValue {
  const normalizedDays = days > 0 ? days : 1
  const endDate = getTodayDateString()

  return {
    startDate: shiftDate(endDate, -(normalizedDays - 1)),
    endDate,
  }
}

export function getDateRangeError(value: Partial<DateRangeValue>) {
  if (!value.startDate || !value.endDate) {
    return '请选择开始和结束日期'
  }

  if (value.startDate > value.endDate) {
    return '结束日期不能早于开始日期'
  }

  return null
}

export function formatDateRangeLabel(range: DateRangeValue) {
  return `${range.startDate} 至 ${range.endDate}`
}

export function formatChartDate(dateString: string, includeYear = false) {
  const [year, month, day] = dateString.split('-').map(Number)

  if (includeYear) {
    return `${year}/${month}/${day}`
  }

  return `${month}/${day}`
}

export function formatLongDate(dateString: string) {
  const [year, month, day] = dateString.split('-').map(Number)
  return `${year}年${month}月${day}日`
}
