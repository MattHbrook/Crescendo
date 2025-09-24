import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Progress } from '@/components/ui/progress'
import { Download, X, RefreshCw, Music, Disc, User, Clock, CheckCircle, XCircle, Loader2, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiService } from '@/services/api'
import type { DownloadJob } from '@/types/api'

export function DownloadsPage() {
  const [downloads, setDownloads] = useState<DownloadJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [cancelDialog, setCancelDialog] = useState<{ isOpen: boolean; jobId: string; title: string }>({
    isOpen: false,
    jobId: '',
    title: ''
  })
  const [isRefreshing, setIsRefreshing] = useState(false)

  const fetchDownloads = async (showRefreshToast = false) => {
    try {
      if (showRefreshToast) setIsRefreshing(true)
      const downloadData = await apiService.getDownloads()
      setDownloads(downloadData)
      if (showRefreshToast) {
        toast.success('Download queue refreshed')
      }
    } catch (error) {
      console.error('Failed to fetch downloads:', error)
      toast.error('Failed to load download queue')
    } finally {
      setIsLoading(false)
      if (showRefreshToast) setIsRefreshing(false)
    }
  }

  useEffect(() => {
    fetchDownloads()

    // Auto-refresh every 5 seconds for active downloads
    const interval = setInterval(() => {
      if (downloads.some(d => d.status === 'downloading' || d.status === 'queued')) {
        fetchDownloads()
      }
    }, 5000)

    return () => clearInterval(interval)
  }, [downloads])

  const handleCancelDownload = async (jobId: string) => {
    try {
      await apiService.cancelDownload(jobId)
      toast.success('Download cancelled successfully')
      fetchDownloads()
      setCancelDialog({ isOpen: false, jobId: '', title: '' })
    } catch (error) {
      console.error('Failed to cancel download:', error)
      toast.error('Failed to cancel download')
    }
  }

  const handleRemoveDownload = (jobId: string, title: string) => {
    // For completed/failed downloads, remove from local state
    setDownloads(prev => prev.filter(d => d.id !== jobId))
    toast.success(`Removed "${title}" from queue`)
  }

  const getStatusBadge = (status: DownloadJob['status']) => {
    switch (status) {
      case 'queued':
        return <Badge variant="secondary"><Clock className="w-3 h-3 mr-1" />Queued</Badge>
      case 'downloading':
        return <Badge variant="default"><Loader2 className="w-3 h-3 mr-1 animate-spin" />Downloading</Badge>
      case 'completed':
        return <Badge variant="default" className="bg-green-500 hover:bg-green-600"><CheckCircle className="w-3 h-3 mr-1" />Completed</Badge>
      case 'failed':
        return <Badge variant="destructive"><XCircle className="w-3 h-3 mr-1" />Failed</Badge>
      case 'cancelled':
        return <Badge variant="outline"><X className="w-3 h-3 mr-1" />Cancelled</Badge>
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  const getTypeIcon = (type: DownloadJob['type']) => {
    switch (type) {
      case 'track':
        return <Music className="w-4 h-4" />
      case 'album':
        return <Disc className="w-4 h-4" />
      case 'artist':
        return <User className="w-4 h-4" />
      default:
        return <Music className="w-4 h-4" />
    }
  }

  const activeDownloads = downloads.filter(d => d.status === 'downloading' || d.status === 'queued')
  const completedDownloads = downloads.filter(d => d.status === 'completed')
  const failedDownloads = downloads.filter(d => d.status === 'failed' || d.status === 'cancelled')

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Downloads</h1>
          <p className="text-muted-foreground">
            Manage your download queue and monitor progress
          </p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-8">
              <Loader2 className="mx-auto h-8 w-8 animate-spin text-muted-foreground mb-4" />
              <p className="text-muted-foreground">Loading download queue...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Downloads</h1>
          <p className="text-muted-foreground">
            Manage your download queue and monitor progress
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => fetchDownloads(true)}
          disabled={isRefreshing}
        >
          <RefreshCw className={`w-4 h-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      {/* Active Downloads */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            Active Downloads
            <Badge variant="secondary">{activeDownloads.length} active</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {activeDownloads.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              <Download className="mx-auto h-12 w-12 mb-4 opacity-50" />
              <p>No active downloads</p>
              <p className="text-sm">Start downloading music from the Search page</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12"></TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Artist</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Progress</TableHead>
                  <TableHead className="w-24">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {activeDownloads.map((download) => (
                  <TableRow key={download.id}>
                    <TableCell>
                      {getTypeIcon(download.type)}
                    </TableCell>
                    <TableCell className="font-medium">{download.title}</TableCell>
                    <TableCell>{download.artist}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">
                        {download.type}
                      </Badge>
                    </TableCell>
                    <TableCell>{getStatusBadge(download.status)}</TableCell>
                    <TableCell className="w-48">
                      <div className="space-y-1">
                        <Progress value={download.progress} className="w-full" />
                        <div className="flex justify-between text-xs text-muted-foreground">
                          <span>{download.progress}%</span>
                          {download.currentFile && (
                            <span className="truncate max-w-32" title={download.currentFile}>
                              {download.currentFile}
                            </span>
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCancelDialog({
                          isOpen: true,
                          jobId: download.id,
                          title: download.title
                        })}
                      >
                        <X className="w-3 h-3" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Completed Downloads */}
      {completedDownloads.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Completed Downloads
              <Badge variant="secondary" className="bg-green-500 hover:bg-green-600">
                {completedDownloads.length} completed
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12"></TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Artist</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Completed</TableHead>
                  <TableHead className="w-24">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {completedDownloads.map((download) => (
                  <TableRow key={download.id}>
                    <TableCell>
                      {getTypeIcon(download.type)}
                    </TableCell>
                    <TableCell className="font-medium">{download.title}</TableCell>
                    <TableCell>{download.artist}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">
                        {download.type}
                      </Badge>
                    </TableCell>
                    <TableCell>{getStatusBadge(download.status)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(download.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRemoveDownload(download.id, download.title)}
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Failed Downloads */}
      {failedDownloads.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Failed Downloads
              <Badge variant="destructive">{failedDownloads.length} failed</Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12"></TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Artist</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Error</TableHead>
                  <TableHead className="w-24">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {failedDownloads.map((download) => (
                  <TableRow key={download.id}>
                    <TableCell>
                      {getTypeIcon(download.type)}
                    </TableCell>
                    <TableCell className="font-medium">{download.title}</TableCell>
                    <TableCell>{download.artist}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">
                        {download.type}
                      </Badge>
                    </TableCell>
                    <TableCell>{getStatusBadge(download.status)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground max-w-48">
                      <span className="truncate block" title={download.error}>
                        {download.error || 'Unknown error'}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRemoveDownload(download.id, download.title)}
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Cancel Confirmation Dialog */}
      <Dialog open={cancelDialog.isOpen} onOpenChange={(open) =>
        setCancelDialog({ isOpen: open, jobId: '', title: '' })
      }>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Cancel Download</DialogTitle>
            <DialogDescription>
              Are you sure you want to cancel the download of "{cancelDialog.title}"?
              This action cannot be undone and any partial progress will be lost.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCancelDialog({ isOpen: false, jobId: '', title: '' })}
            >
              Keep Download
            </Button>
            <Button
              variant="destructive"
              onClick={() => handleCancelDownload(cancelDialog.jobId)}
            >
              Cancel Download
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}