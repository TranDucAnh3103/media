import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { UserIcon, EnvelopeIcon, LockClosedIcon } from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { useAuthStore } from '../store/authStore'

const Register = () => {
  const navigate = useNavigate()
  const { register, isLoading } = useAuthStore()
  
  const [form, setForm] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  })
  const [errors, setErrors] = useState({})

  // Validate form
  const validate = () => {
    const newErrors = {}
    
    if (!form.username) {
      newErrors.username = 'Vui lòng nhập tên đăng nhập'
    } else if (form.username.length < 3) {
      newErrors.username = 'Tên đăng nhập phải có ít nhất 3 ký tự'
    } else if (!/^[a-zA-Z0-9_]+$/.test(form.username)) {
      newErrors.username = 'Tên đăng nhập chỉ chứa chữ, số và dấu gạch dưới'
    }

    if (!form.email) {
      newErrors.email = 'Vui lòng nhập email'
    } else if (!/\S+@\S+\.\S+/.test(form.email)) {
      newErrors.email = 'Email không hợp lệ'
    }

    if (!form.password) {
      newErrors.password = 'Vui lòng nhập mật khẩu'
    } else if (form.password.length < 6) {
      newErrors.password = 'Mật khẩu phải có ít nhất 6 ký tự'
    }

    if (!form.confirmPassword) {
      newErrors.confirmPassword = 'Vui lòng xác nhận mật khẩu'
    } else if (form.password !== form.confirmPassword) {
      newErrors.confirmPassword = 'Mật khẩu không khớp'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  // Handle submit
  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!validate()) return

    const result = await register(form.username, form.email, form.password)
    if (result.success) {
      toast.success('Đăng ký thành công!')
      navigate('/')
    } else {
      toast.error(result.error || 'Đăng ký thất bại')
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 py-8">
      <div className="w-full max-w-md">
        {/* Logo/Header */}
        <div className="text-center mb-8">
          <Link to="/" className="text-3xl font-bold text-primary-500">
            MediaHub
          </Link>
          <h1 className="text-2xl font-bold text-white mt-4">Đăng ký</h1>
          <p className="text-gray-400 mt-2">Tạo tài khoản mới</p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="bg-gray-800 rounded-xl p-6 shadow-xl">
          {/* Username */}
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Tên đăng nhập</label>
            <div className="relative">
              <UserIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
              <input
                type="text"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                className={`input pl-10 ${errors.username ? 'border-red-500' : ''}`}
                placeholder="username"
                autoComplete="username"
              />
            </div>
            {errors.username && (
              <p className="text-red-400 text-sm mt-1">{errors.username}</p>
            )}
          </div>

          {/* Email */}
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Email</label>
            <div className="relative">
              <EnvelopeIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
              <input
                type="email"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                className={`input pl-10 ${errors.email ? 'border-red-500' : ''}`}
                placeholder="email@example.com"
                autoComplete="email"
              />
            </div>
            {errors.email && (
              <p className="text-red-400 text-sm mt-1">{errors.email}</p>
            )}
          </div>

          {/* Password */}
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Mật khẩu</label>
            <div className="relative">
              <LockClosedIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
              <input
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                className={`input pl-10 ${errors.password ? 'border-red-500' : ''}`}
                placeholder="••••••••"
                autoComplete="new-password"
              />
            </div>
            {errors.password && (
              <p className="text-red-400 text-sm mt-1">{errors.password}</p>
            )}
          </div>

          {/* Confirm Password */}
          <div className="mb-6">
            <label className="block text-sm text-gray-400 mb-1">Xác nhận mật khẩu</label>
            <div className="relative">
              <LockClosedIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
              <input
                type="password"
                value={form.confirmPassword}
                onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}
                className={`input pl-10 ${errors.confirmPassword ? 'border-red-500' : ''}`}
                placeholder="••••••••"
                autoComplete="new-password"
              />
            </div>
            {errors.confirmPassword && (
              <p className="text-red-400 text-sm mt-1">{errors.confirmPassword}</p>
            )}
          </div>

          {/* Terms */}
          <div className="mb-6">
            <label className="flex items-start gap-2 text-sm text-gray-400">
              <input
                type="checkbox"
                className="mt-1 w-4 h-4 rounded border-gray-600 bg-gray-700 text-primary-500 focus:ring-primary-500"
                required
              />
              <span>
                Tôi đồng ý với{' '}
                <Link to="/terms" className="text-primary-400 hover:text-primary-300">
                  Điều khoản sử dụng
                </Link>
                {' '}và{' '}
                <Link to="/privacy" className="text-primary-400 hover:text-primary-300">
                  Chính sách bảo mật
                </Link>
              </span>
            </label>
          </div>

          {/* Submit */}
          <button
            type="submit"
            disabled={isLoading}
            className="btn-primary w-full disabled:opacity-50"
          >
            {isLoading ? 'Đang đăng ký...' : 'Đăng ký'}
          </button>

          {/* Login link */}
          <p className="text-center text-gray-400 mt-6">
            Đã có tài khoản?{' '}
            <Link to="/login" className="text-primary-400 hover:text-primary-300">
              Đăng nhập
            </Link>
          </p>
        </form>
      </div>
    </div>
  )
}

export default Register
