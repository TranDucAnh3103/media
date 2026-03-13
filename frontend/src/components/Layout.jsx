import { Outlet } from 'react-router-dom'
import { useEffect } from 'react'
import { useAuthStore } from '../store/authStore'
import Navbar from './Navbar'
import BottomNav from './BottomNav'

// Layout component chính cho toàn bộ app
const Layout = () => {
  const { initAuth } = useAuthStore()

  useEffect(() => {
    // Khởi tạo auth từ localStorage khi app load
    initAuth()
  }, [initAuth])

  return (
    <div className="min-h-screen bg-gray-950">
      {/* Desktop navbar - ẩn trên mobile */}
      <Navbar />

      {/* Main content */}
      <main className="pt-0 md:pt-16 pb-20 md:pb-0 min-h-screen">
        <Outlet />
      </main>

      {/* Footer - ẩn trên mobile để không chồng với bottom nav */}
      <footer className="hidden md:block bg-gray-900 border-t border-white/5 py-8">
        <div className="container-custom text-center">
          <p className="text-gray-500 text-sm">
            &copy; 2026 MediaHub. Persional website.
          </p>
        </div>
      </footer>

      {/* Mobile bottom navigation - luôn hiển thị */}
      <BottomNav />
    </div>
  )
}

export default Layout
