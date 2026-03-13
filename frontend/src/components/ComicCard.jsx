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
            className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-110"
            loading="lazy"
          />
          
          {/* Hover overlay */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity">
            <div className="absolute bottom-0 left-0 right-0 p-3">
              <div className="flex items-center justify-center space-x-1 text-white">
                <BookOpenIcon className="w-5 h-5" />
                <span>Đọc ngay</span>
              </div>
            </div>
          </div>

          {/* Status badge */}
          <div className={`absolute top-2 left-2 px-2 py-0.5 ${getStatusColor(comic.status)} text-white text-xs font-medium rounded`}>
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
        <div className="p-3 flex-1 flex flex-col">
          <h3 className="text-white font-medium truncate whitespace-nowrap group-hover:text-primary-400 transition-colors">
            {comic.title}
          </h3>
          
          {comic.author && (
            <p className="text-sm text-gray-500 mt-1 truncate">
              {comic.author}
            </p>
          )}
          
          <div className="flex items-center justify-between mt-2 text-sm text-gray-400">
            <span className="flex items-center space-x-1">
              <EyeIcon className="w-4 h-4" />
              <span>{formatViews(comic.views)}</span>
            </span>
            
            {comic.rating > 0 && (
              <span className="flex items-center space-x-1">
                <StarIcon className="w-4 h-4 text-yellow-400" />
                <span>{comic.rating.toFixed(1)}</span>
              </span>
            )}
          </div>

          {/* Genres */}
          {comic.genres && comic.genres.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {comic.genres.slice(0, 2).map((genre, index) => (
                <span key={index} className="badge-secondary text-xs">
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
