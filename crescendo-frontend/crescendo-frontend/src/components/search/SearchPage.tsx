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
  const [searchType, setSearchType] = useState<'track' | 'album'>('album')
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
        <h1 style={{
          fontSize: '24px',
          fontWeight: 'bold',
          color: '#f1f5f9',
          marginBottom: '8px'
        }}>Search Music</h1>
        <p style={{
          color: '#94a3b8',
          fontSize: '14px'
        }}>
          Search for any artist, song, or album to download
        </p>
      </div>

      <div style={{
        backgroundColor: '#1e293b',
        border: '1px solid #475569',
        borderRadius: '8px',
        padding: '24px'
      }}>
        <div style={{ marginBottom: '16px' }}>
          <h2 style={{
            fontSize: '20px',
            fontWeight: 'bold',
            color: '#f1f5f9',
            marginBottom: '8px'
          }}>Find Music</h2>
          <p style={{
            fontSize: '14px',
            color: '#94a3b8'
          }}>
            Search for any artist, song, or album. Choose whether to get individual tracks or full albums.
          </p>
        </div>
        <div style={{ display: 'flex', gap: '12px' }}>
          <div style={{ flex: 1 }}>
            <input
              type="text"
              placeholder="Search for any artist, song, or album..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyPress={handleKeyPress}
              disabled={isLoading}
              style={{
                width: '100%',
                padding: '12px 16px',
                backgroundColor: '#0f172a',
                border: '1px solid #475569',
                borderRadius: '6px',
                color: '#f1f5f9',
                fontSize: '14px',
                outline: 'none'
              }}
              onFocus={(e) => {
                e.target.style.borderColor = '#3b82f6'
              }}
              onBlur={(e) => {
                e.target.style.borderColor = '#475569'
              }}
            />
          </div>
          <select
            value={searchType}
            onChange={(e) => setSearchType(e.target.value as 'track' | 'album')}
            style={{
              padding: '12px 16px',
              backgroundColor: '#0f172a',
              border: '1px solid #475569',
              borderRadius: '6px',
              color: '#f1f5f9',
              fontSize: '14px',
              outline: 'none',
              minWidth: '140px'
            }}
          >
            <option value="album">Full Albums</option>
            <option value="track">Individual Tracks</option>
          </select>
          <button
            onClick={handleSearch}
            disabled={isLoading || !query.trim()}
            style={{
              padding: '12px 24px',
              backgroundColor: isLoading || !query.trim() ? '#475569' : '#3b82f6',
              border: 'none',
              borderRadius: '6px',
              color: '#ffffff',
              fontSize: '14px',
              fontWeight: '500',
              cursor: isLoading || !query.trim() ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '8px'
            }}
          >
            <Search className="h-4 w-4" />
            {isLoading ? 'Searching...' : 'Search'}
          </button>
        </div>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '16px' }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} style={{
              backgroundColor: '#1e293b',
              border: '1px solid #475569',
              borderRadius: '8px',
              padding: '20px'
            }}>
              <div style={{ marginBottom: '12px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                  <div style={{ width: '16px', height: '16px', backgroundColor: '#475569', borderRadius: '4px' }}></div>
                  <div style={{ width: '60px', height: '16px', backgroundColor: '#475569', borderRadius: '4px' }}></div>
                </div>
                <div style={{ width: '80%', height: '20px', backgroundColor: '#475569', borderRadius: '4px', marginBottom: '8px' }}></div>
                <div style={{ width: '60%', height: '16px', backgroundColor: '#475569', borderRadius: '4px' }}></div>
              </div>
              <div style={{ width: '100%', height: '40px', backgroundColor: '#475569', borderRadius: '6px' }}></div>
            </div>
          ))}
        </div>
      )}

      {/* Results */}
      {!isLoading && hasSearched && (
        <>
          {results.length > 0 ? (
            <>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
                <h2 style={{
                  fontSize: '18px',
                  fontWeight: '600',
                  color: '#f1f5f9'
                }}>
                  Search Results ({results.length})
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
                  {searchType === 'track' ? 'Individual Tracks' : 'Full Albums'}
                </span>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '16px' }}>
                {results.map((result) => (
                  <div key={`${result.type}-${result.id}`} style={{
                    backgroundColor: '#1e293b',
                    border: '1px solid #475569',
                    borderRadius: '8px',
                    padding: '20px',
                    transition: 'all 0.2s'
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor = '#334155'
                    e.currentTarget.style.borderColor = '#64748b'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = '#1e293b'
                    e.currentTarget.style.borderColor = '#475569'
                  }}>
                    <div style={{ marginBottom: '12px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                        {result.cover ? (
                          <img
                            src={result.cover}
                            alt={`${result.title} cover`}
                            style={{
                              width: '40px',
                              height: '40px',
                              borderRadius: '4px',
                              objectFit: 'cover',
                              backgroundColor: '#475569'
                            }}
                            onError={(e) => {
                              // Fallback to icon if image fails to load
                              e.currentTarget.style.display = 'none';
                              e.currentTarget.nextElementSibling!.style.display = 'inline';
                            }}
                          />
                        ) : null}
                        <span style={{
                          color: '#94a3b8',
                          display: result.cover ? 'none' : 'inline'
                        }}>{getTypeIcon(result.type)}</span>
                        <span style={{
                          padding: '2px 8px',
                          backgroundColor: '#0f172a',
                          border: '1px solid #475569',
                          borderRadius: '4px',
                          fontSize: '12px',
                          color: '#94a3b8',
                          fontWeight: '500'
                        }}>
                          {result.type}
                        </span>
                      </div>
                      <h3 style={{
                        fontSize: '16px',
                        fontWeight: '600',
                        color: '#f1f5f9',
                        marginBottom: '8px',
                        lineHeight: '1.4'
                      }}>
                        {result.title}
                      </h3>
                      <div style={{ color: '#94a3b8', fontSize: '14px' }}>
                        {result.artist && (
                          <div style={{ marginBottom: '4px' }}>
                            <span style={{ fontWeight: '500', color: '#e2e8f0' }}>Artist:</span> {result.artist}
                          </div>
                        )}
                        {result.album && result.type === 'track' && (
                          <div style={{ marginBottom: '4px' }}>
                            <span style={{ fontWeight: '500', color: '#e2e8f0' }}>Album:</span> {result.album}
                          </div>
                        )}
                        {result.duration && (
                          <div>
                            <span style={{ fontWeight: '500', color: '#e2e8f0' }}>Duration:</span> {formatDuration(result.duration)}
                          </div>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={() => handleDownload(result)}
                      disabled={downloadingIds.has(result.id)}
                      style={{
                        width: '100%',
                        padding: '12px 16px',
                        backgroundColor: downloadingIds.has(result.id) ? '#475569' : '#3b82f6',
                        border: 'none',
                        borderRadius: '6px',
                        color: '#ffffff',
                        fontSize: '14px',
                        fontWeight: '500',
                        cursor: downloadingIds.has(result.id) ? 'not-allowed' : 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        gap: '8px'
                      }}
                    >
                      {downloadingIds.has(result.id) ? (
                        <>
                          <Loader2 className="h-4 w-4 animate-spin" />
                          Starting Download...
                        </>
                      ) : (
                        <>
                          <Download className="h-4 w-4" />
                          Download {result.type}
                        </>
                      )}
                    </button>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div style={{
              backgroundColor: '#1e293b',
              border: '1px solid #475569',
              borderRadius: '8px',
              padding: '48px 24px',
              textAlign: 'center'
            }}>
              <Search className="h-12 w-12 mx-auto mb-4" style={{ color: '#94a3b8' }} />
              <h3 style={{
                fontSize: '18px',
                fontWeight: '600',
                color: '#f1f5f9',
                marginBottom: '8px'
              }}>No results found</h3>
              <p style={{
                fontSize: '14px',
                color: '#94a3b8',
                marginBottom: '16px'
              }}>
                Try adjusting your search terms or search type
              </p>
              <button
                onClick={() => setSearchType('track')}
                style={{
                  padding: '8px 16px',
                  backgroundColor: 'transparent',
                  border: '1px solid #475569',
                  borderRadius: '6px',
                  color: '#94a3b8',
                  fontSize: '14px',
                  cursor: 'pointer'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = '#475569'
                  e.currentTarget.style.color = '#e2e8f0'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = 'transparent'
                  e.currentTarget.style.color = '#94a3b8'
                }}
              >
                Try Searching Individual Tracks
              </button>
            </div>
          )}
        </>
      )}

      {/* Initial State */}
      {!hasSearched && !isLoading && (
        <div style={{
          backgroundColor: '#1e293b',
          border: '1px solid #475569',
          borderRadius: '8px',
          padding: '48px 24px',
          textAlign: 'center'
        }}>
          <Search className="h-12 w-12 mx-auto mb-4" style={{ color: '#94a3b8' }} />
          <h3 style={{
            fontSize: '18px',
            fontWeight: '600',
            color: '#f1f5f9',
            marginBottom: '8px'
          }}>Ready to search</h3>
          <p style={{
            fontSize: '14px',
            color: '#94a3b8'
          }}>
            Enter a search term above to find any artist, song, or album
          </p>
        </div>
      )}
    </div>
  )
}