import { useState, useEffect, useCallback, useRef } from 'react'

// IndexedDB config
const DB_NAME = 'MediaHubImageCache'
const DB_VERSION = 1
const STORE_NAME = 'images'
const MAX_CACHE_SIZE = 100 // Maximum cached images
const CACHE_EXPIRY = 7 * 24 * 60 * 60 * 1000 // 7 days

// Open IndexedDB connection
const openDB = () => {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    
    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve(request.result)
    
    request.onupgradeneeded = (e) => {
      const db = e.target.result
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        const store = db.createObjectStore(STORE_NAME, { keyPath: 'url' })
        store.createIndex('timestamp', 'timestamp', { unique: false })
      }
    }
  })
}

// Get cached image from IndexedDB
const getCachedImage = async (url) => {
  try {
    const db = await openDB()
    return new Promise((resolve) => {
      const tx = db.transaction(STORE_NAME, 'readonly')
      const store = tx.objectStore(STORE_NAME)
      const request = store.get(url)
      
      request.onsuccess = () => {
        const result = request.result
        if (result && Date.now() - result.timestamp < CACHE_EXPIRY) {
          resolve(result.blob)
        } else {
          resolve(null)
        }
      }
      request.onerror = () => resolve(null)
    })
  } catch {
    return null
  }
}

// Store image in IndexedDB
const cacheImage = async (url, blob) => {
  try {
    const db = await openDB()
    const tx = db.transaction(STORE_NAME, 'readwrite')
    const store = tx.objectStore(STORE_NAME)
    
    store.put({
      url,
      blob,
      timestamp: Date.now()
    })
    
    // Cleanup old entries if over limit
    const countRequest = store.count()
    countRequest.onsuccess = () => {
      if (countRequest.result > MAX_CACHE_SIZE) {
        cleanupOldCache(store)
      }
    }
  } catch (e) {
    console.warn('Failed to cache image:', e)
  }
}

// Clean up oldest cached images
const cleanupOldCache = async (store) => {
  const index = store.index('timestamp')
  const request = index.openCursor()
  let deleteCount = 0
  const deleteTarget = 20 // Delete oldest 20 entries
  
  request.onsuccess = (e) => {
    const cursor = e.target.result
    if (cursor && deleteCount < deleteTarget) {
      cursor.delete()
      deleteCount++
      cursor.continue()
    }
  }
}

// Preload single image and cache it
export const preloadImage = (url) => {
  return new Promise(async (resolve) => {
    // Check cache first
    const cached = await getCachedImage(url)
    if (cached) {
      resolve(URL.createObjectURL(cached))
      return
    }
    
    // Fetch and cache
    try {
      const response = await fetch(url)
      const blob = await response.blob()
      await cacheImage(url, blob)
      resolve(URL.createObjectURL(blob))
    } catch {
      resolve(url) // Fallback to original URL
    }
  })
}

// Preload multiple images in background
export const preloadImages = (urls, priority = false) => {
  const preloadNext = (url) => {
    return new Promise((resolve) => {
      const img = new Image()
      img.onload = () => resolve()
      img.onerror = () => resolve()
      img.src = url
      
      // For high priority, also cache in IndexedDB
      if (priority) {
        fetch(url)
          .then(res => res.blob())
          .then(blob => cacheImage(url, blob))
          .catch(() => {})
      }
    })
  }
  
  // Load in batches of 3
  const batchSize = 3
  let index = 0
  
  const loadBatch = async () => {
    const batch = urls.slice(index, index + batchSize)
    if (batch.length === 0) return
    
    await Promise.all(batch.map(preloadNext))
    index += batchSize
    
    // Continue loading more in background
    if (index < urls.length) {
      requestIdleCallback ? requestIdleCallback(loadBatch) : setTimeout(loadBatch, 100)
    }
  }
  
  loadBatch()
}

// Custom hook for single image with caching
export const useCachedImage = (url) => {
  const [imageSrc, setImageSrc] = useState(url) // Start with original URL
  const [isLoading, setIsLoading] = useState(false)
  const [hasError, setHasError] = useState(false)
  
  useEffect(() => {
    if (!url) {
      setImageSrc(null)
      setIsLoading(false)
      return
    }

    // Always set the original URL first for immediate display
    setImageSrc(url)
    
    // Then try to load from cache in background (optional enhancement)
    let isMounted = true
    
    const tryCache = async () => {
      try {
        const cached = await getCachedImage(url)
        if (cached && isMounted) {
          const blobUrl = URL.createObjectURL(cached)
          setImageSrc(blobUrl)
        } else if (isMounted) {
          // Try to cache for next time (don't block rendering)
          fetch(url)
            .then(res => res.blob())
            .then(blob => cacheImage(url, blob))
            .catch(() => {}) // Ignore cache errors
        }
      } catch {
        // Ignore errors, original URL is already set
      }
      if (isMounted) setIsLoading(false)
    }
    
    tryCache()
    
    return () => {
      isMounted = false
    }
  }, [url])
  
  return { imageSrc: imageSrc || url, isLoading, hasError }
}

// Hook for preloading upcoming images
export const useImagePreloader = (images, currentIndex = 0) => {
  const preloadedRef = useRef(new Set())
  
  useEffect(() => {
    if (!images?.length) return
    
    // Preload next 5 images
    const preloadCount = 5
    const startIndex = currentIndex
    const endIndex = Math.min(currentIndex + preloadCount, images.length)
    
    const urlsToPreload = images
      .slice(startIndex, endIndex)
      .map(img => img.url || img)
      .filter(url => !preloadedRef.current.has(url))
    
    if (urlsToPreload.length > 0) {
      preloadImages(urlsToPreload, true)
      urlsToPreload.forEach(url => preloadedRef.current.add(url))
    }
  }, [images, currentIndex])
}

// Clear all cached images
export const clearImageCache = async () => {
  try {
    const db = await openDB()
    const tx = db.transaction(STORE_NAME, 'readwrite')
    tx.objectStore(STORE_NAME).clear()
  } catch (e) {
    console.warn('Failed to clear image cache:', e)
  }
}

export default {
  preloadImage,
  preloadImages,
  useCachedImage,
  useImagePreloader,
  clearImageCache,
}
