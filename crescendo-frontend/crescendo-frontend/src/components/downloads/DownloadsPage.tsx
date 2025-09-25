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

    // Cleanup all connections when component unmounts
    return () => {
      if (activeDownloads.length === 0) {
        disconnectAll()
      }
    }
  }, [downloads, connectToDownload, disconnectAll])

  // Separate effect to handle disconnecting from completed downloads
  useEffect(() => {
    const activeDownloads = downloads.filter(d => d.status === 'downloading' || d.status === 'queued')

    // Disconnect from completed/failed downloads
    Object.keys(progressData).forEach(jobId => {
      const isStillActive = activeDownloads.some(d => d.id === jobId)
      if (!isStillActive) {
        disconnectFromDownload(jobId)
      }
    })
  }, [downloads, progressData, disconnectFromDownload])

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
      <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
        <div>
          <h1 style={{
            fontSize: '24px',
            fontWeight: 'bold',
            color: '#f1f5f9',
            marginBottom: '8px'
          }}>Downloads</h1>
          <p style={{
            color: '#94a3b8',
            fontSize: '14px'
          }}>
            Manage your download queue and monitor progress
          </p>
        </div>
        <div style={{
          backgroundColor: '#1e293b',
          border: '1px solid #475569',
          borderRadius: '8px',
          padding: '24px'
        }}>
          <div style={{
            textAlign: 'center',
            padding: '32px 0'
          }}>
            <Loader2 style={{
              width: '32px',
              height: '32px',
              margin: '0 auto 16px',
              color: '#94a3b8'
            }} className="animate-spin" />
            <p style={{
              color: '#94a3b8',
              fontSize: '16px'
            }}>Loading download queue...</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <h1 style={{
            fontSize: '24px',
            fontWeight: 'bold',
            color: '#f1f5f9',
            marginBottom: '8px'
          }}>Downloads</h1>
          <p style={{
            color: '#94a3b8',
            fontSize: '14px'
          }}>
            Manage your download queue and monitor progress
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          {/* Connection Status Indicator */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {connectionStatus === 'connected' && (
              <>
                <Wifi style={{ width: '16px', height: '16px', color: '#10b981' }} />
                <span style={{ fontSize: '14px', color: '#10b981' }}>Live Updates</span>
              </>
            )}
            {connectionStatus === 'connecting' && (
              <>
                <Loader2 style={{ width: '16px', height: '16px', color: '#f59e0b' }} className="animate-spin" />
                <span style={{ fontSize: '14px', color: '#f59e0b' }}>Connecting...</span>
              </>
            )}
            {connectionStatus === 'error' && (
              <>
                <AlertCircle style={{ width: '16px', height: '16px', color: '#ef4444' }} />
                <span style={{ fontSize: '14px', color: '#ef4444' }}>Connection Error</span>
              </>
            )}
            {connectionStatus === 'disconnected' && (
              <>
                <WifiOff style={{ width: '16px', height: '16px', color: '#6b7280' }} />
                <span style={{ fontSize: '14px', color: '#6b7280' }}>
                  {activeDownloads.length > 0 ? 'Disconnected' : 'No Active Downloads'}
                </span>
              </>
            )}
          </div>

          <button
            onClick={() => fetchDownloads(true)}
            disabled={isRefreshing}
            style={{
              padding: '8px 16px',
              backgroundColor: 'transparent',
              border: '1px solid #475569',
              borderRadius: '6px',
              color: '#f1f5f9',
              fontSize: '14px',
              cursor: isRefreshing ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              opacity: isRefreshing ? 0.6 : 1
            }}
            onMouseEnter={(e) => {
              if (!isRefreshing) {
                e.currentTarget.style.backgroundColor = '#334155'
              }
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = 'transparent'
            }}
          >
            <RefreshCw style={{
              width: '16px',
              height: '16px',
              animation: isRefreshing ? 'spin 1s linear infinite' : 'none'
            }} />
            Refresh
          </button>
        </div>
      </div>

      {/* Active Downloads */}
      <div style={{
        backgroundColor: '#1e293b',
        border: '1px solid #475569',
        borderRadius: '8px'
      }}>
        <div style={{
          padding: '24px 24px 0 24px',
          borderBottom: '1px solid #475569'
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            paddingBottom: '16px'
          }}>
            <h2 style={{
              fontSize: '20px',
              fontWeight: 'bold',
              color: '#f1f5f9'
            }}>
              Active Downloads
            </h2>
            <span style={{
              padding: '4px 12px',
              backgroundColor: '#334155',
              border: '1px solid #475569',
              borderRadius: '6px',
              fontSize: '12px',
              color: '#e2e8f0',
              fontWeight: '500'
            }}>
              {activeDownloads.length} active
            </span>
          </div>
        </div>
        <div style={{ padding: '24px' }}>
          {activeDownloads.length === 0 ? (
            <div style={{
              textAlign: 'center',
              padding: '32px 0',
              color: '#94a3b8'
            }}>
              <Download style={{
                width: '48px',
                height: '48px',
                margin: '0 auto 16px',
                opacity: 0.5
              }} />
              <p style={{
                fontSize: '16px',
                marginBottom: '8px',
                color: '#f1f5f9'
              }}>No active downloads</p>
              <p style={{
                fontSize: '14px',
                color: '#94a3b8'
              }}>Start downloading music from the Search page</p>
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
        </div>
      </div>

      {/* Completed Downloads */}
      {completedDownloads.length > 0 && (
        <div style={{
          backgroundColor: '#1e293b',
          border: '1px solid #475569',
          borderRadius: '8px'
        }}>
          <div style={{
            padding: '24px 24px 0 24px',
            borderBottom: '1px solid #475569'
          }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              paddingBottom: '16px'
            }}>
              <h2 style={{
                fontSize: '20px',
                fontWeight: 'bold',
                color: '#f1f5f9'
              }}>
                Completed Downloads
              </h2>
              <span style={{
                padding: '4px 12px',
                backgroundColor: '#10b981',
                border: '1px solid #059669',
                borderRadius: '6px',
                fontSize: '12px',
                color: '#ffffff',
                fontWeight: '500'
              }}>
                {completedDownloads.length} completed
              </span>
            </div>
          </div>
          <div style={{ padding: '24px' }}>
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
          </div>
        </div>
      )}

      {/* Failed Downloads */}
      {failedDownloads.length > 0 && (
        <div style={{
          backgroundColor: '#1e293b',
          border: '1px solid #475569',
          borderRadius: '8px'
        }}>
          <div style={{
            padding: '24px 24px 0 24px',
            borderBottom: '1px solid #475569'
          }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              paddingBottom: '16px'
            }}>
              <h2 style={{
                fontSize: '20px',
                fontWeight: 'bold',
                color: '#f1f5f9'
              }}>
                Failed Downloads
              </h2>
              <span style={{
                padding: '4px 12px',
                backgroundColor: '#dc2626',
                border: '1px solid #b91c1c',
                borderRadius: '6px',
                fontSize: '12px',
                color: '#ffffff',
                fontWeight: '500'
              }}>
                {failedDownloads.length} failed
              </span>
            </div>
          </div>
          <div style={{ padding: '24px' }}>
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
          </div>
        </div>
      )}

      {/* Cancel Confirmation Dialog */}
      <Dialog open={cancelDialog.isOpen} onOpenChange={(open) =>
        setCancelDialog({ isOpen: open, jobId: '', title: '' })
      }>
        <DialogContent style={{
          backgroundColor: '#1e293b',
          border: '1px solid #475569',
          color: '#f1f5f9'
        }}>
          <DialogHeader>
            <DialogTitle style={{ color: '#f1f5f9' }}>Cancel Download</DialogTitle>
            <DialogDescription style={{ color: '#94a3b8' }}>
              Are you sure you want to cancel the download of "{cancelDialog.title}"?
              This action cannot be undone and any partial progress will be lost.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCancelDialog({ isOpen: false, jobId: '', title: '' })}
              style={{
                backgroundColor: 'transparent',
                border: '1px solid #475569',
                color: '#f1f5f9'
              }}
            >
              Keep Download
            </Button>
            <Button
              variant="destructive"
              onClick={() => handleCancelDownload(cancelDialog.jobId)}
              style={{
                backgroundColor: '#dc2626',
                border: '1px solid #b91c1c',
                color: '#ffffff'
              }}
            >
              Cancel Download
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}