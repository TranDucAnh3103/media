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
      {/* Blur background - must be pointer-events-none to allow clicks through */}
      <div className="absolute inset-0 bg-black/80 backdrop-blur-xl border-t border-white/10 pointer-events-none" />
      
      {/* Safe area padding for iPhone notch */}
      <div className="relative flex items-center justify-around h-[4.5rem] pb-safe px-2">
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
              className={`flex flex-col items-center justify-center w-16 h-full relative transition-all duration-300 ${
                tab.special ? '-mt-6 z-10' : ''
              }`}
            >
              {tab.special ? (
                // Special upload FAB
                <div className={`w-14 h-14 rounded-full flex items-center justify-center transition-all duration-300 shadow-[0_4px_15px_rgba(139,92,246,0.3)] ${
                  active 
                    ? 'bg-gradient-to-r from-violet-500 to-fuchsia-500 scale-105 shadow-[0_4px_20px_rgba(217,70,239,0.5)]' 
                    : 'bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:scale-110 hover:shadow-[0_4px_25px_rgba(139,92,246,0.6)]'
                }`}>
                  <Icon className="w-7 h-7 text-white" />
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center w-full h-full group relative pb-1">
                  <div className={`relative transition-all duration-300 ${active ? 'scale-110 -translate-y-1' : 'group-hover:scale-110'}`}>
                    <Icon className={`w-6 h-6 transition-all duration-300 ${
                      active ? 'text-violet-400' : 'text-gray-500 group-hover:text-gray-300'
                    }`} />
                  </div>
                  <span className={`text-[10px] font-medium transition-all duration-300 ${
                    active ? 'text-violet-400 opacity-100 translate-y-0' : 'text-gray-500 opacity-80'
                  }`}>
                    {tab.name}
                  </span>
                  {/* Subtle Dot Indicator */}
                  {active && (
                    <div className="absolute bottom-1 w-1.5 h-1.5 bg-violet-400 rounded-full shadow-[0_0_8px_rgba(167,139,250,0.8)]" />
                  )}
                </div>
              )}
            </NavLink>
          )
        })}
      </div>
    </nav>
  )
}

export default BottomNav
