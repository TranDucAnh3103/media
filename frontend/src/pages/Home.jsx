import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { ChevronRightIcon, FireIcon, PlayIcon } from '@heroicons/react/24/solid'
import VideoCard from '../components/VideoCard'
import ComicCard from '../components/ComicCard'
import { videosAPI, comicsAPI } from '../services/api'

// Home page - Trang chủ với hero comic và trending content
const Home = () => {
  // Fetch latest comics for hero
  const { data: latestComics } = useQuery({
    queryKey: ['comics', 'latest'],
    queryFn: () => comicsAPI.getLatest(1),
    select: (res) => res.data,
  })

  // Fetch trending videos
  const { data: trendingVideos, isLoading: loadingVideos } = useQuery({
    queryKey: ['videos', 'trending'],
    queryFn: () => videosAPI.getTrending(12),
    select: (res) => res.data,
  })

  // Fetch trending comics
  const { data: trendingComics, isLoading: loadingComics } = useQuery({
    queryKey: ['comics', 'trending'],
    queryFn: () => comicsAPI.getTrending(12),
    select: (res) => res.data,
  })

  // Get hero comic (latest)
  const heroComic = latestComics?.[0]

  // Responsive: show 12 on desktop, 6 on mobile
  const [isMd, setIsMd] = useState(false)
  useEffect(() => {
    const mq = window.matchMedia('(min-width: 768px)')
    const onChange = (e) => setIsMd(e.matches)
    // set initial
    setIsMd(mq.matches)
    if (mq.addEventListener) mq.addEventListener('change', onChange)
    else mq.addListener(onChange)
    return () => {
      if (mq.removeEventListener) mq.removeEventListener('change', onChange)
      else mq.removeListener(onChange)
    }
  }, [])

  const cardsCount = isMd ? 12 : 6

  return (
    <div className="min-h-screen">
      {/* Hero Section - Latest Comic */}
      <section className="relative h-[50vh] md:h-[60vh] overflow-hidden">
        {/* Background Image */}
        {heroComic?.cover_image ? (
          <div
            className="absolute inset-0 bg-cover bg-center"
            style={{
              backgroundImage: `url(${heroComic.cover_image})`,
            }}
          >
            {/* Overlay gradient */}
            <div className="absolute inset-0 bg-gradient-to-t from-gray-950 via-gray-950/70 to-gray-950/30" />
            <div className="absolute inset-0 bg-gradient-to-r from-gray-950/80 to-transparent" />
          </div>
        ) : (
          <div className="absolute inset-0 bg-gradient-to-br from-violet-900/50 via-gray-900 to-gray-950" />
        )}

        {/* Content */}
        <div className="relative h-full container-custom flex items-end pb-8 md:pb-12">
          <div className="max-w-2xl">
            {/* Badge */}
            <div className="inline-flex items-center gap-2 px-3 py-1.5 bg-violet-500/20 backdrop-blur-sm border border-violet-500/30 rounded-full mb-4">
              <span className="w-2 h-2 bg-violet-400 rounded-full animate-pulse" />
              <span className="text-sm text-violet-300 font-medium">Mới nhất</span>
            </div>

            {/* Title */}
            <h1 className="text-3xl md:text-5xl font-bold text-white mb-3 line-clamp-2">
              {heroComic?.title || 'Media Hub'}
            </h1>

            {/* Description */}
            <p className="text-gray-300 text-sm md:text-base mb-6 line-clamp-2">
              {heroComic?.description || 'Khám phá truyện tranh và video hấp dẫn'}
            </p>

            {/* Actions */}
            <div className="flex flex-wrap gap-3">
              {heroComic ? (
                <>
                  <Link
                    to={`/comics/${heroComic.id}`}
                    className="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-violet-500 to-fuchsia-500 text-white font-semibold rounded-xl hover:shadow-lg hover:shadow-violet-500/25 transition-all"
                  >
                    <PlayIcon className="w-5 h-5" />
                    Đọc ngay
                  </Link>
                  <Link
                    to="/comics"
                    className="px-6 py-3 bg-white/10 backdrop-blur-sm text-white font-semibold rounded-xl border border-white/20 hover:bg-white/20 transition-all"
                  >
                    Xem tất cả truyện
                  </Link>
                </>
              ) : (
                <>
                  <Link to="/videos" className="btn-primary">
                    Xem Videos
                  </Link>
                  <Link to="/comics" className="btn-secondary">
                    Đọc Truyện
                  </Link>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Bottom fade */}
        <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-t from-gray-950 to-transparent pointer-events-none" />
      </section>

      {/* Trending Comics */}
      <section className="container-custom py-8 pb-12">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl md:text-2xl font-bold text-white flex items-center space-x-2">
            <FireIcon className="w-6 h-6 md:w-7 md:h-7 text-orange-500" />
            <span>Truyện Hot</span>
          </h2>
          <Link to="/comics" className="flex items-center text-violet-400 hover:text-violet-300 text-sm md:text-base">
            <span>Xem tất cả</span>
            <ChevronRightIcon className="w-5 h-5" />
          </Link>
        </div>

        {loadingComics ? (
          <div className="grid-cards">
            {[...Array(cardsCount)].map((_, i) => (
              <div key={i} className="card">
                <div className="skeleton aspect-poster" />
                <div className="p-3 space-y-2">
                  <div className="skeleton h-4 w-3/4" />
                  <div className="skeleton h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="grid-cards">
            {(trendingComics || []).slice(0, cardsCount).map((comic) => (
              <ComicCard key={comic.id} comic={comic} />
            ))}
          </div>
        )}
      </section>

      {/* Trending Videos */}
      <section className="container-custom py-8">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl md:text-2xl font-bold text-white flex items-center space-x-2">
            <FireIcon className="w-6 h-6 md:w-7 md:h-7 text-orange-500" />
            <span>Videos Trending</span>
          </h2>
          <Link to="/videos" className="flex items-center text-violet-400 hover:text-violet-300 text-sm md:text-base">
            <span>Xem tất cả</span>
            <ChevronRightIcon className="w-5 h-5" />
          </Link>
        </div>

        {loadingVideos ? (
          <div className="grid-cards">
            {[...Array(cardsCount)].map((_, i) => (
              <div key={i} className="card">
                <div className="skeleton aspect-video-custom" />
                <div className="p-3 space-y-2">
                  <div className="skeleton h-4 w-3/4" />
                  <div className="skeleton h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="grid-cards">
            {(trendingVideos || []).slice(0, cardsCount).map((video) => (
              <VideoCard key={video.id} video={video} />
            ))}
          </div>
        )}
      </section>

     
    </div>
  )
}

export default Home
