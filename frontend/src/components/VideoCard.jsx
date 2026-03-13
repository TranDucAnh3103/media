import { Link } from 'react-router-dom'
import { PlayIcon, EyeIcon, ClockIcon, FilmIcon } from '@heroicons/react/24/solid'

// Component hiển thị một video card
const VideoCard = ({ video, showDuration = true }) => {
  // Format duration từ giây sang mm:ss
  const formatDuration = (seconds) => {
    if (!seconds) return '0:00'
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  // Format views
  const formatViews = (views) => {
    if (!views) return '0'
    if (views >= 1000000) return `${(views / 1000000).toFixed(1)}M`
    if (views >= 1000) return `${(views / 1000).toFixed(1)}K`
    return views.toString()
  }

  const hasThumbnail = video.thumbnail && video.thumbnail.trim() !== ''

  return (
    <Link to={`/videos/${video.id}`} className="group block h-full">
      <div className="card-hover flex flex-col h-full">
        {/* Thumbnail */}
        <div className="relative aspect-video-custom overflow-hidden flex-none">
          {hasThumbnail ? (
            <img
              src={video.thumbnail}
              alt={video.title}
              className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-110"
              loading="lazy"
            />
          ) : (
            // Placeholder khi không có thumbnail
            <div className="w-full h-full bg-gradient-to-br from-gray-700 to-gray-800 flex items-center justify-center">
              <FilmIcon className="w-16 h-16 text-gray-500" />
            </div>
          )}
          
          {/* Play icon overlay */}
          <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
            <div className="w-14 h-14 bg-white/90 rounded-full flex items-center justify-center">
              <PlayIcon className="w-7 h-7 text-gray-900 ml-1" />
            </div>
          </div>

          {/* Duration badge */}
          {showDuration && video.duration && (
            <div className="absolute bottom-2 right-2 px-2 py-0.5 bg-black/80 text-white text-xs font-medium rounded">
              {formatDuration(video.duration)}
            </div>
          )}

          {/* Status badge */}
          {video.status === 'processing' && (
            <div className="absolute top-2 left-2 px-2 py-0.5 bg-yellow-500 text-black text-xs font-medium rounded">
              Đang xử lý
            </div>
          )}
        </div>

        {/* Info */}
        <div className="p-3 flex-1 flex flex-col">
          <h3 className="text-white font-medium truncate whitespace-nowrap group-hover:text-primary-400 transition-colors">
            {video.title}
          </h3>
          
          <div className="flex items-center space-x-3 mt-2 text-sm text-gray-400">
            <span className="flex items-center space-x-1">
              <EyeIcon className="w-4 h-4" />
              <span>{formatViews(video.views)}</span>
            </span>
            
            {video.duration_type && (
              <span className="flex items-center space-x-1">
                <ClockIcon className="w-4 h-4" />
                <span className="capitalize">{video.duration_type}</span>
              </span>
            )}
          </div>

          {/* Tags */}
          {video.genres && video.genres.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {video.genres.slice(0, 2).map((genre, index) => (
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

export default VideoCard
