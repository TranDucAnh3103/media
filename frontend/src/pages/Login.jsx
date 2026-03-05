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
        <form onSubmit={handleSubmit} className="bg-gray-800 rounded-xl p-6 shadow-xl">
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
                className={`input pl-10 ${errors.password ? 'border-red-500' : ''}`}
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
            className="btn-primary w-full disabled:opacity-50"
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
          <div className="mt-6 pt-6 border-t border-gray-700">
            <p className="text-center text-gray-500 text-sm mb-3">Đăng nhập nhanh (Dev)</p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => quickLogin('admin')}
                disabled={isLoading}
                className="flex-1 flex items-center justify-center gap-2 py-2 px-4 bg-red-600 hover:bg-red-700 rounded-lg text-white transition-colors disabled:opacity-50"
              >
                <ShieldCheckIcon className="w-5 h-5" />
                Admin
              </button>
              <button
                type="button"
                onClick={() => quickLogin('demo')}
                disabled={isLoading}
                className="flex-1 flex items-center justify-center gap-2 py-2 px-4 bg-blue-600 hover:bg-blue-700 rounded-lg text-white transition-colors disabled:opacity-50"
              >
                <UserIcon className="w-5 h-5" />
                Demo User
              </button>
            </div>
          </div>
        </form>

        {/* Social login (placeholder) */}
        <div className="mt-6">
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-gray-700"></div>
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-gray-900 text-gray-500">Hoặc đăng nhập với</span>
            </div>
          </div>

          <div className="mt-4 flex gap-3">
            <button className="flex-1 flex items-center justify-center gap-2 py-2 px-4 border border-gray-700 rounded-lg text-gray-400 hover:bg-gray-800 transition-colors">
              <svg className="w-5 h-5" viewBox="0 0 24 24">
                <path fill="currentColor" d="M12.545,10.239v3.821h5.445c-0.712,2.315-2.647,3.972-5.445,3.972c-3.332,0-6.033-2.701-6.033-6.032 s2.701-6.032,6.033-6.032c1.498,0,2.866,0.549,3.921,1.453l2.814-2.814C17.503,2.988,15.139,2,12.545,2 C7.021,2,2.543,6.477,2.543,12s4.478,10,10.002,10c8.396,0,10.249-7.85,9.426-11.748L12.545,10.239z"/>
              </svg>
              Google
            </button>
            <button className="flex-1 flex items-center justify-center gap-2 py-2 px-4 border border-gray-700 rounded-lg text-gray-400 hover:bg-gray-800 transition-colors">
              <svg className="w-5 h-5" viewBox="0 0 24 24">
                <path fill="currentColor" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.603-3.369-1.342-3.369-1.342-.454-1.155-1.11-1.462-1.11-1.462-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.579.688.481C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z"/>
              </svg>
              GitHub
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Login
