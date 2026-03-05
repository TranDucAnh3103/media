import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import api from '../services/api'

// Store quản lý authentication state
export const useAuthStore = create(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isLoading: false,
      error: null,

      // Login
      login: async (email, password) => {
        set({ isLoading: true, error: null })
        try {
          const response = await api.post('/auth/login', { email, password })
          const { token, user } = response.data
          
          // Lưu token vào localStorage và set header
          localStorage.setItem('token', token)
          api.defaults.headers.common['Authorization'] = `Bearer ${token}`
          
          set({ user, token, isLoading: false })
          return { success: true }
        } catch (error) {
          const message = error.response?.data?.error || 'Login failed'
          set({ error: message, isLoading: false })
          return { success: false, error: message }
        }
      },

      // Register
      register: async (username, email, password) => {
        set({ isLoading: true, error: null })
        try {
          const response = await api.post('/auth/register', { username, email, password })
          const { token, user } = response.data
          
          localStorage.setItem('token', token)
          api.defaults.headers.common['Authorization'] = `Bearer ${token}`
          
          set({ user, token, isLoading: false })
          return { success: true }
        } catch (error) {
          const message = error.response?.data?.error || 'Registration failed'
          set({ error: message, isLoading: false })
          return { success: false, error: message }
        }
      },

      // Logout
      logout: () => {
        localStorage.removeItem('token')
        delete api.defaults.headers.common['Authorization']
        set({ user: null, token: null })
      },

      // Fetch current user profile
      fetchProfile: async () => {
        const token = get().token || localStorage.getItem('token')
        if (!token) return

        api.defaults.headers.common['Authorization'] = `Bearer ${token}`
        
        try {
          const response = await api.get('/user/profile')
          set({ user: response.data, token })
        } catch (error) {
          // Token expired or invalid
          get().logout()
        }
      },

      // Initialize auth from localStorage
      initAuth: () => {
        const token = localStorage.getItem('token')
        if (token) {
          api.defaults.headers.common['Authorization'] = `Bearer ${token}`
          set({ token })
          get().fetchProfile()
        }
      },

      // Check if user is admin
      isAdmin: () => {
        const user = get().user
        return user?.role === 'admin'
      },

      // Clear error
      clearError: () => set({ error: null }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ token: state.token }),
    }
  )
)
