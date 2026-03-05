import axios from 'axios'

// Base URL - trong dev dùng proxy, production dùng env
const baseURL = import.meta.env.VITE_API_URL || '/api'

// Axios instance với base URL và interceptors
const api = axios.create({
  baseURL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor - thêm token vào header
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor - xử lý lỗi
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Handle 401 - Unauthorized
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      // Redirect to login if needed
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// ============ AUTH API ============
export const authAPI = {
  login: (email, password) => api.post('/auth/login', { email, password }),
  register: (data) => api.post('/auth/register', data),
  getProfile: () => api.get('/user/profile'),
  updateProfile: (data) => api.put('/user/profile', data),
}

// ============ VIDEOS API ============
export const videosAPI = {
  getAll: (params) => api.get('/videos', { params }),
  getById: (id) => api.get(`/videos/${id}`),
  getMyVideos: () => api.get('/videos/my'),
  getTrending: (limit = 10) => api.get('/videos/trending', { params: { limit } }),
  getLatest: (limit = 10) => api.get('/videos/latest', { params: { limit } }),
  upload: (formData, onProgress) => 
    api.post('/videos/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        onProgress?.(percent)
      },
    }),
  getUploadProgress: (id) => api.get(`/videos/upload/progress/${id}`),
  update: (id, data) => api.put(`/videos/${id}`, data),
  delete: (id) => api.delete(`/videos/${id}`),
  like: (id) => api.post(`/videos/${id}/like`),
  addComment: (id, content) => api.post(`/videos/${id}/comments`, { content }),
}

// ============ COMICS API ============
export const comicsAPI = {
  getAll: (params) => api.get('/comics', { params }),
  getById: (id) => api.get(`/comics/${id}`),
  getMyComics: () => api.get('/comics/my'),
  getTrending: (limit = 10) => api.get('/comics/trending', { params: { limit } }),
  getLatest: (limit = 10) => api.get('/comics/latest', { params: { limit } }),
  create: (data) => api.post('/comics', data),
  update: (id, data) => api.put(`/comics/${id}`, data),
  delete: (id) => api.delete(`/comics/${id}`),
  like: (id) => api.post(`/comics/${id}/like`),
  getChapter: (comicId, chapterNum) => api.get(`/comics/${comicId}/chapters/${chapterNum}`),
  uploadChapter: (comicId, formData, onProgress) =>
    api.post(`/comics/${comicId}/chapters`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        onProgress?.(percent)
      },
    }),
  // Upload truyện với ảnh (đơn giản hóa - không có chapter)
  uploadWithImages: (formData, onProgress) =>
    api.post('/comics/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000, // 5 phút cho upload ảnh
      onUploadProgress: (progressEvent) => {
        const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        onProgress?.(percent)
      },
    }),
}

// ============ USER API ============
export const userAPI = {
  getProfile: () => api.get('/user/profile'),
  updateProfile: (data) => api.put('/user/profile', data),
  addBookmark: (data) => api.post('/user/bookmarks', data),
  removeBookmark: (contentId) => api.delete(`/user/bookmarks/${contentId}`),
  getBookmarks: () => api.get('/user/bookmarks'),
  createPlaylist: (name) => api.post('/user/playlists', { name }),
  getPlaylists: () => api.get('/user/playlists'),
  addToPlaylist: (playlistId, videoId) => 
    api.post(`/user/playlists/${playlistId}/videos`, { video_id: videoId }),
  getHistory: () => api.get('/user/history'),
  clearHistory: () => api.delete('/user/history'),
  getLiked: () => api.get('/user/liked'),
}

// ============ ADMIN API ============
export const adminAPI = {
  getStats: () => api.get('/admin/stats'),
  getUsers: (params) => api.get('/admin/users', { params }),
  deleteUser: (id) => api.delete(`/admin/users/${id}`),
  banUser: (id) => api.post(`/admin/users/${id}/ban`),
  getVideos: (params) => api.get('/admin/videos', { params }),
  getComics: (params) => api.get('/admin/comics', { params }),
}

// ============ SYNC API ============
export const syncAPI = {
  syncVideos: () => api.post('/sync/videos', null, { timeout: 120000 }), // 2 min timeout
  syncComics: () => api.post('/sync/comics', null, { timeout: 120000 }), // 2 min timeout
}

export default api
