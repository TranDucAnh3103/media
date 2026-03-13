import { useNavigate } from 'react-router-dom'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'

/**
 * Mobile header with back button
 * Hiển thị trên mobile khi vào trang chi tiết
 */
const MobileHeader = ({ title, showTitle = true, transparent = false }) => {
  const navigate = useNavigate()

  const handleBack = () => {
    // Nếu có history thì quay lại, không thì về home
    if (window.history.length > 2) {
      navigate(-1)
    } else {
      navigate('/')
    }
  }

  return (
    <header
      className={`
        fixed top-0 left-0 right-0 z-50 md:hidden
        ${transparent
          ? 'bg-gradient-to-b from-black/80 to-transparent'
          : 'bg-gray-950/95 backdrop-blur-xl border-b border-white/5'
        }
      `}
    >
      <div className="flex items-center gap-3 px-4 h-14 safe-top">
        {/* Back button */}
        <button
          onClick={handleBack}
          className="flex items-center justify-center w-10 h-10 -ml-2 rounded-full hover:bg-white/10 active:scale-95 transition-all"
          aria-label="Quay lại"
        >
          <ArrowLeftIcon className="w-6 h-6 text-white" />
        </button>

        {/* Logo + Title */}
        <div className="flex-1 flex items-center gap-3 truncate">
          <div className="w-8 h-8 rounded-lg overflow-hidden bg-gray-800 flex-shrink-0">
            <img src="/favicon.png" alt="MediaHub" className="w-full h-full object-cover" />
          </div>
          {showTitle && title ? (
            <h1 className="text-lg font-semibold text-white truncate">
              {title}
            </h1>
          ) : (
            <span className="text-lg font-semibold text-white truncate">MediaHub</span>
          )}
        </div>
      </div>
    </header>
  )
}

export default MobileHeader
