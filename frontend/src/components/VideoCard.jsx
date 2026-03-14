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
              className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.05]"
              loading="lazy"
            />
          ) : (
            // Placeholder khi không có thumbnail
            <div className="w-full h-full bg-gradient-to-br from-gray-700 to-gray-800 flex items-center justify-center">
              <FilmIcon className="w-16 h-16 text-gray-500" />
            </div>
          )}
          
          {/* Gradient Overlay for better text legibility */}
          <div className="absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-black/80 via-black/20 to-transparent pointer-events-none" />

          {/* Play icon overlay */}
          <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex items-center justify-center">
            <div className="w-14 h-14 bg-white/90 rounded-full flex items-center justify-center shadow-lg shadow-black/30 group-hover:scale-110 transition-transform duration-300">
              <PlayIcon className="w-7 h-7 text-gray-900 ml-1" />
            </div>
          </div>

          {/* Duration badge */}
          {showDuration && video.duration && (
            <div className="absolute bottom-2.5 right-2.5 px-2.5 py-1 backdrop-blur-md bg-black/50 border border-white/10 text-white text-[10px] font-medium tracking-wide rounded-full shadow-sm">
              {formatDuration(video.duration)}
            </div>
          )}

          {/* Status badge */}
          {video.status === 'processing' && (
            <div className="absolute top-2.5 left-2.5 px-2 py-0.5 bg-amber-500/90 backdrop-blur-sm text-black text-xs font-semibold rounded shadow-sm">
              Đang xử lý
            </div>
          )}
        </div>

        {/* Info */}
        <div className="p-3.5 flex-1 flex flex-col">
          <h3 className="text-gray-100 font-semibold truncate whitespace-nowrap group-hover:text-violet-400 transition-colors">
            {video.title}
          </h3>
          
          <div className="flex items-center space-x-3 mt-1.5 text-[11px] font-medium text-gray-500">
            <span className="flex items-center space-x-1.5">
              <EyeIcon className="w-3.5 h-3.5" />
              <span>{formatViews(video.views)}</span>
            </span>
            
            {video.duration_type && (
              <span className="flex items-center space-x-1.5">
                <ClockIcon className="w-3.5 h-3.5" />
                <span className="capitalize">{video.duration_type}</span>
              </span>
            )}
          </div>

          {/* Tags */}
          {video.genres && video.genres.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2.5">
              {video.genres.slice(0, 2).map((genre, index) => (
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

export default VideoCard
