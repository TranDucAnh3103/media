import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Store quản lý player state (resume playback, volume, etc.)
export const usePlayerStore = create(
  persist(
    (set, get) => ({
      // Video playback state
      videoProgress: {}, // { videoId: { currentTime, duration } }
      volume: 1,
      muted: false,
      playbackRate: 1,

      // Comic reader state
      comicProgress: {}, // { comicId: { chapter, page } }
      readerSettings: {
        mode: 'scroll', // scroll, page
        direction: 'vertical', // vertical, horizontal
        fitMode: 'width', // width, height, original
      },

      // Save video progress
      saveVideoProgress: (videoId, currentTime, duration) => {
        set((state) => ({
          videoProgress: {
            ...state.videoProgress,
            [videoId]: { currentTime, duration, updatedAt: Date.now() }
          }
        }))
      },

      // Get video progress
      getVideoProgress: (videoId) => {
        return get().videoProgress[videoId] || null
      },

      // Save comic progress
      saveComicProgress: (comicId, chapter, page) => {
        set((state) => ({
          comicProgress: {
            ...state.comicProgress,
            [comicId]: { chapter, page, updatedAt: Date.now() }
          }
        }))
      },

      // Get comic progress
      getComicProgress: (comicId) => {
        return get().comicProgress[comicId] || null
      },

      // Set volume
      setVolume: (volume) => set({ volume }),

      // Toggle mute
      toggleMute: () => set((state) => ({ muted: !state.muted })),

      // Set playback rate
      setPlaybackRate: (rate) => set({ playbackRate: rate }),

      // Update reader settings
      updateReaderSettings: (settings) => {
        set((state) => ({
          readerSettings: { ...state.readerSettings, ...settings }
        }))
      },

      // Clear old progress (older than 30 days)
      clearOldProgress: () => {
        const thirtyDaysAgo = Date.now() - 30 * 24 * 60 * 60 * 1000
        
        set((state) => ({
          videoProgress: Object.fromEntries(
            Object.entries(state.videoProgress).filter(
              ([_, value]) => value.updatedAt > thirtyDaysAgo
            )
          ),
          comicProgress: Object.fromEntries(
            Object.entries(state.comicProgress).filter(
              ([_, value]) => value.updatedAt > thirtyDaysAgo
            )
          ),
        }))
      },
    }),
    {
      name: 'player-storage',
    }
  )
)
