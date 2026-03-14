import { Link } from 'react-router-dom'
import { BookOpenIcon, EyeIcon, StarIcon } from '@heroicons/react/24/solid'

// Component hiển thị một comic card
const ComicCard = ({ comic, showChapters = true }) => {
  // Format views
  const formatViews = (views) => {
    if (!views) return '0'
    if (views >= 1000000) return `${(views / 1000000).toFixed(1)}M`
    if (views >= 1000) return `${(views / 1000).toFixed(1)}K`
    return views.toString()
  }

  // Get status color
  const getStatusColor = (status) => {
    switch (status) {
      case 'completed':
        return 'bg-green-500'
      case 'ongoing':
        return 'bg-blue-500'
      default:
        return 'bg-gray-500'
    }
  }

  const getStatusText = (status) => {
    switch (status) {
      case 'completed':
        return 'Hoàn thành'
      case 'ongoing':
        return 'Đang cập nhật'
      default:
        return status
    }
  }

  return (
    <Link to={`/comics/${comic.id}`} className="group block h-full">
      <div className="card-hover flex flex-col h-full">
        {/* Cover image */}
        <div className="relative aspect-poster overflow-hidden flex-none">
          <img
            src={comic.cover_image || '/placeholder-comic.png'}
            alt={comic.title}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.05]"
            loading="lazy"
          />

          {/* Gradient Overlay for better contrast */}
          <div className="absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-black/80 via-black/20 to-transparent pointer-events-none" />

          {/* Hover overlay */}
          <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex items-center justify-center">
            <div className="flex items-center justify-center gap-2 bg-white/90 px-4 py-2.5 rounded-full shadow-lg shadow-black/30 group-hover:scale-110 transition-transform duration-300">
              <BookOpenIcon className="w-5 h-5 text-gray-900" />
              <span className="text-gray-900 text-sm font-bold">Đọc ngay</span>
            </div>
          </div>

          {/* Status badge */}
          <div className={`absolute top-2.5 left-2.5 px-2.5 py-1 backdrop-blur-md bg-black/50 border border-white/10 text-white text-[10px] font-medium tracking-wide rounded-full shadow-sm`}>
            {/* {getStatusText(comic.status)} */}
            <p>Hoàn thành</p>
          </div>

          {/* Chapters count */}
          {/* {showChapters && comic.chapters && comic.chapters.length > 0 && (
            <div className="absolute top-2 right-2 px-2 py-0.5 bg-black/80 text-white text-xs font-medium rounded">
              {comic.chapters.length} chương
            </div>
          )} */}
        </div>

        {/* Info */}
        <div className="p-3.5 flex-1 flex flex-col">
          <h3 className="text-gray-100 font-semibold truncate whitespace-nowrap group-hover:text-violet-400 transition-colors">
            {comic.title}
          </h3>
          
          {comic.author && (
            <p className="text-[11px] font-medium text-gray-400 mt-1 truncate">
              {comic.author}
            </p>
          )}
          
          <div className="flex items-center justify-between mt-1.5 text-[11px] font-medium text-gray-500">
            <span className="flex items-center space-x-1.5">
              <EyeIcon className="w-3.5 h-3.5" />
              <span>{formatViews(comic.views)}</span>
            </span>
            
            {comic.rating > 0 && (
              <span className="flex items-center space-x-1.5">
                <StarIcon className="w-3.5 h-3.5 text-yellow-500" />
                <span className="text-gray-400">{comic.rating.toFixed(1)}</span>
              </span>
            )}
          </div>

          {/* Genres */}
          {comic.genres && comic.genres.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2.5">
              {comic.genres.slice(0, 2).map((genre, index) => (
                <span key={index} className="px-2 py-0.5 rounded-full bg-white/5 border border-white/5 text-[10px] text-gray-400">
                  {genre}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
    </Link>
  )
}

export default ComicCard
