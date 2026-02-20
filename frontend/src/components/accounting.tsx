'use client'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { List, BarChart3 } from 'lucide-react'
import { AccountingRecordList } from './accounting-record-list'
import { AccountingCostStats } from './accounting-cost-stats'

export function Accounting() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold">记账管理</h2>
        <p className="text-muted-foreground">管理供应商进货账目与成本统计</p>
      </div>

      <Tabs defaultValue="records" className="space-y-6">
        <TabsList>
          <TabsTrigger value="records" className="flex items-center gap-2 cursor-pointer">
            <List className="w-4 h-4" />
            记录列表
          </TabsTrigger>
          <TabsTrigger value="stats" className="flex items-center gap-2 cursor-pointer">
            <BarChart3 className="w-4 h-4" />
            成本统计
          </TabsTrigger>
        </TabsList>

        <TabsContent value="records">
          <AccountingRecordList />
        </TabsContent>

        <TabsContent value="stats">
          <AccountingCostStats />
        </TabsContent>
      </Tabs>
    </div>
  )
}
