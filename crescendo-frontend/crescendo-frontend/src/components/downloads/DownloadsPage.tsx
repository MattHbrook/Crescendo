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
import { Download, X, RefreshCw, Music, Disc, User, Clock, CheckCircle, XCircle, Loader2, Trash2, Wifi, WifiOff, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { apiService } from '@/services/api'
import { useDownloadProgress } from '@/hooks/useDownloadProgress'
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

  const { progressData, connectionStatus, connectToDownload, disconnectFromDownload, disconnectAll } = useDownloadProgress()

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
  }, [])

  // Connect to WebSockets for active downloads
  useEffect(() => {
    const activeDownloads = downloads.filter(d => d.status === 'downloading' || d.status === 'queued')

    // Connect to new active downloads
    activeDownloads.forEach(download => {
      if (!progressData[download.id]) {
        connectToDownload(download.id)
      }
    })

    // Disconnect from completed/failed downloads
    Object.keys(progressData).forEach(jobId => {
      const isStillActive = activeDownloads.some(d => d.id === jobId)
      if (!isStillActive) {
        disconnectFromDownload(jobId)
      }
    })

    // Cleanup all connections when component unmounts
    return () => {
      if (activeDownloads.length === 0) {
        disconnectAll()
      }
    }
  }, [downloads, progressData, connectToDownload, disconnectFromDownload, disconnectAll])

  // Handle download completion notifications
  useEffect(() => {
    Object.entries(progressData).forEach(([jobId, progress]) => {
      const download = downloads.find(d => d.id === jobId)
      if (download && progress.status === 'completed' && progress.progress === 100) {
        // Show completion toast notification
        toast.success(`Download completed: "${download.title}"`, {
          description: `${download.artist} - ${download.type}`,
          duration: 5000,
        })

        // Disconnect from this download's WebSocket
        disconnectFromDownload(jobId)

        // Refresh downloads to get updated status
        setTimeout(() => fetchDownloads(), 1000)
      } else if (download && progress.status === 'failed') {
        // Show error toast notification
        toast.error(`Download failed: "${download.title}"`, {
          description: progress.error || 'Unknown error occurred',
          duration: 8000,
        })

        // Disconnect from this download's WebSocket
        disconnectFromDownload(jobId)
      }
    })
  }, [progressData, downloads, disconnectFromDownload, fetchDownloads])

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

  // Helper function to get real-time progress data for a download
  const getProgressInfo = (download: DownloadJob) => {
    const realTimeData = progressData[download.id]
    if (realTimeData) {
      return {
        progress: realTimeData.progress,
        currentFile: realTimeData.currentFile,
        speed: realTimeData.speed,
        eta: realTimeData.eta,
        status: realTimeData.status
      }
    }
    // Fallback to download data from API
    return {
      progress: download.progress,
      currentFile: download.currentFile,
      speed: undefined,
      eta: undefined,
      status: download.status
    }
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
        <div className="flex items-center gap-4">
          {/* Connection Status Indicator */}
          <div className="flex items-center gap-2">
            {connectionStatus === 'connected' && (
              <>
                <Wifi className="w-4 h-4 text-green-500" />
                <span className="text-sm text-green-600">Live Updates</span>
              </>
            )}
            {connectionStatus === 'connecting' && (
              <>
                <Loader2 className="w-4 h-4 animate-spin text-yellow-500" />
                <span className="text-sm text-yellow-600">Connecting...</span>
              </>
            )}
            {connectionStatus === 'error' && (
              <>
                <AlertCircle className="w-4 h-4 text-red-500" />
                <span className="text-sm text-red-600">Connection Error</span>
              </>
            )}
            {connectionStatus === 'disconnected' && (
              <>
                <WifiOff className="w-4 h-4 text-gray-500" />
                <span className="text-sm text-gray-600">
                  {activeDownloads.length > 0 ? 'Disconnected' : 'No Active Downloads'}
                </span>
              </>
            )}
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
                {activeDownloads.map((download) => {
                  const progressInfo = getProgressInfo(download)
                  return (
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
                      <TableCell className="w-64">
                        <div className="space-y-1">
                          <Progress value={progressInfo.progress} className="w-full" />
                          <div className="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
                            <div className="flex justify-between">
                              <span>{progressInfo.progress}%</span>
                              {progressInfo.speed && (
                                <span className="text-blue-600">{progressInfo.speed}</span>
                              )}
                            </div>
                            <div className="flex justify-between">
                              {progressInfo.eta && (
                                <span className="text-orange-600">ETA: {progressInfo.eta}</span>
                              )}
                            </div>
                          </div>
                          {progressInfo.currentFile && (
                            <div className="text-xs text-muted-foreground">
                              <span className="truncate block max-w-60" title={progressInfo.currentFile}>
                                📁 {progressInfo.currentFile}
                              </span>
                            </div>
                          )}
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
                  )
                })}
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