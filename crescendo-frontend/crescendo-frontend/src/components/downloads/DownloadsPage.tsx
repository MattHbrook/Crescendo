import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Download } from 'lucide-react'

export function DownloadsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Downloads</h1>
        <p className="text-muted-foreground">
          Manage your download queue and monitor progress
        </p>
      </div>

      <div className="grid gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Download Queue
              <Badge variant="secondary">0 active</Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-center text-muted-foreground py-8">
              <Download className="mx-auto h-12 w-12 mb-4 opacity-50" />
              <p>No downloads in queue</p>
              <p className="text-sm">Start downloading music from the Search page</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}