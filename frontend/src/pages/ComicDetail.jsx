import { useState, useEffect, useMemo } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import { 
  BookOpenIcon, 
  EyeIcon, 
  StarIcon, 
  BookmarkIcon,
  ShareIcon,
  ChevronDownIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import ComicCard from '../components/ComicCard'
import MobileHeader from '../components/MobileHeader'
import OptimizedImage from '../components/OptimizedImage'
import { comicsAPI, userAPI } from '../services/api'
import { useAuthStore } from '../store/authStore'
import { usePlayerStore } from '../store/playerStore'
import { preloadImages } from '../hooks/useImageCache'

// Comic detail page
const ComicDetail = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const { getComicProgress } = usePlayerStore()
  
  // State cho hiển thị ảnh
  const [visibleImages, setVisibleImages] = useState(6)
  const [showedMore, setShowedMore] = useState(false)

  // Fetch comic detail
  const { data: comic, isLoading } = useQuery({
    queryKey: ['comic', id],
    queryFn: () => comicsAPI.getById(id),
    select: (res) => res.data,
  })

  // Fetch related comics
  const { data: relatedComics } = useQuery({
    queryKey: ['comics', 'related', comic?.genres?.[0]],
    queryFn: () => comicsAPI.getAll({ genres: comic?.genres?.[0], limit: 6 }),
    select: (res) => res.data.comics?.filter(c => c.id !== id),
    enabled: !!comic?.genres?.length,
  })

  // Bookmark mutation
  const bookmarkMutation = useMutation({
    mutationFn: () => userAPI.addBookmark({
      content_id: id,
      content_type: 'comic',
      chapter: savedProgress?.chapter || 1,
      page: savedProgress?.page || 1,
    }),
    onSuccess: () => {
      toast.success('Đã thêm vào bookmark')
    },
  })

  // Get saved reading progress
  const savedProgress = getComicProgress(id)

  // Lấy tất cả ảnh từ các chapters (memoized, safe for null comic)
  const allImages = useMemo(() => {
    if (!comic?.chapters) return []
    return comic.chapters.flatMap(chapter => 
      chapter.images?.map(img => ({
        ...img,
        chapter: chapter.number,
      })) || []
    )
  }, [comic?.chapters])

  // Preload displayed images and upcoming images
  useEffect(() => {
    if (allImages.length > 0) {
      const urlsToPreload = allImages.slice(0, visibleImages + 10).map(img => img.url)
      preloadImages(urlsToPreload, true)
    }
  }, [allImages, visibleImages])

  // Format views
  const formatViews = (views) => {
    if (!views) return '0'
    if (views >= 1000000) return `${(views / 1000000).toFixed(1)}M`
    if (views >= 1000) return `${(views / 1000).toFixed(1)}K`
    return views.toString()
  }

  // Format date
  const formatDate = (date) => {
    if (!date) return ''
    return new Date(date).toLocaleDateString('vi-VN')
  }

  if (isLoading) {
    return (
      <div className="container-custom py-8">
        <div className="flex flex-col md:flex-row gap-8">
          <div className="skeleton w-full md:w-64 aspect-poster rounded-xl" />
          <div className="flex-1 space-y-4">
            <div className="skeleton h-8 w-2/3" />
            <div className="skeleton h-4 w-1/3" />
            <div className="skeleton h-20 w-full" />
          </div>
        </div>
      </div>
    )
  }

  if (!comic) {
    return (
      <div className="container-custom py-16 text-center">
        <h1 className="text-2xl text-white mb-4">Truyện không tồn tại</h1>
        <Link to="/comics" className="btn-primary">
          Quay lại danh sách
        </Link>
      </div>
    )
  }

  // Ảnh hiển thị theo state
  const displayedImages = allImages.slice(0, visibleImages)
  const hasMoreImages = allImages.length > visibleImages

  // Xem thêm 6 ảnh
  const handleLoadMore = () => {
    setVisibleImages(prev => prev + 6)
    setShowedMore(true)
  }

  // Xem tất cả - navigate đến reader (với page đã lưu nếu có)
  const handleViewAll = () => {
    if (savedProgress?.page) {
      navigate(`/comics/${id}/read/1?page=${savedProgress.page}`)
    } else {
      navigate(`/comics/${id}/read/1`)
    }
  }

  return (
    <>
      {/* Mobile header with back button */}
      <MobileHeader title={comic.title} />
      
      <div className="container-custom py-8 pt-16 md:pt-8">
        {/* Comic Info Header */}
        <div className="flex flex-col md:flex-row gap-8 mb-8">
        {/* Cover */}
        <div className="w-full md:w-64 flex-shrink-0">
          <img
            src={comic.cover_image || '/placeholder-comic.png'}
            alt={comic.title}
            className="w-full aspect-poster object-cover rounded-xl shadow-lg"
          />
        </div>

        {/* Info */}
        <div className="flex-1">
          <h1 className="text-3xl font-bold text-white mb-2">{comic.title}</h1>
          
          {comic.author && (
            <p className="text-gray-400 mb-4">Tác giả: {comic.author}</p>
          )}

          {/* Stats */}
          <div className="flex flex-wrap items-center gap-4 text-gray-400 mb-4">
            <span className="flex items-center gap-1">
              <EyeIcon className="w-5 h-5" />
              {formatViews(comic.views)} lượt xem
            </span>
            <span className="flex items-center gap-1">
              <BookOpenIcon className="w-5 h-5" />
              {comic.chapters?.length || 0} chương
            </span>
            {comic.rating > 0 && (
              <span className="flex items-center gap-1">
                <StarIcon className="w-5 h-5 text-yellow-400" />
                {comic.rating.toFixed(1)}
              </span>
            )}
          </div>

          {/* Status */}
          <div className="mb-4">
            <span className={`inline-block px-3 py-1 rounded-full text-sm ${
              comic.status === 'completed' ? 'bg-green-500/20 text-green-400' : 'bg-blue-500/20 text-blue-400'
            }`}>
              {comic.status === 'completed' ? 'Hoàn thành' : 'Đang cập nhật'}
            </span>
          </div>

          {/* Genres */}
          {comic.genres?.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-4">
              {comic.genres.map((genre, index) => (
                <Link
                  key={index}
                  to={`/comics?genres=${genre}`}
                  className="badge-secondary hover:bg-gray-600"
                >
                  {genre}
                </Link>
              ))}
            </div>
          )}

          {/* Description */}
          {comic.description && (
            <p className="text-gray-300 mb-6 line-clamp-3">{comic.description}</p>
          )}

          {/* Actions */}
          <div className="flex flex-wrap gap-3">
            <button 
              onClick={handleViewAll}
              className="btn-primary flex items-center gap-2"
              disabled={!allImages.length}
            >
              <BookOpenIcon className="w-5 h-5" />
              {savedProgress ? `Tiếp tục (Trang ${savedProgress.page})` : 'Đọc ngay'}
            </button>

            {user && (
              <button 
                onClick={() => bookmarkMutation.mutate()}
                className="btn-secondary flex items-center gap-2"
              >
                <BookmarkIcon className="w-5 h-5" />
                Đánh dấu
              </button>
            )}

            <button 
              onClick={() => {
                navigator.clipboard.writeText(window.location.href)
                toast.success('Đã copy link')
              }}
              className="btn-secondary flex items-center gap-2"
            >
              <ShareIcon className="w-5 h-5" />
              Chia sẻ
            </button>
          </div>
        </div>
      </div>

      {/* Chapters List */}
      <div className="bg-gray-800 rounded-xl p-4 mb-8">
        <h2 className="text-xl font-bold text-white mb-4">
          Xem trước ({allImages.length} trang)
        </h2>

        {allImages.length > 0 ? (
          <>
            {/* Image Preview Grid */}
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {displayedImages.map((img, index) => (
                <div 
                  key={index}
                  className="relative aspect-[3/4] bg-gray-700 rounded-lg overflow-hidden cursor-pointer group"
                  onClick={() => navigate(`/comics/${id}/read/1?page=${img.page || index + 1}`)}
                >
                  <OptimizedImage
                    src={img.url}
                    alt={`Trang ${img.page || index + 1}`}
                    className="w-full h-full object-cover transition-transform group-hover:scale-105"
                    wrapperClassName="w-full h-full"
                    priority={index < 6} // First 6 load immediately
                  />
                  <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-center justify-center">
                    <span className="opacity-0 group-hover:opacity-100 text-white font-medium transition-opacity">
                      Trang {img.page || index + 1}
                    </span>
                  </div>
                </div>
              ))}
            </div>

            {/* Load More / View All Buttons */}
            <div className="flex justify-center gap-4 mt-6">
              {hasMoreImages && (
                <button
                  onClick={handleLoadMore}
                  className="btn-secondary flex items-center gap-2"
                >
                  <ChevronDownIcon className="w-5 h-5" />
                  Xem thêm ({Math.min(6, allImages.length - visibleImages)} ảnh)
                </button>
              )}
              
              {showedMore && (
                <button
                  onClick={handleViewAll}
                  className="btn-primary flex items-center gap-2"
                >
                  <BookOpenIcon className="w-5 h-5" />
                  Xem tất cả ({allImages.length} trang)
                </button>
              )}
            </div>
          </>
        ) : (
          <p className="text-gray-500 text-center py-8">Chưa có ảnh nào</p>
        )}
      </div>

      {/* Related Comics */}
      {relatedComics?.length > 0 && (
        <div>
          <h2 className="text-xl font-bold text-white mb-4">Truyện liên quan</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
            {relatedComics.slice(0, 6).map((c) => (
              <ComicCard key={c.id} comic={c} />
            ))}
          </div>
        </div>
      )}
    </div>
    </>
  )
}

export default ComicDetail
