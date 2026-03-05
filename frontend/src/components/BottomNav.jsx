import { NavLink, useLocation } from 'react-router-dom'
import {
  HomeIcon,
  FilmIcon,
  BookOpenIcon,
  UserCircleIcon,
  PlusCircleIcon,
} from '@heroicons/react/24/outline'
import {
  HomeIcon as HomeIconSolid,
  FilmIcon as FilmIconSolid,
  BookOpenIcon as BookOpenIconSolid,
  UserCircleIcon as UserIconSolid,
  PlusCircleIcon as PlusIconSolid,
} from '@heroicons/react/24/solid'
import { useAuthStore } from '../store/authStore'

// Bottom navigation tabs
const tabs = [
  { name: 'Home', href: '/', icon: HomeIcon, iconActive: HomeIconSolid },
  { name: 'Video', href: '/videos', icon: FilmIcon, iconActive: FilmIconSolid },
  { name: 'Upload', href: '/upload', icon: PlusCircleIcon, iconActive: PlusIconSolid, special: true },
  { name: 'Truyện', href: '/comics', icon: BookOpenIcon, iconActive: BookOpenIconSolid },
  { name: 'Tôi', href: '/profile', icon: UserCircleIcon, iconActive: UserIconSolid },
]

const BottomNav = () => {
  const location = useLocation()
  const { user } = useAuthStore()

  // Check if current path matches
  const isActive = (href) => {
    if (href === '/') return location.pathname === '/'
    return location.pathname.startsWith(href)
  }

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 md:hidden">
      {/* Blur background */}
      <div className="absolute inset-0 bg-black/80 backdrop-blur-xl border-t border-white/10" />
      
      {/* Safe area padding for iPhone notch */}
      <div className="relative flex items-center justify-around h-16 pb-safe">
        {tabs.map((tab) => {
          const active = isActive(tab.href)
          const Icon = active ? tab.iconActive : tab.icon
          
          // Upload button - redirect to login if not logged in
          const href = tab.href === '/upload' && !user ? '/login' : tab.href
          const linkHref = tab.href === '/profile' && !user ? '/login' : href
          
          return (
            <NavLink
              key={tab.name}
              to={linkHref}
              className={`flex flex-col items-center justify-center w-16 h-full transition-all duration-200 ${
                tab.special ? 'relative -mt-4' : ''
              }`}
            >
              {tab.special ? (
                // Special upload button
                <div className={`w-12 h-12 rounded-full flex items-center justify-center shadow-lg transition-all duration-300 ${
                  active 
                    ? 'bg-gradient-to-r from-violet-500 to-fuchsia-500 scale-110' 
                    : 'bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:scale-105'
                }`}>
                  <Icon className="w-6 h-6 text-white" />
                </div>
              ) : (
                <>
                  <Icon className={`w-6 h-6 transition-all duration-200 ${
                    active 
                      ? 'text-white scale-110' 
                      : 'text-gray-500'
                  }`} />
                  <span className={`text-[10px] mt-1 font-medium transition-colors ${
                    active ? 'text-white' : 'text-gray-500'
                  }`}>
                    {tab.name}
                  </span>
                  {/* Active indicator */}
                  {active && (
                    <div className="absolute -top-0.5 w-6 h-0.5 bg-gradient-to-r from-violet-500 to-fuchsia-500 rounded-full" />
                  )}
                </>
              )}
            </NavLink>
          )
        })}
      </div>
    </nav>
  )
}

export default BottomNav
