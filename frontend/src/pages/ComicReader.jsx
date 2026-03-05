import { useParams, useNavigate, Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { 
  ChevronLeftIcon, 
  ChevronRightIcon,
  ArrowLeftIcon,
  ListBulletIcon,
} from '@heroicons/react/24/outline'
import ImageGallery from '../components/ImageGallery'
import { comicsAPI } from '../services/api'

// Comic reader page
const ComicReader = () => {
  const { id, chapter } = useParams()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const chapterNum = parseInt(chapter)
  const initialPage = parseInt(searchParams.get('page')) || 1

  // Fetch comic detail (to get chapter list)
  const { data: comic, isLoading: loadingComic } = useQuery({
    queryKey: ['comic', id],
    queryFn: () => comicsAPI.getById(id),
    select: (res) => res.data,
  })

  // Fetch chapter images
  const { data: chapterData, isLoading: loadingChapter } = useQuery({
    queryKey: ['comic', id, 'chapter', chapterNum],
    queryFn: () => comicsAPI.getChapter(id, chapterNum),
    select: (res) => res.data,
    enabled: !!id && !!chapterNum,
  })

  // Navigation
  const sortedChapters = [...(comic?.chapters || [])].sort((a, b) => a.number - b.number)
  const currentIndex = sortedChapters.findIndex(c => c.number === chapterNum)
  const prevChapter = currentIndex > 0 ? sortedChapters[currentIndex - 1] : null
  const nextChapter = currentIndex < sortedChapters.length - 1 ? sortedChapters[currentIndex + 1] : null

  const goToChapter = (num) => {
    navigate(`/comics/${id}/read/${num}`)
  }

  if (loadingComic || loadingChapter) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="animate-spin w-12 h-12 border-4 border-primary-500 border-t-transparent rounded-full" />
      </div>
    )
  }

  if (!chapterData?.images?.length) {
    return (
      <div className="container-custom py-16 text-center">
        <h1 className="text-2xl text-white mb-4">Chapter không tồn tại hoặc chưa có ảnh</h1>
        <Link to={`/comics/${id}`} className="btn-primary">
          Quay lại truyện
        </Link>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-950">
      {/* Top navigation bar */}
      <div className="sticky top-0 z-50 bg-gray-950/95 backdrop-blur-xl border-b border-white/5">
        <div className="container-custom py-3">
          <div className="flex items-center justify-between">
            {/* Left - Back button + Comic info */}
            <div className="flex items-center gap-2 min-w-0">
              <button 
                onClick={() => navigate(`/comics/${id}`)}
                className="p-2 text-gray-400 hover:text-white rounded-full hover:bg-white/10 active:scale-95 transition-all"
                aria-label="Quay lại"
              >
                <ArrowLeftIcon className="w-5 h-5" />
              </button>
              
              <div className="min-w-0">
                <h1 className="text-white font-medium truncate text-sm md:text-base">{comic?.title}</h1>
                <p className="text-xs md:text-sm text-gray-400">
                  Chapter {chapterNum}
                  {chapterData.title && ` - ${chapterData.title}`}
                </p>
              </div>
            </div>

            {/* Right - Chapter navigation */}
            <div className="flex items-center gap-2">
              <button
                onClick={() => prevChapter && goToChapter(prevChapter.number)}
                disabled={!prevChapter}
                className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                title="Chapter trước"
              >
                <ChevronLeftIcon className="w-5 h-5" />
              </button>

              {/* Chapter selector */}
              <select
                value={chapterNum}
                onChange={(e) => goToChapter(parseInt(e.target.value))}
                className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-primary-500"
              >
                {sortedChapters.map((ch) => (
                  <option key={ch.number} value={ch.number}>
                    Ch. {ch.number}
                  </option>
                ))}
              </select>

              <button
                onClick={() => nextChapter && goToChapter(nextChapter.number)}
                disabled={!nextChapter}
                className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                title="Chapter sau"
              >
                <ChevronRightIcon className="w-5 h-5" />
              </button>

              <Link
                to={`/comics/${id}`}
                className="hidden sm:flex items-center gap-1 px-3 py-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800"
              >
                <ListBulletIcon className="w-5 h-5" />
                <span>Danh sách</span>
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* Image Gallery */}
      <ImageGallery
        images={chapterData.images}
        comicId={id}
        chapter={chapterNum}
        initialPage={initialPage}
      />

      {/* Bottom navigation */}
      <div className="bg-gray-800 border-t border-gray-700 py-4">
        <div className="container-custom">
          <div className="flex items-center justify-between">
            <button
              onClick={() => prevChapter && goToChapter(prevChapter.number)}
              disabled={!prevChapter}
              className="btn-secondary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeftIcon className="w-5 h-5" />
              <span>Chapter trước</span>
            </button>

            <Link to={`/comics/${id}`} className="btn-secondary">
              <ListBulletIcon className="w-5 h-5" />
            </Link>

            <button
              onClick={() => nextChapter && goToChapter(nextChapter.number)}
              disabled={!nextChapter}
              className="btn-primary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <span>Chapter sau</span>
              <ChevronRightIcon className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ComicReader
