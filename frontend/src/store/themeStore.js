import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Store quản lý theme (dark/light mode)
export const useThemeStore = create(
  persist(
    (set, get) => ({
      darkMode: true, // Mặc định dark mode

      // Toggle theme
      toggleTheme: () => {
        set((state) => ({ darkMode: !state.darkMode }))
      },

      // Set specific theme
      setDarkMode: (value) => {
        set({ darkMode: value })
      },

      // Initialize theme from system preference
      initTheme: () => {
        const stored = get().darkMode
        if (stored === null) {
          // Check system preference
          const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
          set({ darkMode: prefersDark })
        }
      },
    }),
    {
      name: 'theme-storage',
    }
  )
)
