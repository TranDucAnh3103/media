import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  FilmIcon,
  BookOpenIcon,
  PencilSquareIcon,
  TrashIcon,
  EyeIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
} from '@heroicons/react/24/outline'
import { Dialog } from '@headlessui/react'
import toast from 'react-hot-toast'
import MobileHeader from '../components/MobileHeader'
import { videosAPI, comicsAPI } from '../services/api'

// Dashboard - Quản lý video và truyện của user
const Dashboard = () => {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState('videos') // videos | comics
  const [searchQuery, setSearchQuery] = useState('')
  const [deleteModal, setDeleteModal] = useState({ open: false, item: null, type: null })
  const [editModal, setEditModal] = useState({ open: false, item: null, type: null })

  // Fetch user's videos
  const { data: videosData, isLoading: loadingVideos, refetch: refetchVideos } = useQuery({
    queryKey: ['my-videos'],
    queryFn: () => videosAPI.getMyVideos(),
    select: (res) => res.data?.videos || [],
    staleTime: 0, // Always refetch
    refetchOnMount: true,
  })

  // Fetch user's comics
  const { data: comicsData, isLoading: loadingComics, refetch: refetchComics } = useQuery({
    queryKey: ['my-comics'],
    queryFn: () => comicsAPI.getMyComics(),
    select: (res) => res.data?.comics || [],
    staleTime: 0, // Always refetch
    refetchOnMount: true,
  })

  // Delete video mutation
  const deleteVideoMutation = useMutation({
    mutationFn: (id) => videosAPI.delete(id),
    onSuccess: () => {
      toast.success('Đã xóa video')
      queryClient.invalidateQueries({ queryKey: ['my-videos'] })
      queryClient.invalidateQueries({ queryKey: ['videos'] })
      setDeleteModal({ open: false, item: null, type: null })
    },
    onError: () => toast.error('Xóa thất bại'),
  })

  // Delete comic mutation
  const deleteComicMutation = useMutation({
    mutationFn: (id) => comicsAPI.delete(id),
    onSuccess: () => {
      toast.success('Đã xóa truyện')
      queryClient.invalidateQueries({ queryKey: ['my-comics'] })
      queryClient.invalidateQueries({ queryKey: ['comics'] })
      setDeleteModal({ open: false, item: null, type: null })
    },
    onError: () => toast.error('Xóa thất bại'),
  })

  // Update video mutation
  const updateVideoMutation = useMutation({
    mutationFn: ({ id, data }) => videosAPI.update(id, data),
    onSuccess: () => {
      toast.success('Đã cập nhật video')
      queryClient.invalidateQueries({ queryKey: ['my-videos'] })
      queryClient.invalidateQueries({ queryKey: ['videos'] })
      setEditModal({ open: false, item: null, type: null })
    },
    onError: () => toast.error('Cập nhật thất bại'),
  })

  // Update comic mutation
  const updateComicMutation = useMutation({
    mutationFn: ({ id, data }) => comicsAPI.update(id, data),
    onSuccess: () => {
      toast.success('Đã cập nhật truyện')
      queryClient.invalidateQueries({ queryKey: ['my-comics'] })
      queryClient.invalidateQueries({ queryKey: ['comics'] })
      setEditModal({ open: false, item: null, type: null })
    },
    onError: () => toast.error('Cập nhật thất bại'),
  })

  // Handle refresh
  const handleRefresh = () => {
    refetchVideos()
    refetchComics()
    toast.success('Đã làm mới dữ liệu')
  }

  // Filter items by search
  const filteredVideos = (videosData || []).filter(v => 
    v.title?.toLowerCase().includes(searchQuery.toLowerCase())
  )
  const filteredComics = (comicsData || []).filter(c => 
    c.title?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  // Handle delete
  const handleDelete = () => {
    if (deleteModal.type === 'video') {
      deleteVideoMutation.mutate(deleteModal.item.id)
    } else {
      deleteComicMutation.mutate(deleteModal.item.id)
    }
  }

  // Handle edit save
  const handleEditSave = (formData) => {
    if (editModal.type === 'video') {
      updateVideoMutation.mutate({ id: editModal.item.id, data: formData })
    } else {
      updateComicMutation.mutate({ id: editModal.item.id, data: formData })
    }
  }

  // Format views
  const formatViews = (views) => {
    if (!views) return '0'
    if (views >= 1000000) return `${(views / 1000000).toFixed(1)}M`
    if (views >= 1000) return `${(views / 1000).toFixed(1)}K`
    return views.toString()
  }

  // Format date
  const formatDate = (date) => {
    if (!date) return ''
    return new Date(date).toLocaleDateString('vi-VN')
  }

  return (
    <>
      <MobileHeader title="Quản lý nội dung" />
      
      <div className="container-custom py-8 pt-16 md:pt-8">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
          <h1 className="text-2xl md:text-3xl font-bold text-white">Quản lý nội dung</h1>
          
          <div className="flex items-center gap-3">
            <button
              onClick={handleRefresh}
              disabled={loadingVideos || loadingComics}
              className="inline-flex items-center gap-2 px-4 py-2 bg-white/10 text-white rounded-xl font-medium hover:bg-white/20 transition-all disabled:opacity-50"
            >
              <ArrowPathIcon className={`w-5 h-5 ${loadingVideos || loadingComics ? 'animate-spin' : ''}`} />
              Làm mới
            </button>
            <Link
              to="/upload"
              className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-violet-600 to-fuchsia-600 text-white rounded-xl font-medium hover:shadow-lg hover:shadow-violet-500/25 transition-all"
            >
              <PlusIcon className="w-5 h-5" />
              Upload mới
            </Link>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-2 mb-6">
          <button
            onClick={() => setActiveTab('videos')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl font-medium transition-all ${
              activeTab === 'videos'
                ? 'bg-violet-500/20 text-violet-400 border border-violet-500/30'
                : 'bg-white/5 text-gray-400 border border-white/10 hover:bg-white/10'
            }`}
          >
            <FilmIcon className="w-5 h-5" />
            Videos ({filteredVideos.length})
          </button>
          <button
            onClick={() => setActiveTab('comics')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl font-medium transition-all ${
              activeTab === 'comics'
                ? 'bg-violet-500/20 text-violet-400 border border-violet-500/30'
                : 'bg-white/5 text-gray-400 border border-white/10 hover:bg-white/10'
            }`}
          >
            <BookOpenIcon className="w-5 h-5" />
            Truyện ({filteredComics.length})
          </button>
        </div>

        {/* Search */}
        <div className="relative mb-6">
          <MagnifyingGlassIcon className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
          <input
            type="text"
            placeholder="Tìm kiếm..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="input pl-12"
          />
        </div>

        {/* Content */}
        {activeTab === 'videos' ? (
          <VideosList 
            videos={filteredVideos}
            loading={loadingVideos}
            onEdit={(item) => setEditModal({ open: true, item, type: 'video' })}
            onDelete={(item) => setDeleteModal({ open: true, item, type: 'video' })}
            formatViews={formatViews}
            formatDate={formatDate}
          />
        ) : (
          <ComicsList 
            comics={filteredComics}
            loading={loadingComics}
            onEdit={(item) => setEditModal({ open: true, item, type: 'comic' })}
            onDelete={(item) => setDeleteModal({ open: true, item, type: 'comic' })}
            formatViews={formatViews}
            formatDate={formatDate}
          />
        )}
      </div>

      {/* Delete Confirmation Modal */}
      <DeleteModal
        isOpen={deleteModal.open}
        onClose={() => setDeleteModal({ open: false, item: null, type: null })}
        onConfirm={handleDelete}
        item={deleteModal.item}
        type={deleteModal.type}
        isLoading={deleteVideoMutation.isPending || deleteComicMutation.isPending}
      />

      {/* Edit Modal */}
      <EditModal
        isOpen={editModal.open}
        onClose={() => setEditModal({ open: false, item: null, type: null })}
        onSave={handleEditSave}
        item={editModal.item}
        type={editModal.type}
        isLoading={updateVideoMutation.isPending || updateComicMutation.isPending}
      />
    </>
  )
}

// Videos List Component
const VideosList = ({ videos, loading, onEdit, onDelete, formatViews, formatDate }) => {
  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="skeleton h-24 rounded-xl" />
        ))}
      </div>
    )
  }

  if (!videos.length) {
    return (
      <div className="text-center py-12">
        <FilmIcon className="w-16 h-16 mx-auto text-gray-600 mb-4" />
        <p className="text-gray-400">Bạn chưa có video nào</p>
        <Link to="/upload" className="btn-primary mt-4 inline-block">
          Upload video đầu tiên
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {videos.map((video) => (
        <div
          key={video.id}
          className="flex items-center gap-4 p-4 bg-white/5 rounded-xl border border-white/5 hover:bg-white/10 transition-all"
        >
          {/* Thumbnail */}
          <img
            src={video.thumbnail || '/placeholder-video.png'}
            alt={video.title}
            className="w-24 h-16 md:w-32 md:h-20 object-cover rounded-lg flex-shrink-0"
          />

          {/* Info */}
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-white truncate">{video.title}</h3>
            <div className="flex items-center gap-3 text-sm text-gray-400 mt-1">
              <span className="flex items-center gap-1">
                <EyeIcon className="w-4 h-4" />
                {formatViews(video.views)}
              </span>
              <span>{formatDate(video.created_at)}</span>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <Link
              to={`/videos/${video.id}`}
              className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-all"
              title="Xem"
            >
              <EyeIcon className="w-5 h-5" />
            </Link>
            <button
              onClick={() => onEdit(video)}
              className="p-2 text-gray-400 hover:text-violet-400 hover:bg-violet-500/10 rounded-lg transition-all"
              title="Sửa"
            >
              <PencilSquareIcon className="w-5 h-5" />
            </button>
            <button
              onClick={() => onDelete(video)}
              className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-all"
              title="Xóa"
            >
              <TrashIcon className="w-5 h-5" />
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

// Comics List Component
const ComicsList = ({ comics, loading, onEdit, onDelete, formatViews, formatDate }) => {
  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="skeleton h-24 rounded-xl" />
        ))}
      </div>
    )
  }

  if (!comics.length) {
    return (
      <div className="text-center py-12">
        <BookOpenIcon className="w-16 h-16 mx-auto text-gray-600 mb-4" />
        <p className="text-gray-400">Bạn chưa có truyện nào</p>
        <Link to="/upload" className="btn-primary mt-4 inline-block">
          Upload truyện đầu tiên
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {comics.map((comic) => (
        <div
          key={comic.id}
          className="flex items-center gap-4 p-4 bg-white/5 rounded-xl border border-white/5 hover:bg-white/10 transition-all"
        >
          {/* Cover */}
          <img
            src={comic.cover_image || '/placeholder-comic.png'}
            alt={comic.title}
            className="w-16 h-24 md:w-20 md:h-28 object-cover rounded-lg flex-shrink-0"
          />

          {/* Info */}
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-white truncate">{comic.title}</h3>
            <p className="text-sm text-gray-500 truncate">{comic.description}</p>
            <div className="flex items-center gap-3 text-sm text-gray-400 mt-1">
              <span className="flex items-center gap-1">
                <EyeIcon className="w-4 h-4" />
                {formatViews(comic.views)}
              </span>
              <span>{comic.chapters?.length || 0} ảnh</span>
              <span>{formatDate(comic.created_at)}</span>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <Link
              to={`/comics/${comic.id}`}
              className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-all"
              title="Xem"
            >
              <EyeIcon className="w-5 h-5" />
            </Link>
            <button
              onClick={() => onEdit(comic)}
              className="p-2 text-gray-400 hover:text-violet-400 hover:bg-violet-500/10 rounded-lg transition-all"
              title="Sửa"
            >
              <PencilSquareIcon className="w-5 h-5" />
            </button>
            <button
              onClick={() => onDelete(comic)}
              className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-all"
              title="Xóa"
            >
              <TrashIcon className="w-5 h-5" />
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

// Delete Confirmation Modal
const DeleteModal = ({ isOpen, onClose, onConfirm, item, type, isLoading }) => {
  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" aria-hidden="true" />
      
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <Dialog.Panel className="w-full max-w-md bg-gray-900 rounded-2xl p-6 border border-white/10">
          <div className="flex items-center gap-4 mb-4">
            <div className="w-12 h-12 rounded-full bg-red-500/20 flex items-center justify-center">
              <ExclamationTriangleIcon className="w-6 h-6 text-red-400" />
            </div>
            <div>
              <Dialog.Title className="text-lg font-bold text-white">
                Xác nhận xóa
              </Dialog.Title>
              <p className="text-sm text-gray-400">
                Hành động này không thể hoàn tác
              </p>
            </div>
          </div>

          <p className="text-gray-300 mb-6">
            Bạn có chắc chắn muốn xóa {type === 'video' ? 'video' : 'truyện'}{' '}
            <span className="font-medium text-white">"{item?.title}"</span>?
          </p>

          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="flex-1 btn-secondary"
              disabled={isLoading}
            >
              Hủy
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 btn bg-red-500 hover:bg-red-600 text-white"
              disabled={isLoading}
            >
              {isLoading ? 'Đang xóa...' : 'Xóa'}
            </button>
          </div>
        </Dialog.Panel>
      </div>
    </Dialog>
  )
}

// Edit Modal
const EditModal = ({ isOpen, onClose, onSave, item, type, isLoading }) => {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')

  // Reset form when item changes
  useEffect(() => {
    if (item) {
      setTitle(item.title || '')
      setDescription(item.description || '')
    }
  }, [item])

  const handleSave = () => {
    onSave({ title, description })
  }

  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" aria-hidden="true" />
      
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <Dialog.Panel className="w-full max-w-md bg-gray-900 rounded-2xl p-6 border border-white/10">
          <Dialog.Title className="text-xl font-bold text-white mb-6">
            Chỉnh sửa {type === 'video' ? 'video' : 'truyện'}
          </Dialog.Title>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Tiêu đề
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="input"
                placeholder="Nhập tiêu đề"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Mô tả
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="input min-h-[100px] resize-none"
                placeholder="Nhập mô tả"
                rows={3}
              />
            </div>
          </div>

          <div className="flex gap-3 mt-6">
            <button
              onClick={onClose}
              className="flex-1 btn-secondary"
              disabled={isLoading}
            >
              Hủy
            </button>
            <button
              onClick={handleSave}
              className="flex-1 btn-primary"
              disabled={isLoading || !title.trim()}
            >
              {isLoading ? 'Đang lưu...' : 'Lưu thay đổi'}
            </button>
          </div>
        </Dialog.Panel>
      </div>
    </Dialog>
  )
}

export default Dashboard
