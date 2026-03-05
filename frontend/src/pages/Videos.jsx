import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { MagnifyingGlassIcon, FunnelIcon, XMarkIcon } from '@heroicons/react/24/outline'
import VideoCard from '../components/VideoCard'
import { videosAPI } from '../services/api'

// Danh sách thể loại video
const genres = [
  'Hành động', 'Hài hước', 'Kinh dị', 'Tình cảm', 
  'Khoa học viễn tưởng', 'Hoạt hình', 'Tài liệu', 'Âm nhạc'
]

const durationTypes = [
  { value: 'short', label: 'Ngắn (<5 phút)' },
  { value: 'medium', label: 'Trung bình (5-10 phút)' },
  { value: 'long', label: 'Dài (>10 phút)' },
]

// Videos page - Danh sách video với filter
const Videos = () => {
  const [searchParams, setSearchParams] = useSearchParams()
  const [showFilters, setShowFilters] = useState(false)
  const [search, setSearch] = useState(searchParams.get('search') || '')

  // Parse params
  const page = parseInt(searchParams.get('page') || '1')
  const selectedGenres = searchParams.get('genres')?.split(',').filter(Boolean) || []
  const durationType = searchParams.get('duration_type') || ''
  const sortBy = searchParams.get('sort_by') || 'created_at'

  // Fetch videos
  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['videos', { page, genres: selectedGenres, durationType, sortBy, search: searchParams.get('search') }],
    queryFn: () => videosAPI.getAll({
      page,
      limit: 24,
      genres: selectedGenres.join(','),
      duration_type: durationType,
      sort_by: sortBy,
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
    params.set('page', '1') // Reset to page 1
    setSearchParams(params)
  }

  // Handle search
  const handleSearch = (e) => {
    e.preventDefault()
    updateParams('search', search)
  }

  // Toggle genre
  const toggleGenre = (genre) => {
    const newGenres = selectedGenres.includes(genre)
      ? selectedGenres.filter(g => g !== genre)
      : [...selectedGenres, genre]
    updateParams('genres', newGenres.join(','))
  }

  // Clear all filters
  const clearFilters = () => {
    setSearchParams({ page: '1' })
    setSearch('')
  }

  const hasFilters = selectedGenres.length > 0 || durationType || searchParams.get('search')

  return (
    <div className="container-custom py-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8">
        <h1 className="text-3xl font-bold text-white">Videos</h1>
        
        <div className="flex items-center gap-3">
          {/* Search */}
          <form onSubmit={handleSearch} className="relative flex-1 md:w-80">
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

          {/* Filter toggle */}
          <button 
            onClick={() => setShowFilters(!showFilters)}
            className={`btn-secondary flex items-center gap-2 ${showFilters ? 'bg-primary-600' : ''}`}
          >
            <FunnelIcon className="w-5 h-5" />
            <span className="hidden sm:inline">Lọc</span>
            {hasFilters && (
              <span className="w-2 h-2 bg-primary-400 rounded-full" />
            )}
          </button>
        </div>
      </div>

      {/* Filters panel */}
      {showFilters && (
        <div className="bg-gray-800 rounded-xl p-4 mb-6 animate-slide-up">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-medium text-white">Bộ lọc</h3>
            {hasFilters && (
              <button onClick={clearFilters} className="text-sm text-primary-400 hover:text-primary-300">
                Xóa tất cả
              </button>
            )}
          </div>

          {/* Genres */}
          <div className="mb-4">
            <h4 className="text-sm text-gray-400 mb-2">Thể loại</h4>
            <div className="flex flex-wrap gap-2">
              {genres.map((genre) => (
                <button
                  key={genre}
                  onClick={() => toggleGenre(genre)}
                  className={`px-3 py-1 rounded-full text-sm transition-colors ${
                    selectedGenres.includes(genre)
                      ? 'bg-primary-600 text-white'
                      : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                  }`}
                >
                  {genre}
                </button>
              ))}
            </div>
          </div>

          {/* Duration */}
          <div className="mb-4">
            <h4 className="text-sm text-gray-400 mb-2">Thời lượng</h4>
            <div className="flex flex-wrap gap-2">
              {durationTypes.map((dt) => (
                <button
                  key={dt.value}
                  onClick={() => updateParams('duration_type', durationType === dt.value ? '' : dt.value)}
                  className={`px-3 py-1 rounded-full text-sm transition-colors ${
                    durationType === dt.value
                      ? 'bg-primary-600 text-white'
                      : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                  }`}
                >
                  {dt.label}
                </button>
              ))}
            </div>
          </div>

          {/* Sort */}
          <div>
            <h4 className="text-sm text-gray-400 mb-2">Sắp xếp</h4>
            <select
              value={sortBy}
              onChange={(e) => updateParams('sort_by', e.target.value)}
              className="input w-auto"
            >
              <option value="created_at">Mới nhất</option>
              <option value="views">Lượt xem</option>
              <option value="likes">Lượt thích</option>
            </select>
          </div>
        </div>
      )}

      {/* Active filters */}
      {hasFilters && (
        <div className="flex flex-wrap items-center gap-2 mb-4">
          <span className="text-sm text-gray-400">Đang lọc:</span>
          {selectedGenres.map((genre) => (
            <button
              key={genre}
              onClick={() => toggleGenre(genre)}
              className="flex items-center gap-1 px-2 py-1 bg-primary-600/20 text-primary-400 rounded text-sm"
            >
              {genre}
              <XMarkIcon className="w-4 h-4" />
            </button>
          ))}
          {durationType && (
            <button
              onClick={() => updateParams('duration_type', '')}
              className="flex items-center gap-1 px-2 py-1 bg-primary-600/20 text-primary-400 rounded text-sm"
            >
              {durationTypes.find(d => d.value === durationType)?.label}
              <XMarkIcon className="w-4 h-4" />
            </button>
          )}
          {searchParams.get('search') && (
            <button
              onClick={() => { updateParams('search', ''); setSearch(''); }}
              className="flex items-center gap-1 px-2 py-1 bg-primary-600/20 text-primary-400 rounded text-sm"
            >
              "{searchParams.get('search')}"
              <XMarkIcon className="w-4 h-4" />
            </button>
          )}
        </div>
      )}

      {/* Results */}
      {isLoading ? (
        <div className="grid-cards">
          {[...Array(24)].map((_, i) => (
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
          {hasFilters && (
            <button onClick={clearFilters} className="btn-primary mt-4">
              Xóa bộ lọc
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
