import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import VideoCard from '../components/VideoCard'
import { videosAPI } from '../services/api'

// Videos page - Danh sách video với pagination
const Videos = () => {
  const [searchParams, setSearchParams] = useSearchParams()
  const [search, setSearch] = useState(searchParams.get('search') || '')

  // Parse params
  const page = parseInt(searchParams.get('page') || '1')

  // Fetch videos
  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['videos', { page, search: searchParams.get('search') }],
    queryFn: () => videosAPI.getAll({
      page,
      limit: 12,
      search: searchParams.get('search'),
    }),
    select: (res) => res.data,
    keepPreviousData: true,
  })

  // Update search params
  const updateParams = (key, value) => {
    const params = new URLSearchParams(searchParams)
    if (value) {
      params.set(key, value)
    } else {
      params.delete(key)
    }
    if (key !== 'page') params.set('page', '1') // Reset to page 1 when not changing page
    setSearchParams(params)
  }

  // Handle search
  const handleSearch = (e) => {
    e.preventDefault()
    updateParams('search', search)
  }

  return (
    <div className="container-custom py-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8">
        <h1 className="text-3xl font-bold text-white">Videos</h1>
        
        {/* Search */}
        <form onSubmit={handleSearch} className="relative flex-1 md:w-80 md:max-w-md">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Tìm kiếm video..."
            className="input pr-10"
          />
          <button type="submit" className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white">
            <MagnifyingGlassIcon className="w-5 h-5" />
          </button>
        </form>
      </div>

      {/* Results */}
      {isLoading ? (
        <div className="grid-cards">
          {[...Array(12)].map((_, i) => (
            <div key={i} className="card">
              <div className="skeleton aspect-video-custom" />
              <div className="p-3 space-y-2">
                <div className="skeleton h-4 w-3/4" />
                <div className="skeleton h-3 w-1/2" />
              </div>
            </div>
          ))}
        </div>
      ) : data?.videos?.length > 0 ? (
        <>
          <div className="grid-cards">
            {data.videos.map((video) => (
              <VideoCard key={video.id} video={video} />
            ))}
          </div>

          {/* Pagination */}
          {data.total_pages > 1 && (
            <div className="flex justify-center mt-8 gap-2">
              <button
                disabled={page <= 1}
                onClick={() => updateParams('page', String(page - 1))}
                className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Trước
              </button>
              
              <span className="flex items-center px-4 text-gray-400">
                Trang {page} / {data.total_pages}
              </span>
              
              <button
                disabled={page >= data.total_pages}
                onClick={() => updateParams('page', String(page + 1))}
                className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Sau
              </button>
            </div>
          )}
        </>
      ) : (
        <div className="text-center py-16">
          <p className="text-gray-400 text-lg">Không tìm thấy video nào</p>
          {searchParams.get('search') && (
            <button onClick={() => { setSearch(''); updateParams('search', ''); }} className="btn-primary mt-4">
              Xóa tìm kiếm
            </button>
          )}
        </div>
      )}

      {/* Loading overlay */}
      {isFetching && !isLoading && (
        <div className="fixed inset-0 bg-black/20 flex items-center justify-center z-50">
          <div className="animate-spin w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full" />
        </div>
      )}
    </div>
  )
}

export default Videos
