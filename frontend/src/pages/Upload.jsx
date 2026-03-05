import { useState, useCallback, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  CloudArrowUpIcon, 
  FilmIcon, 
  BookOpenIcon,
  PhotoIcon,
  XMarkIcon,
  CheckCircleIcon,
  ArrowPathIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import MobileHeader from '../components/MobileHeader'
import { videosAPI, comicsAPI } from '../services/api'

// Upload page - Upload video hoặc truyện
const Upload = () => {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [uploadType, setUploadType] = useState('video') // video | comic
  const [dragActive, setDragActive] = useState(false)
  const [progress, setProgress] = useState(0)
  const [uploadStatus, setUploadStatus] = useState(null) // null | 'uploading' | 'success' | 'error'
  const [cooldown, setCooldown] = useState(0)
  const cooldownRef = useRef(null)
  const fileInputRef = useRef(null)

  // Video upload form
  const [videoForm, setVideoForm] = useState({
    title: '',
    description: '',
    file: null,
  })

  // Comic upload form - Đơn giản hóa: chỉ title, description, images
  const [comicForm, setComicForm] = useState({
    title: '',
    description: 'Truyện hành động', // Mặc định
    images: [], // Multiple images
  })

  // Cooldown timer
  useEffect(() => {
    if (cooldown > 0) {
      cooldownRef.current = setTimeout(() => {
        setCooldown(cooldown - 1)
      }, 1000)
    }
    return () => clearTimeout(cooldownRef.current)
  }, [cooldown])

  // Video upload mutation
  const videoMutation = useMutation({
    mutationFn: (formData) => videosAPI.upload(formData, (p) => setProgress(p)),
    onMutate: () => {
      setUploadStatus('uploading')
      setProgress(0)
    },
    onSuccess: (res) => {
      setUploadStatus('success')
      setProgress(100)
      toast.success('Upload video thành công!')
      setCooldown(30) // 30s cooldown
      
      // Invalidate queries to refresh dashboard and lists
      queryClient.invalidateQueries({ queryKey: ['my-videos'] })
      queryClient.invalidateQueries({ queryKey: ['videos'] })
      
      // Reset form
      setVideoForm({ title: '', description: '', file: null })
      
      // Navigate after delay
      setTimeout(() => {
        if (res.data?.video?.id) {
          navigate(`/videos/${res.data.video.id}`)
        } else {
          navigate('/dashboard')
        }
      }, 2000)
    },
    onError: (error) => {
      setUploadStatus('error')
      toast.error(error.response?.data?.error || 'Upload thất bại')
    },
  })

  // Comic upload mutation - Upload comic với ảnh
  const comicMutation = useMutation({
    mutationFn: (formData) => comicsAPI.uploadWithImages(formData, (p) => setProgress(p)),
    onMutate: () => {
      setUploadStatus('uploading')
      setProgress(0)
    },
    onSuccess: (res) => {
      setUploadStatus('success')
      setProgress(100)
      toast.success('Upload truyện thành công!')
      setCooldown(30) // 30s cooldown
      
      // Invalidate queries to refresh dashboard and lists
      queryClient.invalidateQueries({ queryKey: ['my-comics'] })
      queryClient.invalidateQueries({ queryKey: ['comics'] })
      
      // Reset form
      setComicForm({ title: '', description: 'Truyện hành động', images: [] })
      
      // Navigate after delay
      setTimeout(() => {
        if (res.data?.comic?.id) {
          navigate(`/comics/${res.data.comic.id}`)
        } else {
          navigate('/dashboard')
        }
      }, 2000)
    },
    onError: (error) => {
      setUploadStatus('error')
      toast.error(error.response?.data?.error || 'Upload thất bại')
    },
  })

  // Check if can submit (not in cooldown)
  const canSubmit = cooldown === 0 && uploadStatus !== 'uploading'

  // Handle drag and drop
  const handleDrag = useCallback((e) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }, [])

  // Handle video file drop
  const handleVideoDrop = useCallback((e) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    
    const file = e.dataTransfer?.files?.[0]
    if (file && file.type.startsWith('video/')) {
      setVideoForm(prev => ({ ...prev, file }))
    } else {
      toast.error('Vui lòng chọn file video')
    }
  }, [])

  // Handle comic images drop
  const handleImagesDrop = useCallback((e) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    
    const files = Array.from(e.dataTransfer?.files || [])
    const imageFiles = files.filter(f => f.type.startsWith('image/'))
    
    if (imageFiles.length > 0) {
      setComicForm(prev => ({ 
        ...prev, 
        images: [...prev.images, ...imageFiles] 
      }))
    } else {
      toast.error('Vui lòng chọn file ảnh')
    }
  }, [])

  // Handle video file select
  const handleVideoSelect = (e) => {
    const file = e.target.files?.[0]
    if (file) {
      setVideoForm(prev => ({ ...prev, file }))
    }
  }

  // Handle comic images select
  const handleImagesSelect = (e) => {
    const files = Array.from(e.target.files || [])
    const imageFiles = files.filter(f => f.type.startsWith('image/'))
    
    if (imageFiles.length > 0) {
      setComicForm(prev => ({ 
        ...prev, 
        images: [...prev.images, ...imageFiles] 
      }))
    }
  }

  // Remove image from comic
  const removeImage = (index) => {
    setComicForm(prev => ({
      ...prev,
      images: prev.images.filter((_, i) => i !== index)
    }))
  }

  // Submit video
  const handleVideoSubmit = (e) => {
    e.preventDefault()
    
    if (!canSubmit) {
      toast.error(`Vui lòng đợi ${cooldown}s`)
      return
    }
    
    if (!videoForm.title.trim()) {
      toast.error('Vui lòng nhập tiêu đề')
      return
    }
    if (!videoForm.file) {
      toast.error('Vui lòng chọn video')
      return
    }

    const formData = new FormData()
    formData.append('title', videoForm.title)
    formData.append('description', videoForm.description)
    formData.append('video', videoForm.file)

    videoMutation.mutate(formData)
  }

  // Submit comic
  const handleComicSubmit = (e) => {
    e.preventDefault()
    
    if (!canSubmit) {
      toast.error(`Vui lòng đợi ${cooldown}s`)
      return
    }
    
    if (!comicForm.title.trim()) {
      toast.error('Vui lòng nhập tiêu đề')
      return
    }
    if (comicForm.images.length === 0) {
      toast.error('Vui lòng chọn ít nhất 1 ảnh')
      return
    }

    const formData = new FormData()
    formData.append('title', comicForm.title)
    formData.append('description', comicForm.description || 'Truyện hành động')
    
    // Append all images
    comicForm.images.forEach((img) => {
      formData.append('images', img)
    })

    comicMutation.mutate(formData)
  }

  // Format file size
  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  return (
    <>
      <MobileHeader title="Upload" />
      
      <div className="container-custom py-8 pt-16 md:pt-8 max-w-2xl mx-auto">
        <h1 className="text-2xl md:text-3xl font-bold text-white mb-6">Upload nội dung</h1>

        {/* Progress Bar - Luôn hiển thị khi đang upload hoặc đã upload xong */}
        {(uploadStatus === 'uploading' || uploadStatus === 'success') && (
          <div className={`mb-6 p-4 rounded-xl border ${
            uploadStatus === 'success' 
              ? 'bg-emerald-500/10 border-emerald-500/30' 
              : 'bg-white/5 border-white/10'
          }`}>
            <div className="flex items-center justify-between mb-2">
              <span className={`text-sm ${uploadStatus === 'success' ? 'text-emerald-400' : 'text-gray-400'}`}>
                {uploadStatus === 'success' ? 'Upload hoàn tất!' : 'Đang upload...'}
              </span>
              <span className={`text-sm font-medium ${uploadStatus === 'success' ? 'text-emerald-400' : 'text-violet-400'}`}>
                {progress}%
              </span>
            </div>
            <div className="w-full h-3 bg-gray-700 rounded-full overflow-hidden">
              <div 
                className={`h-full transition-all duration-300 ${
                  uploadStatus === 'success' 
                    ? 'bg-gradient-to-r from-emerald-500 to-green-500' 
                    : 'bg-gradient-to-r from-violet-500 to-fuchsia-500'
                }`}
                style={{ width: `${progress}%` }}
              />
            </div>
            {uploadStatus === 'success' && (
              <p className="text-emerald-300 text-sm mt-2 flex items-center gap-2">
                <CheckCircleIcon className="w-5 h-5" />
                Đang chuyển hướng...
              </p>
            )}
          </div>
        )}

        {/* Error message */}
        {uploadStatus === 'error' && (
          <div className="mb-6 p-4 bg-red-500/10 rounded-xl border border-red-500/30 flex items-center gap-3">
            <XMarkIcon className="w-6 h-6 text-red-400" />
            <span className="text-red-300">Upload thất bại! Vui lòng thử lại.</span>
          </div>
        )}

        {/* Cooldown warning */}
        {cooldown > 0 && uploadStatus !== 'uploading' && uploadStatus !== 'success' && (
          <div className="mb-6 p-4 bg-amber-500/10 rounded-xl border border-amber-500/30 flex items-center gap-3">
            <ArrowPathIcon className="w-6 h-6 text-amber-400 animate-spin" />
            <span className="text-amber-300">
              Vui lòng đợi {cooldown}s trước khi upload tiếp
            </span>
          </div>
        )}

        {/* Type selector */}
        <div className="flex gap-3 mb-6">
          <button
            onClick={() => setUploadType('video')}
            disabled={uploadStatus === 'uploading'}
            className={`flex-1 flex items-center justify-center gap-2 py-4 rounded-xl border-2 transition-all ${
              uploadType === 'video'
                ? 'bg-violet-500/20 border-violet-500 text-violet-300'
                : 'bg-white/5 border-white/10 text-gray-400 hover:border-white/20'
            } ${uploadStatus === 'uploading' ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <FilmIcon className="w-6 h-6" />
            <span className="font-medium">Video</span>
          </button>
          <button
            onClick={() => setUploadType('comic')}
            disabled={uploadStatus === 'uploading'}
            className={`flex-1 flex items-center justify-center gap-2 py-4 rounded-xl border-2 transition-all ${
              uploadType === 'comic'
                ? 'bg-violet-500/20 border-violet-500 text-violet-300'
                : 'bg-white/5 border-white/10 text-gray-400 hover:border-white/20'
            } ${uploadStatus === 'uploading' ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <BookOpenIcon className="w-6 h-6" />
            <span className="font-medium">Truyện</span>
          </button>
        </div>

        {/* Video Upload Form */}
        {uploadType === 'video' && (
          <form onSubmit={handleVideoSubmit} className="space-y-6">
            {/* Title */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Tiêu đề video <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                value={videoForm.title}
                onChange={(e) => setVideoForm(prev => ({ ...prev, title: e.target.value }))}
                className="input"
                placeholder="Nhập tiêu đề video"
                disabled={uploadStatus === 'uploading'}
              />
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Mô tả
              </label>
              <textarea
                value={videoForm.description}
                onChange={(e) => setVideoForm(prev => ({ ...prev, description: e.target.value }))}
                className="input min-h-[100px] resize-none"
                placeholder="Nhập mô tả video"
                rows={3}
                disabled={uploadStatus === 'uploading'}
              />
            </div>

            {/* Video file drop zone */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                File video <span className="text-red-400">*</span>
              </label>
              
              {!videoForm.file ? (
                <div
                  onDragEnter={handleDrag}
                  onDragLeave={handleDrag}
                  onDragOver={handleDrag}
                  onDrop={handleVideoDrop}
                  className={`
                    relative border-2 border-dashed rounded-xl p-8 text-center transition-all cursor-pointer
                    ${dragActive 
                      ? 'border-violet-500 bg-violet-500/10' 
                      : 'border-white/20 hover:border-white/40'
                    }
                    ${uploadStatus === 'uploading' ? 'opacity-50 pointer-events-none' : ''}
                  `}
                  onClick={() => fileInputRef.current?.click()}
                >
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="video/*"
                    onChange={handleVideoSelect}
                    className="hidden"
                    disabled={uploadStatus === 'uploading'}
                  />
                  <CloudArrowUpIcon className="w-12 h-12 mx-auto text-gray-500 mb-4" />
                  <p className="text-gray-300 mb-2">
                    Kéo thả video vào đây hoặc <span className="text-violet-400">chọn file</span>
                  </p>
                  <p className="text-sm text-gray-500">
                    MP4, WebM, MKV (tối đa 2GB)
                  </p>
                </div>
              ) : (
                <div className="flex items-center gap-4 p-4 bg-white/5 rounded-xl border border-white/10">
                  <FilmIcon className="w-10 h-10 text-violet-400" />
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-white truncate">{videoForm.file.name}</p>
                    <p className="text-sm text-gray-400">{formatFileSize(videoForm.file.size)}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setVideoForm(prev => ({ ...prev, file: null }))}
                    className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg"
                    disabled={uploadStatus === 'uploading'}
                  >
                    <XMarkIcon className="w-5 h-5" />
                  </button>
                </div>
              )}
            </div>

            {/* Submit */}
            <button
              type="submit"
              disabled={!canSubmit || !videoForm.title || !videoForm.file}
              className="w-full btn-primary py-3 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {uploadStatus === 'uploading' ? (
                <span className="flex items-center justify-center gap-2">
                  <ArrowPathIcon className="w-5 h-5 animate-spin" />
                  Đang upload... {progress}%
                </span>
              ) : cooldown > 0 ? (
                `Đợi ${cooldown}s`
              ) : (
                'Upload video'
              )}
            </button>
          </form>
        )}

        {/* Comic Upload Form - Đơn giản hóa */}
        {uploadType === 'comic' && (
          <form onSubmit={handleComicSubmit} className="space-y-6">
            {/* Title */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Tên truyện <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                value={comicForm.title}
                onChange={(e) => setComicForm(prev => ({ ...prev, title: e.target.value }))}
                className="input"
                placeholder="Nhập tên truyện"
                disabled={uploadStatus === 'uploading'}
              />
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Mô tả
              </label>
              <textarea
                value={comicForm.description}
                onChange={(e) => setComicForm(prev => ({ ...prev, description: e.target.value }))}
                className="input min-h-[80px] resize-none"
                placeholder="Mặc định: Truyện hành động"
                rows={2}
                disabled={uploadStatus === 'uploading'}
              />
            </div>

            {/* Images drop zone */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Ảnh truyện <span className="text-red-400">*</span>
                <span className="text-gray-500 font-normal ml-2">
                  ({comicForm.images.length} ảnh đã chọn)
                </span>
              </label>
              
              <div
                onDragEnter={handleDrag}
                onDragLeave={handleDrag}
                onDragOver={handleDrag}
                onDrop={handleImagesDrop}
                className={`
                  relative border-2 border-dashed rounded-xl p-6 text-center transition-all cursor-pointer mb-4
                  ${dragActive 
                    ? 'border-violet-500 bg-violet-500/10' 
                    : 'border-white/20 hover:border-white/40'
                  }
                  ${uploadStatus === 'uploading' ? 'opacity-50 pointer-events-none' : ''}
                `}
                onClick={() => document.getElementById('comic-images-input')?.click()}
              >
                <input
                  id="comic-images-input"
                  type="file"
                  accept="image/*"
                  multiple
                  onChange={handleImagesSelect}
                  className="hidden"
                  disabled={uploadStatus === 'uploading'}
                />
                <PhotoIcon className="w-10 h-10 mx-auto text-gray-500 mb-3" />
                <p className="text-gray-300 text-sm mb-1">
                  Kéo thả nhiều ảnh vào đây hoặc <span className="text-violet-400">chọn files</span>
                </p>
                <p className="text-xs text-gray-500">
                  JPG, PNG, WebP (chọn nhiều ảnh cùng lúc)
                </p>
              </div>

              {/* Preview images */}
              {comicForm.images.length > 0 && (
                <div className="grid grid-cols-4 sm:grid-cols-5 md:grid-cols-6 gap-2">
                  {comicForm.images.map((img, index) => (
                    <div key={index} className="relative aspect-[2/3] group">
                      <img
                        src={URL.createObjectURL(img)}
                        alt={`Page ${index + 1}`}
                        className="w-full h-full object-cover rounded-lg"
                      />
                      <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity rounded-lg flex items-center justify-center">
                        <button
                          type="button"
                          onClick={() => removeImage(index)}
                          className="p-1.5 bg-red-500 rounded-full text-white"
                          disabled={uploadStatus === 'uploading'}
                        >
                          <XMarkIcon className="w-4 h-4" />
                        </button>
                      </div>
                      <span className="absolute bottom-1 left-1 text-xs bg-black/70 text-white px-1.5 py-0.5 rounded">
                        {index + 1}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Submit */}
            <button
              type="submit"
              disabled={!canSubmit || !comicForm.title || comicForm.images.length === 0}
              className="w-full btn-primary py-3 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {uploadStatus === 'uploading' ? (
                <span className="flex items-center justify-center gap-2">
                  <ArrowPathIcon className="w-5 h-5 animate-spin" />
                  Đang upload... {progress}%
                </span>
              ) : cooldown > 0 ? (
                `Đợi ${cooldown}s`
              ) : (
                `Upload truyện (${comicForm.images.length} ảnh)`
              )}
            </button>
          </form>
        )}
      </div>
    </>
  )
}

export default Upload
