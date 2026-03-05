import { useState, useEffect, useRef, memo } from 'react'
import { useCachedImage } from '../hooks/useImageCache'

// Optimized image component with:
// - Intersection Observer for lazy loading
// - IndexedDB caching (background, non-blocking)
// - Loading skeleton
// - Error fallback

const OptimizedImage = memo(({
  src,
  alt,
  className = '',
  wrapperClassName = '',
  placeholder = null,
  onLoad,
  onError,
  priority = false, // Load immediately without lazy loading
  ...props
}) => {
  const imgRef = useRef(null)
  const [isInView, setIsInView] = useState(priority)
  const [isImgLoaded, setIsImgLoaded] = useState(false)
  const [hasError, setHasError] = useState(false)
  
  // Use cached image source (non-blocking - returns original URL immediately)
  const { imageSrc } = useCachedImage(isInView ? src : null)

  // Intersection Observer for lazy loading
  useEffect(() => {
    if (priority || !imgRef.current) return
    
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsInView(true)
            observer.unobserve(entry.target)
          }
        })
      },
      {
        rootMargin: '200px', // Start loading 200px before visible
        threshold: 0.01,
      }
    )
    
    observer.observe(imgRef.current)
    
    return () => observer.disconnect()
  }, [priority])

  // Handle image load
  const handleLoad = () => {
    setIsImgLoaded(true)
    onLoad?.()
  }

  // Handle error
  const handleError = () => {
    setHasError(true)
    onError?.()
  }

  // Show skeleton when image not loaded yet
  const showSkeleton = !isImgLoaded && !hasError

  return (
    <div 
      ref={imgRef}
      className={`relative overflow-hidden ${wrapperClassName}`}
    >
      {/* Skeleton/Placeholder */}
      {showSkeleton && (
        <div className="absolute inset-0 bg-gray-700 animate-pulse flex items-center justify-center">
          <div className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
        </div>
      )}

      {/* Error state */}
      {hasError && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-800">
          <div className="text-center text-gray-500">
            <svg className="w-8 h-8 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
            <span className="text-sm">Lỗi tải ảnh</span>
          </div>
        </div>
      )}

      {/* Main image - always render if we have src */}
      {(isInView || priority) && (imageSrc || src) && (
        <img
          src={imageSrc || src}
          alt={alt}
          className={`transition-opacity duration-300 ${
            isImgLoaded ? 'opacity-100' : 'opacity-0'
          } ${className}`}
          onLoad={handleLoad}
          onError={handleError}
          loading={priority ? 'eager' : 'lazy'}
          decoding="async"
          {...props}
        />
      )}
    </div>
  )
})

OptimizedImage.displayName = 'OptimizedImage'

export default OptimizedImage
