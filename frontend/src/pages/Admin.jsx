import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  UsersIcon, 
  FilmIcon, 
  BookOpenIcon,
  TrashIcon,
  PencilIcon,
  ChartBarIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { adminAPI, videosAPI, comicsAPI } from '../services/api'

// Admin dashboard
const Admin = () => {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState('stats')
  const [searchQuery, setSearchQuery] = useState('')

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: () => adminAPI.getStats(),
    select: (res) => res.data,
  })

  // Fetch users
  const { data: usersData } = useQuery({
    queryKey: ['admin', 'users', searchQuery],
    queryFn: () => adminAPI.getUsers({ search: searchQuery }),
    select: (res) => res.data,
    enabled: activeTab === 'users',
  })

  // Fetch videos
  const { data: videosData } = useQuery({
    queryKey: ['admin', 'videos', searchQuery],
    queryFn: () => adminAPI.getVideos({ search: searchQuery }),
    select: (res) => res.data,
    enabled: activeTab === 'videos',
  })

  // Fetch comics
  const { data: comicsData } = useQuery({
    queryKey: ['admin', 'comics', searchQuery],
    queryFn: () => adminAPI.getComics({ search: searchQuery }),
    select: (res) => res.data,
    enabled: activeTab === 'comics',
  })

  // Delete video mutation
  const deleteVideoMutation = useMutation({
    mutationFn: (id) => videosAPI.delete(id),
    onSuccess: () => {
      toast.success('Đã xóa video')
      queryClient.invalidateQueries(['admin', 'videos'])
    },
    onError: () => toast.error('Xóa thất bại'),
  })

  // Delete comic mutation
  const deleteComicMutation = useMutation({
    mutationFn: (id) => comicsAPI.delete(id),
    onSuccess: () => {
      toast.success('Đã xóa truyện')
      queryClient.invalidateQueries(['admin', 'comics'])
    },
    onError: () => toast.error('Xóa thất bại'),
  })

  // Ban user mutation
  const banUserMutation = useMutation({
    mutationFn: (userId) => adminAPI.banUser(userId),
    onSuccess: () => {
      toast.success('Đã cập nhật trạng thái user')
      queryClient.invalidateQueries(['admin', 'users'])
    },
    onError: () => toast.error('Cập nhật thất bại'),
  })

  // Format number
  const formatNumber = (num) => {
    if (!num) return '0'
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
    return num.toString()
  }

  const tabs = [
    { id: 'stats', label: 'Thống kê', icon: ChartBarIcon },
    { id: 'users', label: 'Users', icon: UsersIcon },
    { id: 'videos', label: 'Videos', icon: FilmIcon },
    { id: 'comics', label: 'Comics', icon: BookOpenIcon },
  ]

  return (
    <div className="container-custom py-8">
      <h1 className="text-3xl font-bold text-white mb-8">Admin Dashboard</h1>

      {/* Tabs */}
      <div className="flex flex-wrap gap-2 mb-8 border-b border-gray-700 pb-4">
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

      {/* Stats Tab */}
      {activeTab === 'stats' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-blue-500/20 rounded-lg">
                <UsersIcon className="w-8 h-8 text-blue-400" />
              </div>
              <div>
                <p className="text-gray-400 text-sm">Tổng Users</p>
                <p className="text-2xl font-bold text-white">
                  {formatNumber(stats?.total_users)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-red-500/20 rounded-lg">
                <FilmIcon className="w-8 h-8 text-red-400" />
              </div>
              <div>
                <p className="text-gray-400 text-sm">Tổng Videos</p>
                <p className="text-2xl font-bold text-white">
                  {formatNumber(stats?.total_videos)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-green-500/20 rounded-lg">
                <BookOpenIcon className="w-8 h-8 text-green-400" />
              </div>
              <div>
                <p className="text-gray-400 text-sm">Tổng Truyện</p>
                <p className="text-2xl font-bold text-white">
                  {formatNumber(stats?.total_comics)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-purple-500/20 rounded-lg">
                <ChartBarIcon className="w-8 h-8 text-purple-400" />
              </div>
              <div>
                <p className="text-gray-400 text-sm">Lượt xem hôm nay</p>
                <p className="text-2xl font-bold text-white">
                  {formatNumber(stats?.views_today)}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Search bar for data tabs */}
      {activeTab !== 'stats' && (
        <div className="mb-6">
          <div className="relative max-w-md">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Tìm kiếm..."
              className="input pl-10"
            />
          </div>
        </div>
      )}

      {/* Users Tab */}
      {activeTab === 'users' && (
        <div className="bg-gray-800 rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700">
                <tr>
                  <th className="text-left p-4 text-gray-300">Username</th>
                  <th className="text-left p-4 text-gray-300">Email</th>
                  <th className="text-left p-4 text-gray-300">Role</th>
                  <th className="text-left p-4 text-gray-300">Ngày tạo</th>
                  <th className="text-left p-4 text-gray-300">Trạng thái</th>
                  <th className="p-4"></th>
                </tr>
              </thead>
              <tbody>
                {usersData?.users?.map((user) => (
                  <tr key={user.id} className="border-t border-gray-700">
                    <td className="p-4 text-white">{user.username}</td>
                    <td className="p-4 text-gray-400">{user.email}</td>
                    <td className="p-4">
                      <span className={`px-2 py-1 rounded text-xs ${
                        user.role === 'admin' ? 'bg-red-500/20 text-red-400' : 'bg-gray-600 text-gray-300'
                      }`}>
                        {user.role}
                      </span>
                    </td>
                    <td className="p-4 text-gray-400">
                      {new Date(user.created_at).toLocaleDateString('vi-VN')}
                    </td>
                    <td className="p-4">
                      <span className={`px-2 py-1 rounded text-xs ${
                        user.is_banned ? 'bg-red-500/20 text-red-400' : 'bg-green-500/20 text-green-400'
                      }`}>
                        {user.is_banned ? 'Banned' : 'Active'}
                      </span>
                    </td>
                    <td className="p-4">
                      {user.role !== 'admin' && (
                        <button
                          onClick={() => banUserMutation.mutate(user.id)}
                          className={`px-3 py-1 rounded text-sm ${
                            user.is_banned 
                              ? 'bg-green-500/20 text-green-400 hover:bg-green-500/30' 
                              : 'bg-red-500/20 text-red-400 hover:bg-red-500/30'
                          }`}
                        >
                          {user.is_banned ? 'Unban' : 'Ban'}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(!usersData?.users || usersData.users.length === 0) && (
            <p className="text-center text-gray-500 py-8">Không có dữ liệu</p>
          )}
        </div>
      )}

      {/* Videos Tab */}
      {activeTab === 'videos' && (
        <div className="bg-gray-800 rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700">
                <tr>
                  <th className="text-left p-4 text-gray-300">Video</th>
                  <th className="text-left p-4 text-gray-300">Uploader</th>
                  <th className="text-left p-4 text-gray-300">Lượt xem</th>
                  <th className="text-left p-4 text-gray-300">Ngày tạo</th>
                  <th className="p-4"></th>
                </tr>
              </thead>
              <tbody>
                {videosData?.videos?.map((video) => (
                  <tr key={video.id} className="border-t border-gray-700">
                    <td className="p-4">
                      <div className="flex items-center gap-3">
                        <img
                          src={video.thumbnail || '/placeholder-video.png'}
                          alt={video.title}
                          className="w-16 h-10 object-cover rounded"
                        />
                        <span className="text-white truncate max-w-xs">{video.title}</span>
                      </div>
                    </td>
                    <td className="p-4 text-gray-400">{video.uploader?.username || 'Unknown'}</td>
                    <td className="p-4 text-gray-400">{formatNumber(video.views)}</td>
                    <td className="p-4 text-gray-400">
                      {new Date(video.created_at).toLocaleDateString('vi-VN')}
                    </td>
                    <td className="p-4">
                      <div className="flex gap-2">
                        <button className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded">
                          <PencilIcon className="w-4 h-4" />
                        </button>
                        <button 
                          onClick={() => {
                            if (confirm('Xác nhận xóa video này?')) {
                              deleteVideoMutation.mutate(video.id)
                            }
                          }}
                          className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/20 rounded"
                        >
                          <TrashIcon className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(!videosData?.videos || videosData.videos.length === 0) && (
            <p className="text-center text-gray-500 py-8">Không có dữ liệu</p>
          )}
        </div>
      )}

      {/* Comics Tab */}
      {activeTab === 'comics' && (
        <div className="bg-gray-800 rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700">
                <tr>
                  <th className="text-left p-4 text-gray-300">Truyện</th>
                  <th className="text-left p-4 text-gray-300">Tác giả</th>
                  <th className="text-left p-4 text-gray-300">Chapters</th>
                  <th className="text-left p-4 text-gray-300">Lượt xem</th>
                  <th className="text-left p-4 text-gray-300">Trạng thái</th>
                  <th className="p-4"></th>
                </tr>
              </thead>
              <tbody>
                {comicsData?.comics?.map((comic) => (
                  <tr key={comic.id} className="border-t border-gray-700">
                    <td className="p-4">
                      <div className="flex items-center gap-3">
                        <img
                          src={comic.cover_image || '/placeholder-comic.png'}
                          alt={comic.title}
                          className="w-10 h-14 object-cover rounded"
                        />
                        <span className="text-white truncate max-w-xs">{comic.title}</span>
                      </div>
                    </td>
                    <td className="p-4 text-gray-400">{comic.author || '-'}</td>
                    <td className="p-4 text-gray-400">{comic.chapters?.length || 0}</td>
                    <td className="p-4 text-gray-400">{formatNumber(comic.views)}</td>
                    <td className="p-4">
                      <span className={`px-2 py-1 rounded text-xs ${
                        comic.status === 'completed' 
                          ? 'bg-green-500/20 text-green-400' 
                          : 'bg-blue-500/20 text-blue-400'
                      }`}>
                        {comic.status === 'completed' ? 'Hoàn thành' : 'Đang cập nhật'}
                      </span>
                    </td>
                    <td className="p-4">
                      <div className="flex gap-2">
                        <button className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded">
                          <PencilIcon className="w-4 h-4" />
                        </button>
                        <button 
                          onClick={() => {
                            if (confirm('Xác nhận xóa truyện này?')) {
                              deleteComicMutation.mutate(comic.id)
                            }
                          }}
                          className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/20 rounded"
                        >
                          <TrashIcon className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(!comicsData?.comics || comicsData.comics.length === 0) && (
            <p className="text-center text-gray-500 py-8">Không có dữ liệu</p>
          )}
        </div>
      )}
    </div>
  )
}

export default Admin
