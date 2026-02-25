'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { parseCurl, testCustomQuery } from '@/lib/api'
import { Plus, Trash2, Play, Loader2 } from 'lucide-react'

export interface CustomQueryData {
  method: string
  url: string
  headers: Record<string, string>
  balance_field_path: string
}

interface CustomQueryFormProps {
  providerId?: number
  visible: boolean
  onChange: (data: CustomQueryData) => void
  initialData?: CustomQueryData | null
}

interface TestResult {
  success: boolean
  balance?: number
  rawResponse?: unknown
  error?: string
}

interface ValidationErrors {
  url?: string
  balanceFieldPath?: string
}

export function CustomQueryForm({ providerId, visible, onChange, initialData }: CustomQueryFormProps) {
  const [curlInput, setCurlInput] = useState('')
  const [method, setMethod] = useState('GET')
  const [url, setUrl] = useState('')
  const [headers, setHeaders] = useState<Array<{ key: string; value: string }>>([])
  const [balanceFieldPath, setBalanceFieldPath] = useState('')
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<TestResult | null>(null)
  const [parseError, setParseError] = useState('')
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({})
  const lastCurlRef = useRef('')
  const isPopulatingRef = useRef(false)
  const appliedInitialRef = useRef<string | null>(null)

  // Build current form data for onChange callback
  const buildFormData = useCallback((): CustomQueryData => {
    const headersMap: Record<string, string> = {}
    headers.forEach(({ key, value }) => {
      if (key.trim()) headersMap[key.trim()] = value
    })
    return { method, url, headers: headersMap, balance_field_path: balanceFieldPath }
  }, [method, url, headers, balanceFieldPath])

  // Stable serialization of headers to avoid infinite re-render loops
  const headersKey = JSON.stringify(headers)

  // Notify parent on any field change (skip during initialData population)
  useEffect(() => {
    if (isPopulatingRef.current) return
    onChange(buildFormData())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [method, url, headersKey, balanceFieldPath])

  // Populate form from initialData (only once per unique data)
  useEffect(() => {
    if (!initialData) return
    const key = JSON.stringify(initialData)
    if (key === appliedInitialRef.current) return
    appliedInitialRef.current = key

    isPopulatingRef.current = true
    setMethod(initialData.method || 'GET')
    setUrl(initialData.url || '')
    setBalanceFieldPath(initialData.balance_field_path || '')
    const headerEntries = Object.entries(initialData.headers || {})
    setHeaders(headerEntries.length > 0
      ? headerEntries.map(([key, value]) => ({ key, value }))
      : []
    )
    // Allow onChange to fire again after this render cycle
    requestAnimationFrame(() => { isPopulatingRef.current = false })
  }, [initialData])

  // Parse curl command
  const handleParseCurl = async (value: string) => {
    const trimmed = value.trim()
    if (!trimmed || trimmed === lastCurlRef.current) return
    lastCurlRef.current = trimmed

    try {
      const response = await parseCurl(trimmed)
      const data = response.data
      setParseError('')
      setMethod(data.method || 'GET')
      setUrl(data.url || '')
      if (data.headers && Object.keys(data.headers).length > 0) {
        setHeaders(Object.entries(data.headers).map(([key, value]) => ({ key, value: value as string })))
      }
      // Clear URL validation error since we just populated it
      if (data.url) {
        setValidationErrors(prev => ({ ...prev, url: undefined }))
      }
    } catch (error: any) {
      const msg = error.response?.data?.error || '解析 curl 命令失败'
      setParseError(msg)
    }
  }

  // Test custom query
  const handleTest = async () => {
    // Validate before testing
    const errors: ValidationErrors = {}
    if (!url.trim()) errors.url = '请求 URL 不能为空'
    if (!balanceFieldPath.trim()) errors.balanceFieldPath = '余额字段路径不能为空'
    setValidationErrors(errors)
    if (Object.keys(errors).length > 0) return

    setTesting(true)
    setTestResult(null)
    try {
      const response = await testCustomQuery(buildFormData())
      const data = response.data
      setTestResult({
        success: data.success,
        balance: data.extracted_balance,
        rawResponse: data.raw_response,
        error: data.error,
      })
    } catch (error: any) {
      setTestResult({
        success: false,
        error: error.response?.data?.error || error.message || '测试请求失败',
      })
    } finally {
      setTesting(false)
    }
  }

  // Header management
  const addHeader = () => setHeaders([...headers, { key: '', value: '' }])
  const removeHeader = (index: number) => setHeaders(headers.filter((_, i) => i !== index))
  const updateHeader = (index: number, field: 'key' | 'value', val: string) => {
    const updated = [...headers]
    updated[index] = { ...updated[index], [field]: val }
    setHeaders(updated)
  }

  // Validation on blur
  const validateUrl = () => {
    if (!url.trim()) {
      setValidationErrors(prev => ({ ...prev, url: '请求 URL 不能为空' }))
    }
  }
  const validateBalancePath = () => {
    if (!balanceFieldPath.trim()) {
      setValidationErrors(prev => ({ ...prev, balanceFieldPath: '余额字段路径不能为空' }))
    }
  }

  if (!visible) return null

  return (
    <div className="space-y-4">
      {/* Curl paste area */}
      <div className="grid gap-2">
        <Label htmlFor="curl-input">Curl 命令 (可选)</Label>
        <textarea
          id="curl-input"
          className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 font-mono"
          placeholder="粘贴 curl 命令，自动解析填充下方字段..."
          value={curlInput}
          onChange={(e) => {
            setCurlInput(e.target.value)
            if (parseError) setParseError('')
          }}
          onBlur={() => handleParseCurl(curlInput)}
          onPaste={(e) => {
            const pasted = e.clipboardData.getData('text')
            // Let the onChange fire first, then parse
            setTimeout(() => handleParseCurl(pasted), 0)
          }}
        />
        {parseError && (
          <p className="text-sm text-destructive">{parseError}</p>
        )}
      </div>

      {/* HTTP Method */}
      <div className="grid gap-2">
        <Label htmlFor="http-method">HTTP 方法</Label>
        <select
          id="http-method"
          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          value={method}
          onChange={(e) => setMethod(e.target.value)}
        >
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
          <option value="PATCH">PATCH</option>
        </select>
      </div>

      {/* URL */}
      <div className="grid gap-2">
        <Label htmlFor="request-url">请求 URL</Label>
        <Input
          id="request-url"
          value={url}
          onChange={(e) => {
            setUrl(e.target.value)
            if (validationErrors.url) setValidationErrors(prev => ({ ...prev, url: undefined }))
          }}
          onBlur={validateUrl}
          placeholder="https://api.example.com/balance"
        />
        {validationErrors.url && (
          <p className="text-sm text-destructive">{validationErrors.url}</p>
        )}
      </div>

      {/* Headers */}
      <div className="grid gap-2">
        <Label>请求头</Label>
        <div className="space-y-2">
          {headers.map((header, index) => (
            <div key={index} className="flex gap-2 items-center">
              <Input
                value={header.key}
                onChange={(e) => updateHeader(index, 'key', e.target.value)}
                placeholder="Header 名称"
                className="flex-1"
              />
              <Input
                value={header.value}
                onChange={(e) => updateHeader(index, 'value', e.target.value)}
                placeholder="Header 值"
                className="flex-1"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => removeHeader(index)}
                className="shrink-0"
              >
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>
          ))}
        </div>
        <Button type="button" variant="outline" size="sm" onClick={addHeader} className="w-fit">
          <Plus className="mr-2 h-4 w-4" />
          添加请求头
        </Button>
      </div>

      {/* Balance Field Path */}
      <div className="grid gap-2">
        <Label htmlFor="balance-field-path">余额字段路径</Label>
        <Input
          id="balance-field-path"
          value={balanceFieldPath}
          onChange={(e) => {
            setBalanceFieldPath(e.target.value)
            if (validationErrors.balanceFieldPath) setValidationErrors(prev => ({ ...prev, balanceFieldPath: undefined }))
          }}
          onBlur={validateBalancePath}
          placeholder="data.balance"
        />
        {validationErrors.balanceFieldPath && (
          <p className="text-sm text-destructive">{validationErrors.balanceFieldPath}</p>
        )}
      </div>

      {/* Test Button */}
      <Button type="button" variant="outline" onClick={handleTest} disabled={testing}>
        {testing ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Play className="mr-2 h-4 w-4" />
        )}
        测试
      </Button>

      {/* Test Result */}
      {testResult && (
        <div className={`rounded-md border p-4 space-y-2 ${
          testResult.success ? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950' : 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950'
        }`}>
          <p className="text-sm font-medium">测试结果</p>
          {testResult.success && testResult.balance !== undefined && (
            <p className="text-sm text-green-700 dark:text-green-300">
              提取余额: <span className="font-mono font-bold">{testResult.balance}</span>
            </p>
          )}
          {testResult.error && (
            <p className="text-sm text-destructive">{testResult.error}</p>
          )}
          {testResult.rawResponse != null && (
            <div className="mt-2">
              <p className="text-xs text-muted-foreground mb-1">JSON 响应:</p>
              <pre className="text-xs font-mono bg-muted p-2 rounded overflow-auto max-h-48">
                {JSON.stringify(testResult.rawResponse, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
