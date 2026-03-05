import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'

// Component bảo vệ routes yêu cầu đăng nhập
const ProtectedRoute = ({ requireAdmin = false }) => {
  const { user, token } = useAuthStore()

  // Chưa đăng nhập
  if (!token) {
    return <Navigate to="/login" replace />
  }

  // Yêu cầu admin nhưng không phải admin
  if (requireAdmin && user?.role !== 'admin') {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}

export default ProtectedRoute
