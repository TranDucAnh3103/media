import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  UserCircleIcon,
  BookmarkIcon,
  HeartIcon,
  Cog6ToothIcon,
  TrashIcon,
  PlayIcon,
  Squares2X2Icon,
  ArrowRightOnRectangleIcon,
  FilmIcon,
  BookOpenIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { userAPI } from '../services/api'
import { useAuthStore } from '../store/authStore'

// Helper function to format duration
const formatDuration = (seconds) => {
  if (!seconds) return ''
  const hrs = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  if (hrs > 0) {
    return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

// User profile page
const Profile = () => {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { user, updateUser, logout } = useAuthStore()
  const [activeTab, setActiveTab] = useState('bookmarks')
  const [editMode, setEditMode] = useState(false)
  const [profileForm, setProfileForm] = useState({
    username: user?.username || '',
    avatar: user?.avatar || '',
  })

  // Fetch user profile (with bookmarks, playlists, history)
  const { data: profile } = useQuery({
    queryKey: ['user', 'profile'],
    queryFn: () => userAPI.getProfile(),
    select: (res) => res.data,
  })

  // Update profile mutation
  const updateMutation = useMutation({
    mutationFn: (data) => userAPI.updateProfile(data),
    onSuccess: (res) => {
      updateUser(res.data.user)
      setEditMode(false)
      toast.success('Cập nhật thành công')
    },
    onError: (err) => {
      toast.error(err.response?.data?.error || 'Cập nhật thất bại')
    },
  })

  // Remove bookmark mutation
  const removeBookmarkMutation = useMutation({
    mutationFn: (contentId) => userAPI.removeBookmark(contentId),
    onSuccess: () => {
      toast.success('Đã xóa bookmark')
      queryClient.invalidateQueries({ queryKey: ['user', 'profile'] })
    },
    onError: () => toast.error('Xóa bookmark thất bại'),
  })

  // Remove like mutation
  const removeLikeMutation = useMutation({
    mutationFn: ({ type, contentId }) => userAPI.removeLike(type, contentId),
    onSuccess: () => {
      toast.success('Đã bỏ thích')
      queryClient.invalidateQueries({ queryKey: ['user', 'profile'] })
    },
    onError: () => toast.error('Bỏ thích thất bại'),
  })

  // Clear history mutation
  const clearHistoryMutation = useMutation({
    mutationFn: () => userAPI.clearHistory(),
    onSuccess: () => {
      toast.success('Đã xóa lịch sử')
      queryClient.invalidateQueries({ queryKey: ['user', 'profile'] })
    },
  })

  // Handle logout
  const handleLogout = () => {
    logout()
    navigate('/login')
    toast.success('Đã đăng xuất')
  }

  const tabs = [
    { id: 'bookmarks', label: 'Đánh dấu', icon: BookmarkIcon },
    // { id: 'playlists', label: 'Playlists', icon: PlayIcon },
    // { id: 'history', label: 'Lịch sử', icon: ClockIcon },
    { id: 'liked', label: 'Đã thích', icon: HeartIcon },
    { id: 'settings', label: 'Cài đặt', icon: Cog6ToothIcon },
  ]

  // Quick action links
  const quickLinks = [
    { href: '/dashboard', label: 'Dashboard', icon: Squares2X2Icon, color: 'violet' },
    // { href: '/admin', label: 'Admin', icon: ShieldCheckIcon, color: 'amber' },
  ]

  // Format date
  const formatDate = (date) => {
    if (!date) return ''
    return new Date(date).toLocaleDateString('vi-VN', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="container-custom py-8">
      {/* Profile Header */}
      <div className="bg-gray-800 rounded-xl p-6 mb-8">
        <div className="flex flex-col sm:flex-row items-center gap-6">
          {/* Avatar */}
          <div className="relative">
            {user?.avatar ? (
              <img
                src={user.avatar}
                alt={user.username}
                className="w-24 h-24 rounded-full object-cover"
              />
            ) : (
              <UserCircleIcon className="w-24 h-24 text-gray-500" />
            )}
          </div>

          {/* Info */}
          <div className="flex-1 text-center sm:text-left">
            {editMode ? (
              <div className="space-y-3 max-w-xs">
                <input
                  type="text"
                  value={profileForm.username}
                  onChange={(e) => setProfileForm({ ...profileForm, username: e.target.value })}
                  className="input"
                  placeholder="Username"
                />
                <input
                  type="url"
                  value={profileForm.avatar}
                  onChange={(e) => setProfileForm({ ...profileForm, avatar: e.target.value })}
                  className="input"
                  placeholder="URL avatar"
                />
                <div className="flex gap-2">
                  <button
                    onClick={() => updateMutation.mutate(profileForm)}
                    className="btn-primary"
                    disabled={updateMutation.isLoading}
                  >
                    Lưu
                  </button>
                  <button
                    onClick={() => setEditMode(false)}
                    className="btn-secondary"
                  >
                    Hủy
                  </button>
                </div>
              </div>
            ) : (
              <>
                <h1 className="text-2xl font-bold text-white">{user?.username}</h1>
                <p className="text-gray-400">{user?.email}</p>
                <p className="text-sm text-gray-500 mt-1">
                  Tham gia: {formatDate(user?.created_at)}
                </p>
                <button
                  onClick={() => setEditMode(true)}
                  className="btn-secondary mt-3"
                >
                  Chỉnh sửa profile
                </button>
              </>
            )}
          </div>

          {/* Stats */}
          <div className="flex gap-6 text-center">
            <div>
              <p className="text-2xl font-bold text-white">
                {profile?.bookmarks?.length || 0}
              </p>
              <p className="text-sm text-gray-400">Bookmarks</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-white">
                {profile?.playlists?.length || 0}
              </p>
              <p className="text-sm text-gray-400">Playlists</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-white">
                {profile?.watch_history?.length || 0}
              </p>
              <p className="text-sm text-gray-400">Đã xem</p>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="flex flex-wrap gap-3 mb-6">
        {quickLinks.map((link) => (
          <Link
            key={link.href}
            to={link.href}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-lg transition-colors ${
              link.color === 'violet' 
                ? 'bg-violet-500/20 text-violet-300 hover:bg-violet-500/30 border border-violet-500/30' 
                : 'bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 border border-amber-500/30'
            }`}
          >
            <link.icon className="w-5 h-5" />
            {link.label}
          </Link>
        ))}
        <button
          onClick={handleLogout}
          className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30 transition-colors"
        >
          <ArrowRightOnRectangleIcon className="w-5 h-5" />
          Đăng xuất
        </button>
      </div>

      {/* Tabs */}
      <div className="flex flex-wrap gap-2 mb-6 border-b border-gray-700 pb-4">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
              activeTab === tab.id
                ? 'bg-primary-600 text-white'
                : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
            }`}
          >
            <tab.icon className="w-5 h-5" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Bookmarks Tab */}
      {activeTab === 'bookmarks' && (
        <div>
          <h2 className="text-xl font-bold text-white mb-4">Đã đánh dấu</h2>
          {profile?.bookmarks?.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {profile.bookmarks.map((bookmark, index) => (
                <div 
                  key={bookmark.content_id || index}
                  className="bg-gray-800 rounded-xl overflow-hidden group hover:bg-gray-750 transition-colors"
                >
                  <Link 
                    to={bookmark.content_type === 'video' 
                      ? `/videos/${bookmark.content_id}` 
                      : `/comics/${bookmark.content_id}`}
                    className="block"
                  >
                    <div className="relative aspect-video bg-gray-700">
                      {bookmark.thumbnail ? (
                        <img
                          src={bookmark.thumbnail}
                          alt={bookmark.title}
                          className="w-full h-full object-cover"
                          onError={(e) => { e.target.src = '/placeholder.png' }}
                        />
                      ) : (
                        <div className="flex items-center justify-center h-full">
                          {bookmark.content_type === 'video' ? (
                            <FilmIcon className="w-12 h-12 text-gray-600" />
                          ) : (
                            <BookOpenIcon className="w-12 h-12 text-gray-600" />
                          )}
                        </div>
                      )}
                      <div className="absolute top-2 left-2 px-2 py-1 bg-black/60 rounded text-xs text-white">
                        {bookmark.content_type === 'video' ? 'Video' : 'Truyện'}
                      </div>
                    </div>
                  </Link>
                  <div className="p-4">
                    <div className="flex items-start justify-between gap-2">
                      <Link 
                        to={bookmark.content_type === 'video' 
                          ? `/videos/${bookmark.content_id}` 
                          : `/comics/${bookmark.content_id}`}
                        className="flex-1"
                      >
                        <h3 className="text-white font-medium line-clamp-2 group-hover:text-violet-400 transition-colors">
                          {bookmark.title || 'Không có tiêu đề'}
                        </h3>
                        <p className="text-sm text-gray-400 mt-1">
                          {bookmark.chapter && `Chapter ${bookmark.chapter}`}
                          {bookmark.page && ` • Trang ${bookmark.page}`}
                        </p>
                        <p className="text-xs text-gray-500 mt-1">
                          {formatDate(bookmark.created_at)}
                        </p>
                      </Link>
                      <button
                        onClick={() => removeBookmarkMutation.mutate(bookmark.content_id)}
                        disabled={removeBookmarkMutation.isPending}
                        className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/20 rounded-lg transition-colors shrink-0"
                        title="Xóa bookmark"
                      >
                        <TrashIcon className="w-5 h-5" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <BookmarkIcon className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-500">Chưa có bookmark nào</p>
              <p className="text-sm text-gray-600 mt-1">Đánh dấu video hoặc truyện để xem sau</p>
            </div>
          )}
        </div>
      )}

      {/* Playlists Tab */}
      {activeTab === 'playlists' && (
        <div>
          <h2 className="text-xl font-bold text-white mb-4">Playlists</h2>
          {profile?.playlists?.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {profile.playlists.map((playlist) => (
                <Link
                  key={playlist.id}
                  to={`/playlist/${playlist.id}`}
                  className="bg-gray-800 rounded-lg overflow-hidden group"
                >
                  <div className="aspect-video bg-gray-700 relative">
                    {playlist.videos?.[0]?.thumbnail ? (
                      <img
                        src={playlist.videos[0].thumbnail}
                        alt={playlist.name}
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="flex items-center justify-center h-full">
                        <PlayIcon className="w-12 h-12 text-gray-600" />
                      </div>
                    )}
                    <div className="absolute bottom-0 right-0 bg-black/80 px-2 py-1 text-sm text-white">
                      {playlist.videos?.length || 0} video
                    </div>
                  </div>
                  <div className="p-3">
                    <h3 className="text-white font-medium group-hover:text-primary-400 truncate">
                      {playlist.name}
                    </h3>
                    <p className="text-sm text-gray-400 mt-1">
                      {playlist.is_public ? 'Công khai' : 'Riêng tư'}
                    </p>
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <p className="text-gray-500 text-center py-8">Chưa có playlist nào</p>
          )}
        </div>
      )}

      {/* History Tab */}
      {activeTab === 'history' && (
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-white">Lịch sử xem</h2>
            {profile?.watch_history?.length > 0 && (
              <button
                onClick={() => {
                  if (confirm('Xác nhận xóa tất cả lịch sử?')) {
                    clearHistoryMutation.mutate()
                  }
                }}
                className="text-sm text-red-400 hover:text-red-300"
              >
                Xóa tất cả
              </button>
            )}
          </div>
          {profile?.watch_history?.length > 0 ? (
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
              {profile.watch_history.map((item) => (
                <div key={item.id} className="relative group">
                  {item.content_type === 'video' ? (
                    <VideoCard video={item.content} />
                  ) : (
                    <ComicCard comic={item.content} />
                  )}
                  <div className="absolute bottom-0 left-0 right-0 h-1 bg-gray-700">
                    <div 
                      className="h-full bg-primary-500"
                      style={{ width: `${item.progress || 0}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-500 text-center py-8">Chưa có lịch sử xem</p>
          )}
        </div>
      )}

      {/* Liked Tab */}
      {activeTab === 'liked' && (
        <div>
          <h2 className="text-xl font-bold text-white mb-4">Đã thích</h2>
          {profile?.liked?.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {profile.liked.map((item, index) => (
                <div 
                  key={item.id || index}
                  className="bg-gray-800 rounded-xl overflow-hidden group hover:bg-gray-750 transition-colors"
                >
                  <Link 
                    to={item.content_type === 'video' 
                      ? `/videos/${item.id}` 
                      : `/comics/${item.id}`}
                    className="block"
                  >
                    <div className="relative aspect-video bg-gray-700">
                      {item.thumbnail ? (
                        <img
                          src={item.thumbnail}
                          alt={item.title}
                          className="w-full h-full object-cover"
                          onError={(e) => { e.target.src = '/placeholder.png' }}
                        />
                      ) : (
                        <div className="flex items-center justify-center h-full">
                          {item.content_type === 'video' ? (
                            <FilmIcon className="w-12 h-12 text-gray-600" />
                          ) : (
                            <BookOpenIcon className="w-12 h-12 text-gray-600" />
                          )}
                        </div>
                      )}
                      <div className="absolute top-2 left-2 px-2 py-1 bg-black/60 rounded text-xs text-white">
                        {item.content_type === 'video' ? 'Video' : 'Truyện'}
                      </div>
                    </div>
                  </Link>
                  <div className="p-4">
                    <div className="flex items-start justify-between gap-2">
                      <Link 
                        to={item.content_type === 'video' 
                          ? `/videos/${item.id}` 
                          : `/comics/${item.id}`}
                        className="flex-1"
                      >
                        <h3 className="text-white font-medium line-clamp-2 group-hover:text-violet-400 transition-colors">
                          {item.title || 'Không có tiêu đề'}
                        </h3>
                        {item.content_type === 'video' && item.duration && (
                          <p className="text-sm text-gray-400 mt-1">
                            {formatDuration(item.duration)}
                          </p>
                        )}
                        {item.views !== undefined && (
                          <p className="text-xs text-gray-500 mt-1">
                            {item.views} lượt xem
                          </p>
                        )}
                      </Link>
                      <button
                        onClick={() => removeLikeMutation.mutate({ type: item.content_type, contentId: item.id })}
                        disabled={removeLikeMutation.isPending}
                        className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/20 rounded-lg transition-colors shrink-0"
                        title="Bỏ thích"
                      >
                        <HeartIcon className="w-5 h-5" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <HeartIcon className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-500">Chưa thích nội dung nào</p>
              <p className="text-sm text-gray-600 mt-1">Thích video hoặc truyện để xem lại sau</p>
            </div>
          )}
        </div>
      )}

      {/* Settings Tab */}
      {activeTab === 'settings' && (
        <div className="max-w-xl">
          <h2 className="text-xl font-bold text-white mb-4">Cài đặt tài khoản</h2>
          
          <div className="bg-gray-800 rounded-lg p-6 space-y-6">
            {/* Change Password */}
            <div>
              <h3 className="text-white font-medium mb-3">Đổi mật khẩu</h3>
              <form className="space-y-3">
                <input
                  type="password"
                  placeholder="Mật khẩu hiện tại"
                  className="input"
                />
                <input
                  type="password"
                  placeholder="Mật khẩu mới"
                  className="input"
                />
                <input
                  type="password"
                  placeholder="Xác nhận mật khẩu mới"
                  className="input"
                />
                <button type="submit" className="btn-primary">
                  Đổi mật khẩu
                </button>
              </form>
            </div>

            <hr className="border-gray-700" />

            {/* Notifications */}
            <div>
              <h3 className="text-white font-medium mb-3">Thông báo</h3>
              <div className="space-y-2">
                <label className="flex items-center gap-2 text-gray-400">
                  <input
                    type="checkbox"
                    className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-primary-500"
                    defaultChecked
                  />
                  Nhận thông báo khi có truyện mới
                </label>
                <label className="flex items-center gap-2 text-gray-400">
                  <input
                    type="checkbox"
                    className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-primary-500"
                    defaultChecked
                  />
                  Nhận thông báo khi có video mới
                </label>
              </div>
            </div>

            <hr className="border-gray-700" />

            {/* Danger Zone */}
            <div>
              <h3 className="text-red-400 font-medium mb-3">Vùng nguy hiểm</h3>
              <button className="px-4 py-2 border border-red-500 text-red-400 rounded-lg hover:bg-red-500/20 transition-colors">
                Xóa tài khoản
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default Profile
