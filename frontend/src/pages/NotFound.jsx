import { Link } from 'react-router-dom'
import { HomeIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline'

const NotFound = () => {
  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4">
      <div className="text-center">
        {/* 404 */}
        <h1 className="text-9xl font-bold text-primary-500 mb-4">404</h1>
        
        {/* Message */}
        <h2 className="text-2xl font-bold text-white mb-2">
          Trang không tồn tại
        </h2>
        <p className="text-gray-400 mb-8 max-w-md mx-auto">
          Trang bạn đang tìm kiếm không tồn tại hoặc đã bị di chuyển. 
          Vui lòng kiểm tra lại URL hoặc quay về trang chủ.
        </p>

        {/* Actions */}
        <div className="flex flex-wrap gap-4 justify-center">
          <Link
            to="/"
            className="btn-primary flex items-center gap-2"
          >
            <HomeIcon className="w-5 h-5" />
            Về trang chủ
          </Link>
          <Link
            to="/videos"
            className="btn-secondary flex items-center gap-2"
          >
            <MagnifyingGlassIcon className="w-5 h-5" />
            Tìm kiếm
          </Link>
        </div>

        {/* Illustration */}
        <div className="mt-12 text-gray-700">
          <svg
            className="w-64 h-48 mx-auto"
            viewBox="0 0 200 150"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            {/* Film reel */}
            <circle cx="50" cy="75" r="40" stroke="currentColor" strokeWidth="4" />
            <circle cx="50" cy="75" r="15" stroke="currentColor" strokeWidth="2" />
            <circle cx="50" cy="45" r="5" fill="currentColor" />
            <circle cx="50" cy="105" r="5" fill="currentColor" />
            <circle cx="80" cy="75" r="5" fill="currentColor" />
            <circle cx="20" cy="75" r="5" fill="currentColor" />
            <circle cx="71" cy="54" r="5" fill="currentColor" />
            <circle cx="29" cy="96" r="5" fill="currentColor" />
            <circle cx="71" cy="96" r="5" fill="currentColor" />
            <circle cx="29" cy="54" r="5" fill="currentColor" />
            
            {/* Film strip */}
            <rect x="100" y="30" width="80" height="90" rx="4" stroke="currentColor" strokeWidth="3" />
            <rect x="105" y="40" width="10" height="10" fill="currentColor" />
            <rect x="165" y="40" width="10" height="10" fill="currentColor" />
            <rect x="105" y="100" width="10" height="10" fill="currentColor" />
            <rect x="165" y="100" width="10" height="10" fill="currentColor" />
            
            {/* X mark */}
            <line x1="125" y1="55" x2="155" y2="95" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
            <line x1="155" y1="55" x2="125" y2="95" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
          </svg>
        </div>
      </div>
    </div>
  )
}

export default NotFound
