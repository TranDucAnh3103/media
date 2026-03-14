import { useState } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { EnvelopeIcon, LockClosedIcon, UserIcon, ShieldCheckIcon } from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { useAuthStore } from '../store/authStore'

const Login = () => {
  const navigate = useNavigate()
  const location = useLocation()
  const { login, isLoading } = useAuthStore()
  
  const [form, setForm] = useState({
    email: '',
    password: '',
  })
  const [errors, setErrors] = useState({})

  // Get redirect path
  const from = location.state?.from?.pathname || '/'

  // Quick login với tài khoản có sẵn
  const quickLogin = async (type) => {
    const accounts = {
      admin: { email: 'admin@mediahub.com', password: 'admin123456' },
      demo: { email: 'demo@mediahub.com', password: 'demo123456' },
    }
    const account = accounts[type]
    if (!account) return

    const result = await login(account.email, account.password)
    if (result.success) {
      toast.success(`Đăng nhập ${type === 'admin' ? 'Admin' : 'Demo'} thành công`)
      navigate(from, { replace: true })
    } else {
      toast.error(result.error || 'Đăng nhập thất bại')
    }
  }

  // Validate form
  const validate = () => {
    const newErrors = {}
    if (!form.email) {
      newErrors.email = 'Vui lòng nhập email'
    } else if (!/\S+@\S+\.\S+/.test(form.email)) {
      newErrors.email = 'Email không hợp lệ'
    }
    if (!form.password) {
      newErrors.password = 'Vui lòng nhập mật khẩu'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  // Handle submit
  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!validate()) return

    const result = await login(form.email, form.password)
    if (result.success) {
      toast.success('Đăng nhập thành công')
      navigate(from, { replace: true })
    } else {
      toast.error(result.error || 'Đăng nhập thất bại')
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        {/* Logo/Header */}
        <div className="text-center mb-8">
          <Link to="/" className="text-3xl font-bold text-primary-500">
            MediaHub
          </Link>
          <h1 className="text-2xl font-bold text-white mt-4">Đăng nhập</h1>
          <p className="text-gray-400 mt-2">Đăng nhập để tiếp tục</p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="bg-white/[0.03] backdrop-blur-xl rounded-3xl p-6 md:p-8 shadow-2xl border border-white/5 relative z-10">
          {/* Email */}
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Email</label>
            <div className="relative">
              <EnvelopeIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
              <input
                type="email"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                className={`input pl-11 h-12 rounded-xl bg-white/5 border border-white/10 text-white placeholder-gray-500 focus:bg-white/10 focus:border-violet-500/50 transition-all ${errors.email ? 'border-red-500/50 focus:border-red-500' : ''}`}
                placeholder="demo@mediahub.com"
                autoComplete="email"
              />
            </div>
            {errors.email && (
              <p className="text-red-400 text-sm mt-1">{errors.email}</p>
            )}
          </div>

          {/* Password */}
          <div className="mb-6">
            <label className="block text-sm text-gray-400 mb-1">Mật khẩu</label>
            <div className="relative">
              <LockClosedIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
              <input
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                className={`input pl-11 h-12 rounded-xl bg-white/5 border border-white/10 text-white placeholder-gray-500 focus:bg-white/10 focus:border-violet-500/50 transition-all ${errors.password ? 'border-red-500/50 focus:border-red-500' : ''}`}
                placeholder="••••••••"
                autoComplete="current-password"
              />
            </div>
            {errors.password && (
              <p className="text-red-400 text-sm mt-1">{errors.password}</p>
            )}
          </div>

          {/* Forgot password */}
          <div className="flex justify-end mb-6">
            <Link to="/forgot-password" className="text-sm text-primary-400 hover:text-primary-300">
              Quên mật khẩu?
            </Link>
          </div>

          {/* Submit */}
          <button
            type="submit"
            disabled={isLoading}
            className="w-full flex justify-center py-3.5 px-4 rounded-xl text-sm font-semibold text-white bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:scale-[1.02] hover:shadow-[0_4px_20px_rgba(139,92,246,0.4)] transition-all duration-300 disabled:opacity-50 disabled:hover:scale-100"
          >
            {isLoading ? 'Đang đăng nhập...' : 'Đăng nhập'}
          </button>

          {/* Register link */}
          <p className="text-center text-gray-400 mt-6">
            Chưa có tài khoản?{' '}
            <Link to="/register" className="text-primary-400 hover:text-primary-300">
              Đăng ký ngay
            </Link>
          </p>

          {/* Quick login buttons */}
          <div className="mt-8 pt-6 border-t border-white/10">
            <p className="text-center text-gray-400 text-sm mb-4">Đăng nhập nhanh (Dev)</p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => quickLogin('admin')}
                disabled={isLoading}
                className="flex-1 flex items-center justify-center gap-2 py-2.5 px-4 bg-white/5 hover:bg-white/10 border border-white/10 rounded-xl text-white transition-all duration-300 hover:shadow-lg disabled:opacity-50"
              >
                <ShieldCheckIcon className="w-5 h-5 text-red-400" />
                <span className="font-medium">Admin</span>
              </button>
              <button
                type="button"
                onClick={() => quickLogin('demo')}
                disabled={isLoading}
                className="flex-1 flex items-center justify-center gap-2 py-2.5 px-4 bg-white/5 hover:bg-white/10 border border-white/10 rounded-xl text-white transition-all duration-300 hover:shadow-lg disabled:opacity-50"
              >
                <UserIcon className="w-5 h-5 text-blue-400" />
                <span className="font-medium">Demo</span>
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Login
