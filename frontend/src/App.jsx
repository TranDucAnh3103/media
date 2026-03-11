import { Routes, Route } from 'react-router-dom'
import { useEffect } from 'react'
import { useThemeStore } from './store/themeStore'

// Layout
import Layout from './components/Layout'

// Pages
import Home from './pages/Home'
import Videos from './pages/Videos'
import VideoDetail from './pages/VideoDetail'
import Comics from './pages/Comics'
import ComicDetail from './pages/ComicDetail'
import ComicReader from './pages/ComicReader'
import Upload from './pages/Upload'
import Login from './pages/Login'
import Register from './pages/Register'
import Profile from './pages/Profile'
import Dashboard from './pages/Dashboard'
import NotFound from './pages/NotFound'

// Protected Route
import ProtectedRoute from './components/ProtectedRoute'

function App() {
  const { darkMode, initTheme } = useThemeStore()

  useEffect(() => {
    initTheme()
  }, [initTheme])

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }, [darkMode])

  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        {/* Public routes */}
        <Route index element={<Home />} />
        <Route path="videos" element={<Videos />} />
        <Route path="videos/:id" element={<VideoDetail />} />
        <Route path="comics" element={<Comics />} />
        <Route path="comics/:id" element={<ComicDetail />} />
        <Route path="comics/:id/read/:chapter" element={<ComicReader />} />
        <Route path="login" element={<Login />} />
        <Route path="register" element={<Register />} />

        {/* Protected routes */}
        <Route element={<ProtectedRoute />}>
          <Route path="upload" element={<Upload />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="profile" element={<Profile />} />
        </Route>

        {/* 404 */}
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}

export default App
