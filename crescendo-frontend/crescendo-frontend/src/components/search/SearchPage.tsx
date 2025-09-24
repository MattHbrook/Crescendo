import { useState } from 'react'
import { Search, Download, Music, User, Disc, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { apiService } from '@/services/api'
import type { SearchResult } from '@/types/api'

export function SearchPage() {
  const [query, setQuery] = useState('')
  const [searchType, setSearchType] = useState<'track' | 'album'>('track')
  const [results, setResults] = useState<SearchResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)

  const handleSearch = async () => {
    if (!query.trim()) {
      toast.error('Please enter a search term')
      return
    }

    setIsLoading(true)
    try {
      const searchResults = await apiService.search(
        query.trim(),
        searchType
      )
      setResults(searchResults)
      setHasSearched(true)

      if (searchResults.length === 0) {
        toast.info('No results found for your search')
      } else {
        toast.success(`Found ${searchResults.length} result${searchResults.length === 1 ? '' : 's'}`)
      }
    } catch (error) {
      console.error('Search failed:', error)
      toast.error('Search failed. Please check your connection and try again.')
      setResults([])
    } finally {
      setIsLoading(false)
    }
  }

  const [downloadingIds, setDownloadingIds] = useState<Set<string>>(new Set())

  const handleDownload = async (result: SearchResult) => {
    if (downloadingIds.has(result.id)) {
      toast.info('Download already in progress for this item')
      return
    }

    setDownloadingIds(prev => new Set(prev).add(result.id))

    try {
      let response: { jobId: string }

      switch (result.type) {
        case 'track':
          response = await apiService.downloadTrack(result.id)
          break
        case 'album':
          response = await apiService.downloadAlbum(result.id)
          break
        default:
          throw new Error('Unknown result type')
      }

      toast.success(`Download started for "${result.title}"`, {
        description: `Job ID: ${response.jobId} - View progress in Downloads tab`,
        duration: 4000
      })
    } catch (error) {
      console.error('Download failed:', error)
      toast.error('Failed to start download. Please try again.')
    } finally {
      // Remove from downloading state after a short delay to prevent rapid re-clicking
      setTimeout(() => {
        setDownloadingIds(prev => {
          const newSet = new Set(prev)
          newSet.delete(result.id)
          return newSet
        })
      }, 2000)
    }
  }

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'track':
        return <Music className="h-4 w-4" />
      case 'album':
        return <Disc className="h-4 w-4" />
      case 'artist':
        return <User className="h-4 w-4" />
      default:
        return <Music className="h-4 w-4" />
    }
  }

  const formatDuration = (seconds?: number) => {
    if (!seconds) return ''
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">Search Music</h1>
        <p className="text-muted-foreground">
          Search for any artist, song, or album to download
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Find Music</CardTitle>
          <CardDescription>
            Search for any artist, song, or album. Choose whether to get individual tracks or full albums.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-3">
            <div className="flex-1">
              <Input
                placeholder="Search for any artist, song, or album..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyPress={handleKeyPress}
                disabled={isLoading}
              />
            </div>
            <Select value={searchType} onValueChange={(value: any) => setSearchType(value)}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="track">Individual Tracks</SelectItem>
                <SelectItem value="album">Full Albums</SelectItem>
              </SelectContent>
            </Select>
            <Button onClick={handleSearch} disabled={isLoading || !query.trim()}>
              <Search className="h-4 w-4 mr-2" />
              {isLoading ? 'Searching...' : 'Search'}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Loading State */}
      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <div className="flex items-center space-x-2">
                  <Skeleton className="h-4 w-4" />
                  <Skeleton className="h-4 w-16" />
                </div>
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-9 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Results */}
      {!isLoading && hasSearched && (
        <>
          {results.length > 0 ? (
            <>
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">
                  Search Results ({results.length})
                </h2>
                <Badge variant="secondary">
                  {searchType === 'track' ? 'Individual Tracks' : 'Full Albums'}
                </Badge>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {results.map((result) => (
                  <Card key={`${result.type}-${result.id}`} className="hover:shadow-md transition-shadow">
                    <CardHeader className="pb-3">
                      <div className="flex items-center space-x-2 mb-2">
                        {getTypeIcon(result.type)}
                        <Badge variant="outline" className="text-xs">
                          {result.type}
                        </Badge>
                      </div>
                      <CardTitle className="text-base line-clamp-2">
                        {result.title}
                      </CardTitle>
                      <CardDescription className="space-y-1">
                        {result.artist && (
                          <div className="text-sm">
                            <span className="font-medium">Artist:</span> {result.artist}
                          </div>
                        )}
                        {result.album && result.type === 'track' && (
                          <div className="text-sm">
                            <span className="font-medium">Album:</span> {result.album}
                          </div>
                        )}
                        {result.duration && (
                          <div className="text-sm">
                            <span className="font-medium">Duration:</span> {formatDuration(result.duration)}
                          </div>
                        )}
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <Button
                        onClick={() => handleDownload(result)}
                        className="w-full"
                        size="sm"
                        disabled={downloadingIds.has(result.id)}
                      >
                        {downloadingIds.has(result.id) ? (
                          <>
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            Starting Download...
                          </>
                        ) : (
                          <>
                            <Download className="h-4 w-4 mr-2" />
                            Download {result.type}
                          </>
                        )}
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="text-center py-12">
                <Search className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No results found</h3>
                <p className="text-muted-foreground mb-4">
                  Try adjusting your search terms or search type
                </p>
                <Button variant="outline" onClick={() => setSearchType('track')}>
                  Try Searching Tracks
                </Button>
              </CardContent>
            </Card>
          )}
        </>
      )}

      {/* Initial State */}
      {!hasSearched && !isLoading && (
        <Card>
          <CardContent className="text-center py-12">
            <Search className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">Ready to search</h3>
            <p className="text-muted-foreground">
              Enter a search term above to find any artist, song, or album
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}