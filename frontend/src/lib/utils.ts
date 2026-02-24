import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatBalance(balance: number): string {
  return balance.toFixed(2)
}


export function calculateRMB(balance: number, priceRatio: number | null | undefined): number {
  const ratio = priceRatio && priceRatio > 0 ? priceRatio : 1
  return balance / ratio
}

export function formatRMB(amount: number): string {
  return `¥${amount.toFixed(2)}`
}
