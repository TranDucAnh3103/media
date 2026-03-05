import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { 
  ChevronLeftIcon, 
  ChevronRightIcon,
  ArrowsPointingOutIcon,
  ArrowsPointingInIcon,
  Bars3Icon,
  Cog6ToothIcon,
} from '@heroicons/react/24/solid'
import { usePlayerStore } from '../store/playerStore'
import OptimizedImage from './OptimizedImage'
import { preloadImages, useImagePreloader } from '../hooks/useImageCache'

// Component đọc truyện với swipe/scroll, bookmark
const ImageGallery = ({ 
  images, 
  comicId, 
  chapter,
  onPageChange,
  initialPage = 1 
}) => {
  const containerRef = useRef(null)
  const [currentPage, setCurrentPage] = useState(initialPage)
  const [fullscreen, setFullscreen] = useState(false)
  const [showControls, setShowControls] = useState(true)
  const [showSettings, setShowSettings] = useState(false)
  const [imageLoaded, setImageLoaded] = useState({})

  const { 
    readerSettings, 
    updateReaderSettings, 
    saveComicProgress,
    getComicProgress 
  } = usePlayerStore()

  const { mode, direction, fitMode } = readerSettings

  // Extract image URLs for preloading
  const imageUrls = useMemo(() => 
    images?.map(img => img.url || img) || [], 
    [images]
  )

  // Preload upcoming images based on current page
  useImagePreloader(imageUrls, currentPage - 1)

  // Also preload on initial load (first 10 images)
  useEffect(() => {
    if (imageUrls.length > 0) {
      preloadImages(imageUrls.slice(0, 10), true)
    }
  }, [imageUrls])

  // Resume từ saved progress
  useEffect(() => {
    const saved = getComicProgress(comicId)
    if (saved && saved.chapter === chapter) {
      setCurrentPage(saved.page)
    }
  }, [comicId, chapter, getComicProgress])

  // Scroll to initial page on mount (for scroll mode)
  useEffect(() => {
    if (mode === 'scroll' && initialPage > 1 && containerRef.current) {
      // Wait for images to render
      const timer = setTimeout(() => {
        const targetEl = containerRef.current?.querySelector(`[data-page="${initialPage}"]`)
        if (targetEl) {
          targetEl.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      }, 300)
      return () => clearTimeout(timer)
    }
  }, [mode, initialPage])

  // Save progress khi đổi trang
  useEffect(() => {
    saveComicProgress(comicId, chapter, currentPage)
    onPageChange?.(currentPage)
  }, [currentPage, comicId, chapter, saveComicProgress, onPageChange])

  // Navigation
  const goToPage = useCallback((page) => {
    const newPage = Math.max(1, Math.min(images.length, page))
    setCurrentPage(newPage)
  }, [images.length])

  const nextPage = () => goToPage(currentPage + 1)
  const prevPage = () => goToPage(currentPage - 1)

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (mode === 'page') {
        if (e.key === 'ArrowRight' || e.key === ' ') {
          nextPage()
          e.preventDefault()
        } else if (e.key === 'ArrowLeft') {
          prevPage()
          e.preventDefault()
        }
      }
      if (e.key === 'f') {
        toggleFullscreen()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [mode, currentPage])

  // Touch swipe for mobile (page mode)
  const [touchStart, setTouchStart] = useState(null)
  
  const handleTouchStart = (e) => {
    setTouchStart(e.touches[0].clientX)
  }

  const handleTouchEnd = (e) => {
    if (!touchStart || mode !== 'page') return
    
    const touchEnd = e.changedTouches[0].clientX
    const diff = touchStart - touchEnd

    if (Math.abs(diff) > 50) {
      if (diff > 0) nextPage()
      else prevPage()
    }
    
    setTouchStart(null)
  }

  // Scroll tracking for scroll mode - update current page based on visible image
  useEffect(() => {
    if (mode !== 'scroll' || !containerRef.current) return

    const imageElements = containerRef.current.querySelectorAll('[data-page]')
    if (!imageElements.length) return

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && entry.intersectionRatio > 0.5) {
            const page = parseInt(entry.target.dataset.page)
            if (page && page !== currentPage) {
              setCurrentPage(page)
            }
          }
        })
      },
      {
        root: null,
        rootMargin: '-40% 0px -40% 0px',
        threshold: 0.5,
      }
    )

    imageElements.forEach((el) => observer.observe(el))

    return () => observer.disconnect()
  }, [mode, images.length])

  // Fullscreen toggle
  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      containerRef.current?.requestFullscreen()
      setFullscreen(true)
    } else {
      document.exitFullscreen()
      setFullscreen(false)
    }
  }

  // Fit mode classes
  const getFitClass = () => {
    switch (fitMode) {
      case 'width':
        return 'w-full h-auto'
      case 'height':
        return 'h-screen w-auto'
      case 'original':
        return ''
      default:
        return 'w-full h-auto max-w-4xl'
    }
  }

  // Scroll mode - render all images
  if (mode === 'scroll') {
    return (
      <div 
        ref={containerRef}
        className="relative bg-gray-900"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
      >
        {/* Controls bar */}
        <div className={`sticky top-16 z-10 bg-gray-900/95 backdrop-blur-sm p-2 flex items-center justify-between transition-opacity ${showControls ? 'opacity-100' : 'opacity-0'}`}>
          <div className="flex items-center space-x-2">
            <span className="text-white text-sm">
              Trang {currentPage} / {images.length}
            </span>
          </div>
          
          <div className="flex items-center space-x-2">
            <button 
              onClick={() => setShowSettings(!showSettings)}
              className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800"
            >
              <Cog6ToothIcon className="w-5 h-5" />
            </button>
            <button 
              onClick={toggleFullscreen}
              className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800"
            >
              {fullscreen ? (
                <ArrowsPointingInIcon className="w-5 h-5" />
              ) : (
                <ArrowsPointingOutIcon className="w-5 h-5" />
              )}
            </button>
          </div>
        </div>

        {/* Settings panel */}
        {showSettings && (
          <div className="sticky top-28 z-10 mx-auto max-w-xs bg-gray-800 rounded-lg shadow-lg p-4 mb-4">
            <h4 className="text-white font-medium mb-3">Cài đặt đọc</h4>
            
            <div className="space-y-3">
              <div>
                <label className="text-sm text-gray-400">Chế độ</label>
                <div className="flex space-x-2 mt-1">
                  <button 
                    onClick={() => updateReaderSettings({ mode: 'scroll' })}
                    className={`px-3 py-1 rounded text-sm ${mode === 'scroll' ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300'}`}
                  >
                    Cuộn
                  </button>
                  <button 
                    onClick={() => updateReaderSettings({ mode: 'page' })}
                    className={`px-3 py-1 rounded text-sm ${mode === 'page' ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300'}`}
                  >
                    Từng trang
                  </button>
                </div>
              </div>
              
              <div>
                <label className="text-sm text-gray-400">Fit ảnh</label>
                <div className="flex space-x-2 mt-1">
                  {['width', 'height', 'original'].map((fit) => (
                    <button 
                      key={fit}
                      onClick={() => updateReaderSettings({ fitMode: fit })}
                      className={`px-3 py-1 rounded text-sm capitalize ${fitMode === fit ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300'}`}
                    >
                      {fit === 'width' ? 'Full width' : fit === 'height' ? 'Full height' : 'Original'}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Images (scroll mode) */}
        <div className="flex flex-col items-center">
          {images.map((image, index) => (
            <div 
              key={index}
              className="relative min-h-[200px]"
              data-page={index + 1}
            >
              <OptimizedImage
                src={image.url}
                alt={`Trang ${index + 1}`}
                className={`${getFitClass()} mx-auto`}
                wrapperClassName="w-full"
                priority={index < 3} // Load first 3 immediately
                onLoad={() => {
                  setImageLoaded({ ...imageLoaded, [index]: true })
                  // Update current page based on scroll position
                  if (containerRef.current) {
                    const rect = containerRef.current.querySelector(`[data-page="${index + 1}"]`)?.getBoundingClientRect()
                    if (rect && rect.top >= 0 && rect.top < window.innerHeight / 2) {
                      setCurrentPage(index + 1)
                    }
                  }
                }}
              />
            </div>
          ))}
        </div>
      </div>
    )
  }

  // Page mode - render one image at a time
  const currentImage = images[currentPage - 1]

  return (
    <div 
      ref={containerRef}
      className="relative bg-gray-900 min-h-[80vh] select-none"
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      onClick={() => setShowControls(!showControls)}
    >
      {/* Current image */}
      <div className="flex items-center justify-center min-h-[80vh]">
        {currentImage && (
          <OptimizedImage
            src={currentImage.url}
            alt={`Trang ${currentPage}`}
            className={`${getFitClass()} mx-auto`}
            wrapperClassName="flex items-center justify-center"
            priority={true}
          />
        )}
      </div>

      {/* Navigation buttons */}
      <button 
        onClick={(e) => { e.stopPropagation(); prevPage(); }}
        disabled={currentPage <= 1}
        className={`absolute left-2 top-1/2 -translate-y-1/2 p-3 bg-black/50 rounded-full text-white transition-opacity ${
          showControls ? 'opacity-100' : 'opacity-0'
        } ${currentPage <= 1 ? 'opacity-30 cursor-not-allowed' : 'hover:bg-black/70'}`}
      >
        <ChevronLeftIcon className="w-6 h-6" />
      </button>

      <button 
        onClick={(e) => { e.stopPropagation(); nextPage(); }}
        disabled={currentPage >= images.length}
        className={`absolute right-2 top-1/2 -translate-y-1/2 p-3 bg-black/50 rounded-full text-white transition-opacity ${
          showControls ? 'opacity-100' : 'opacity-0'
        } ${currentPage >= images.length ? 'opacity-30 cursor-not-allowed' : 'hover:bg-black/70'}`}
      >
        <ChevronRightIcon className="w-6 h-6" />
      </button>

      {/* Bottom controls */}
      <div className={`absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/90 to-transparent p-4 transition-opacity ${showControls ? 'opacity-100' : 'opacity-0'}`}>
        {/* Page slider */}
        <div className="mb-3">
          <input
            type="range"
            min="1"
            max={images.length}
            value={currentPage}
            onChange={(e) => goToPage(parseInt(e.target.value))}
            className="w-full accent-primary-500"
            onClick={(e) => e.stopPropagation()}
          />
        </div>

        <div className="flex items-center justify-between">
          <span className="text-white">
            Trang {currentPage} / {images.length}
          </span>
          
          <div className="flex items-center space-x-2">
            <button 
              onClick={(e) => { e.stopPropagation(); setShowSettings(!showSettings); }}
              className="p-2 text-gray-400 hover:text-white"
            >
              <Cog6ToothIcon className="w-5 h-5" />
            </button>
            <button 
              onClick={(e) => { e.stopPropagation(); toggleFullscreen(); }}
              className="p-2 text-gray-400 hover:text-white"
            >
              {fullscreen ? (
                <ArrowsPointingInIcon className="w-5 h-5" />
              ) : (
                <ArrowsPointingOutIcon className="w-5 h-5" />
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Settings panel */}
      {showSettings && (
        <div 
          className="absolute bottom-20 right-4 bg-gray-800 rounded-lg shadow-lg p-4 min-w-[200px]"
          onClick={(e) => e.stopPropagation()}
        >
          <h4 className="text-white font-medium mb-3">Cài đặt đọc</h4>
          
          <div className="space-y-3">
            <div>
              <label className="text-sm text-gray-400">Chế độ</label>
              <div className="flex space-x-2 mt-1">
                <button 
                  onClick={() => updateReaderSettings({ mode: 'scroll' })}
                  className={`px-3 py-1 rounded text-sm ${mode === 'scroll' ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300'}`}
                >
                  Cuộn
                </button>
                <button 
                  onClick={() => updateReaderSettings({ mode: 'page' })}
                  className={`px-3 py-1 rounded text-sm ${mode === 'page' ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300'}`}
                >
                  Từng trang
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default ImageGallery
