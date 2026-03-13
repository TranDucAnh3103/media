import { Link, NavLink, useNavigate } from 'react-router-dom'
import { Fragment } from 'react'
import { Menu, Transition } from '@headlessui/react'
import {
  MagnifyingGlassIcon,
  BellIcon,
  UserCircleIcon,
  ArrowRightOnRectangleIcon,
  Cog6ToothIcon,
  CloudArrowUpIcon,
  SparklesIcon,
} from '@heroicons/react/24/outline'
import { useState } from 'react'
import { useAuthStore } from '../store/authStore'
import NotificationModal from './NotificationModal'

// Desktop navigation links
const navLinks = [
  { name: 'Home', href: '/' },
  { name: 'Videos', href: '/videos' },
  { name: 'Truyện', href: '/comics' },
]

const Navbar = () => {
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [notifOpen, setNotifOpen] = useState(false)
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/')
  }

  const handleSearch = (e) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate(`/videos?search=${encodeURIComponent(searchQuery)}`)
      setSearchOpen(false)
      setSearchQuery('')
    }
  }

  return (
    <header className="fixed top-0 left-0 right-0 z-50 hidden md:block">
      {/* Glass morphism background - pointer-events-none to allow clicks through */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-2xl border-b border-white/5 pointer-events-none" />
      
      <nav className="relative container-custom">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-3 group">
            <div className="w-9 h-9 rounded-xl overflow-hidden bg-gray-800 flex items-center justify-center">
              <img src="/favicon.png" alt="MediaHub" className="w-full h-full object-cover" />
            </div>
            <span className="text-xl font-bold bg-gradient-to-r from-white to-gray-300 bg-clip-text text-transparent">
              MediaHub
            </span>
          </Link>

          {/* Center Navigation */}
          <div className="flex items-center gap-1 bg-white/5 rounded-full px-2 py-1">
            {navLinks.map((link) => (
              <NavLink
                key={link.name}
                to={link.href}
                className={({ isActive }) => `
                  px-4 py-1.5 rounded-full text-sm font-medium transition-all duration-200
                  ${isActive 
                    ? 'bg-white/10 text-white' 
                    : 'text-gray-400 hover:text-white hover:bg-white/5'
                  }
                `}
              >
                {link.name}
              </NavLink>
            ))}
          </div>

          {/* Right side */}
          <div className="flex items-center gap-2">
            {/* Search */}
            {/* <div className="relative">
              {searchOpen ? (
                <form onSubmit={handleSearch} className="flex items-center">
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Tìm kiếm..."
                    className="w-48 px-4 py-1.5 bg-white/10 border border-white/10 rounded-full text-sm text-white placeholder-gray-500 focus:outline-none focus:border-violet-500/50 transition-all"
                    autoFocus
                    onBlur={() => !searchQuery && setSearchOpen(false)}
                  />
                </form>
              ) : (
                <button
                  onClick={() => setSearchOpen(true)}
                  className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-full transition-all"
                >
                  <MagnifyingGlassIcon className="w-5 h-5" />
                </button>
              )}
            </div> */}

            {/* Notifications */}
            {user && (
              <button
                onClick={() => setNotifOpen(true)}
                aria-expanded={notifOpen}
                className="relative p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-full transition-all"
              >
                <BellIcon className="w-5 h-5" />
                <span className="absolute top-1 right-1 w-2 h-2 bg-fuchsia-500 rounded-full" />
              </button>
            )}

            {/* Upload */}
            {user && (
              <Link
                to="/upload"
                className="flex items-center gap-2 px-4 py-1.5 bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:from-violet-500 hover:to-fuchsia-500 text-white text-sm font-medium rounded-full transition-all duration-300 hover:shadow-lg hover:shadow-violet-500/25"
              >
                <CloudArrowUpIcon className="w-4 h-4" />
                <span>Upload</span>
              </Link>
            )}

            {/* User menu */}
            {user ? (
              <Menu as="div" className="relative">
                <Menu.Button className="flex items-center gap-2 p-1 rounded-full hover:bg-white/10 transition-all">
                  {user.avatar ? (
                    <img
                      src={user.avatar}
                      alt={user.username}
                      className="w-8 h-8 rounded-full object-cover ring-2 ring-white/20"
                    />
                  ) : (
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center">
                      <span className="text-sm font-bold text-white">
                        {user.username?.charAt(0).toUpperCase()}
                      </span>
                    </div>
                  )}
                </Menu.Button>

                <Transition
                  as={Fragment}
                  enter="transition ease-out duration-200"
                  enterFrom="transform opacity-0 scale-95 translate-y-1"
                  enterTo="transform opacity-100 scale-100 translate-y-0"
                  leave="transition ease-in duration-150"
                  leaveFrom="transform opacity-100 scale-100 translate-y-0"
                  leaveTo="transform opacity-0 scale-95 translate-y-1"
                >
                  <Menu.Items className="absolute right-0 mt-2 w-56 bg-gray-900/95 backdrop-blur-xl rounded-2xl shadow-2xl border border-white/10 py-2 focus:outline-none overflow-hidden">
                    {/* User info */}
                    <div className="px-4 py-3 border-b border-white/10">
                      <p className="text-sm font-medium text-white">{user.username}</p>
                      <p className="text-xs text-gray-500">{user.email}</p>
                    </div>

                    <div className="py-1">
                      <Menu.Item>
                        {({ active }) => (
                          <Link
                            to="/profile"
                            className={`flex items-center gap-3 px-4 py-2.5 text-sm transition-colors ${
                              active ? 'bg-white/5 text-white' : 'text-gray-400'
                            }`}
                          >
                            <UserCircleIcon className="w-5 h-5" />
                            <span>Hồ sơ của tôi</span>
                          </Link>
                        )}
                      </Menu.Item>
                      
                      {user.role === 'admin' && (
                        <Menu.Item>
                          {({ active }) => (
                            <Link
                              to="/admin"
                              className={`flex items-center gap-3 px-4 py-2.5 text-sm transition-colors ${
                                active ? 'bg-white/5 text-white' : 'text-gray-400'
                              }`}
                            >
                              <Cog6ToothIcon className="w-5 h-5" />
                              <span>Quản trị</span>
                            </Link>
                          )}
                        </Menu.Item>
                      )}
                    </div>

                    <div className="border-t border-white/10 pt-1">
                      <Menu.Item>
                        {({ active }) => (
                          <button
                            onClick={handleLogout}
                            className={`w-full flex items-center gap-3 px-4 py-2.5 text-sm transition-colors ${
                              active ? 'bg-red-500/10 text-red-400' : 'text-gray-400'
                            }`}
                          >
                            <ArrowRightOnRectangleIcon className="w-5 h-5" />
                            <span>Đăng xuất</span>
                          </button>
                        )}
                      </Menu.Item>
                    </div>
                  </Menu.Items>
                </Transition>
              </Menu>
            ) : (
              <div className="flex items-center gap-2">
                <Link
                  to="/login"
                  className="px-4 py-1.5 text-sm text-gray-400 hover:text-white transition-colors"
                >
                  Đăng nhập
                </Link>
                <Link
                  to="/register"
                  className="px-4 py-1.5 bg-white text-gray-900 text-sm font-medium rounded-full hover:bg-gray-100 transition-colors"
                >
                  Đăng ký
                </Link>
              </div>
            )}
          </div>
        </div>
      </nav>
      <NotificationModal open={notifOpen} onClose={() => setNotifOpen(false)} notifications={[]} />
    </header>
  )
}

export default Navbar
