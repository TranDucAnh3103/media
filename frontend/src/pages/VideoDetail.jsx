import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { HandThumbUpIcon, BookmarkIcon, ShareIcon, EyeIcon } from '@heroicons/react/24/outline'
import { HandThumbUpIcon as HandThumbUpSolidIcon } from '@heroicons/react/24/solid'
import { useState, useMemo } from 'react'
import toast from 'react-hot-toast'
import Player from '../components/Player'
import VideoCard from '../components/VideoCard'
import MobileHeader from '../components/MobileHeader'
import { videosAPI, userAPI } from '../services/api'
import { useAuthStore } from '../store/authStore'

// Helper: Extract keywords from title for matching
const extractKeywords = (title) => {
  if (!title) return []
  // Remove common words and split into keywords
  const stopWords = ['video', 'clip', 'phim', 'và', 'của', 'trong', 'với', 'cho', 'các', 'những', 'được', 'là', 'có', 'để', 'một', 'này', 'khi', 'từ', 'theo', 'như', 'đã', 'vì', 'tại', 'về', 'ra', 'lên', 'hơn', 'nên', 'làm', 'đến']
  return title
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s]/gu, ' ') // Keep letters, numbers, spaces (Unicode aware)
    .split(/\s+/)
    .filter(word => word.length > 2 && !stopWords.includes(word))
}

// Helper: Calculate relevance score between two videos
const calculateRelevance = (video, targetKeywords) => {
  if (!video?.title || !targetKeywords?.length) return 0
  const videoKeywords = extractKeywords(video.title)
  let score = 0
  for (const keyword of targetKeywords) {
    if (videoKeywords.some(vk => vk.includes(keyword) || keyword.includes(vk))) {
      score += 2 // Exact or partial match
    }
    if (video.title.toLowerCase().includes(keyword)) {
      score += 1 // Title contains keyword
    }
  }
  return score
}

// Video detail page với player và comments
const VideoDetail = () => {
  const { id } = useParams()
  const { user } = useAuthStore()
  const queryClient = useQueryClient()
  const [comment, setComment] = useState('')
  const [liked, setLiked] = useState(false)

  // Fetch video detail
  const { data: video, isLoading } = useQuery({
    queryKey: ['video', id],
    queryFn: () => videosAPI.getById(id),
    select: (res) => res.data,
  })

  // Fetch latest videos for related section (always fetch)
  const { data: latestVideos } = useQuery({
    queryKey: ['videos', 'latest', 20],
    queryFn: () => videosAPI.getLatest(20),
    select: (res) => res.data?.filter(v => v.id !== id) || [],
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  })

  // Calculate related videos based on title similarity + latest
  const relatedVideos = useMemo(() => {
    if (!latestVideos?.length) return []
    if (!video?.title) return latestVideos.slice(0, 8)

    const keywords = extractKeywords(video.title)
    
    // Score each video by relevance
    const scoredVideos = latestVideos.map(v => ({
      ...v,
      relevanceScore: calculateRelevance(v, keywords)
    }))

    // Sort by relevance first, then by created_at (newest first)
    scoredVideos.sort((a, b) => {
      if (b.relevanceScore !== a.relevanceScore) {
        return b.relevanceScore - a.relevanceScore
      }
      // If same relevance, sort by date
      return new Date(b.created_at) - new Date(a.created_at)
    })

    return scoredVideos.slice(0, 8)
  }, [video?.title, latestVideos, id])

  // Like mutation
  const likeMutation = useMutation({
    mutationFn: () => videosAPI.like(id),
    onSuccess: () => {
      setLiked(true)
      queryClient.invalidateQueries(['video', id])
      toast.success('Đã thích video')
    },
  })

  // Comment mutation
  const commentMutation = useMutation({
    mutationFn: (content) => videosAPI.addComment(id, content),
    onSuccess: () => {
      setComment('')
      queryClient.invalidateQueries(['video', id])
      toast.success('Đã thêm bình luận')
    },
  })

  // Bookmark mutation
  const bookmarkMutation = useMutation({
    mutationFn: () => userAPI.addBookmark({
      content_id: id,
      content_type: 'video',
      timestamp: 0,
    }),
    onSuccess: () => {
      toast.success('Đã thêm vào bookmark')
    },
  })

  // Handle comment submit
  const handleComment = (e) => {
    e.preventDefault()
    if (!user) {
      toast.error('Vui lòng đăng nhập để bình luận')
      return
    }
    if (!comment.trim()) return
    commentMutation.mutate(comment)
  }

  // Format date
  const formatDate = (date) => {
    if (!date) return ''
    return new Date(date).toLocaleDateString('vi-VN', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })
  }

  // Format views
  const formatViews = (views) => {
    if (!views) return '0'
    if (views >= 1000000) return `${(views / 1000000).toFixed(1)}M`
    if (views >= 1000) return `${(views / 1000).toFixed(1)}K`
    return views.toString()
  }

  if (isLoading) {
    return (
      <div className="container-custom py-8">
        <div className="skeleton aspect-video rounded-xl mb-4" />
        <div className="skeleton h-8 w-2/3 mb-2" />
        <div className="skeleton h-4 w-1/3" />
      </div>
    )
  }

  if (!video) {
    return (
      <div className="container-custom py-16 text-center">
        <h1 className="text-2xl text-white mb-4">Video không tồn tại</h1>
        <Link to="/videos" className="btn-primary">
          Quay lại danh sách
        </Link>
      </div>
    )
  }

  return (
    <>
      {/* Mobile header with back button */}
      <MobileHeader title={video.title} transparent />
      
      <div className="container-custom py-8 pt-16 md:pt-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Main content */}
          <div className="lg:col-span-2">
            {/* Player */}
          <div className="mb-4">
            <Player
              url={videosAPI.getStreamURL(video)}
              videoId={video.id}
              poster={video.thumbnail}
            />
          </div>

          {/* Video info */}
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-white mb-2">{video.title}</h1>
            
            <div className="flex flex-wrap items-center gap-4 text-gray-400 text-sm mb-4">
              <span className="flex items-center gap-1">
                <EyeIcon className="w-4 h-4" />
                {formatViews(video.views)} lượt xem
              </span>
              <span>{formatDate(video.created_at)}</span>
            </div>

            {/* Actions */}
            <div className="flex flex-wrap items-center gap-3 pb-4 border-b border-gray-700">
              <button 
                onClick={() => !liked && user && likeMutation.mutate()}
                disabled={liked || likeMutation.isLoading}
                className={`flex items-center gap-1 px-4 py-2 rounded-full transition-colors ${
                  liked ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                }`}
              >
                {liked ? (
                  <HandThumbUpSolidIcon className="w-5 h-5" />
                ) : (
                  <HandThumbUpIcon className="w-5 h-5" />
                )}
                <span>{video.likes || 0}</span>
              </button>

              {user && (
                <button 
                  onClick={() => bookmarkMutation.mutate()}
                  className="flex items-center gap-1 px-4 py-2 bg-gray-700 text-gray-300 rounded-full hover:bg-gray-600"
                >
                  <BookmarkIcon className="w-5 h-5" />
                  <span>Lưu</span>
                </button>
              )}

              <button 
                onClick={() => {
                  navigator.clipboard.writeText(window.location.href)
                  toast.success('Đã copy link')
                }}
                className="flex items-center gap-1 px-4 py-2 bg-gray-700 text-gray-300 rounded-full hover:bg-gray-600"
              >
                <ShareIcon className="w-5 h-5" />
                <span>Chia sẻ</span>
              </button>
            </div>

            {/* Description */}
            {video.description && (
              <div className="py-4 border-b border-gray-700">
                <p className="text-gray-300 whitespace-pre-wrap">{video.description}</p>
              </div>
            )}

            {/* Tags */}
            {video.genres?.length > 0 && (
              <div className="flex flex-wrap gap-2 py-4">
                {video.genres.map((genre, index) => (
                  <Link
                    key={index}
                    to={`/videos?genres=${genre}`}
                    className="badge-secondary hover:bg-gray-600"
                  >
                    #{genre}
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Comments */}
          <div className="bg-gray-800 rounded-xl p-4">
            <h3 className="font-bold text-white mb-4">
              Bình luận ({video.comments?.length || 0})
            </h3>

            {/* Comment form */}
            {user ? (
              <form onSubmit={handleComment} className="mb-6">
                <textarea
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="Viết bình luận..."
                  rows="3"
                  className="input resize-none mb-2"
                />
                <button 
                  type="submit" 
                  disabled={!comment.trim() || commentMutation.isLoading}
                  className="btn-primary disabled:opacity-50"
                >
                  {commentMutation.isLoading ? 'Đang gửi...' : 'Gửi'}
                </button>
              </form>
            ) : (
              <p className="text-gray-400 mb-4">
                <Link to="/login" className="text-primary-400 hover:underline">Đăng nhập</Link> để bình luận
              </p>
            )}

            {/* Comments list */}
            <div className="space-y-4">
              {video.comments?.map((cmt) => (
                <div key={cmt.id} className="flex gap-3">
                  <div className="w-10 h-10 bg-gray-700 rounded-full flex items-center justify-center text-gray-400">
                    {cmt.username?.[0]?.toUpperCase() || 'U'}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium text-white">{cmt.username}</span>
                      <span className="text-xs text-gray-500">{formatDate(cmt.created_at)}</span>
                    </div>
                    <p className="text-gray-300">{cmt.content}</p>
                  </div>
                </div>
              ))}

              {(!video.comments || video.comments.length === 0) && (
                <p className="text-gray-500 text-center py-4">Chưa có bình luận nào</p>
              )}
            </div>
          </div>
        </div>

        {/* Sidebar - Related videos */}
        <div className="lg:col-span-1">
          <h3 className="font-bold text-white mb-4">Video liên quan</h3>
          <div className="space-y-4">
            {relatedVideos?.length > 0 ? (
              relatedVideos.map((vid) => (
                <VideoCard key={vid.id} video={vid} />
              ))
            ) : (
              <p className="text-gray-400 text-sm">Không có video liên quan</p>
            )}
          </div>
        </div>
      </div>
    </div>
    </>
  )
}

export default VideoDetail
