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
  ArrowPathIcon,
  CloudArrowDownIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { userAPI, telegramAPI } from '../services/api'
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

  const { data: profile } = useQuery({
    queryKey: ['user', 'profile'],
    queryFn: () => userAPI.getProfile(),
    select: (res) => res.data,
  })

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

  const removeBookmarkMutation = useMutation({
    mutationFn: (contentId) => userAPI.removeBookmark(contentId),
    onSuccess: () => {
      toast.success('Đã xóa bookmark')
      queryClient.invalidateQueries({ queryKey: ['user', 'profile'] })
    },
    onError: () => toast.error('Xóa bookmark thất bại'),
  })

  const removeLikeMutation = useMutation({
    mutationFn: ({ type, contentId }) => userAPI.removeLike(type, contentId),
    onSuccess: () => {
      toast.success('Đã bỏ thích')
      queryClient.invalidateQueries({ queryKey: ['user', 'profile'] })
    },
    onError: () => toast.error('Bỏ thích thất bại'),
  })

  const clearHistoryMutation = useMutation({
    mutationFn: () => userAPI.clearHistory(),
    onSuccess: () => {
      toast.success('Đã xóa lịch sử')
      queryClient.invalidateQueries({ queryKey: ['user', 'profile'] })
    },
  })

  const { data: telegramStatus, refetch: refetchTelegramStatus } = useQuery({
    queryKey: ['telegram', 'status'],
    queryFn: () => telegramAPI.getStatus(),
    select: (res) => res.data,
    refetchInterval: 30000,
  })

  const syncTelegramMutation = useMutation({
    mutationFn: (params) => telegramAPI.syncChannel(params),
    onSuccess: (res) => {
      const data = res.data
      toast.success(`Đồng bộ thành công! Tìm thấy ${data.new_videos_count} video mới`)
      queryClient.invalidateQueries({ queryKey: ['telegram'] })
    },
    onError: (err) => {
      const message = err.response?.data?.error || 'Đồng bộ thất bại'
      toast.error(message)
    },
  })

  const handleLogout = () => {
    logout()
    navigate('/login')
    toast.success('Đã đăng xuất')
  }

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

  const tabs = [
    { id: 'bookmarks', label: 'Bookmarks', icon: BookmarkIcon },
    { id: 'liked', label: 'Đã thích', icon: HeartIcon },
    { id: 'settings', label: 'Cài đặt', icon: Cog6ToothIcon },
  ]

  return (
    <>
      {/* ── Global styles injected once ── */}
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Sora:wght@400;500;600;700;800&family=DM+Sans:wght@400;500;600&display=swap');

        .profile-root {
          font-family: 'DM Sans', sans-serif;
        }
        .profile-root h1,
        .profile-root h2,
        .profile-root h3 {
          font-family: 'Sora', sans-serif;
        }

        /* ── glow blobs ── */
        .profile-blob {
          position: absolute;
          border-radius: 9999px;
          filter: blur(90px);
          pointer-events: none;
        }

        /* ── gradient text ── */
        .grad-text {
          background: linear-gradient(135deg, #a78bfa 0%, #c084fc 50%, #e879f9 100%);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        /* ── avatar ring ── */
        .avatar-ring {
          background: conic-gradient(
            from 180deg,
            #7c3aed, #a855f7, #ec4899, #7c3aed
          );
          padding: 3px;
          border-radius: 9999px;
          animation: spin-slow 8s linear infinite;
        }
        @keyframes spin-slow {
          to { transform: rotate(360deg); }
        }

        /* ── stat card ── */
        .stat-card {
          background: rgba(255,255,255,0.04);
          border: 1px solid rgba(255,255,255,0.08);
          backdrop-filter: blur(12px);
          border-radius: 16px;
          transition: background 0.2s, transform 0.2s;
        }
        .stat-card:hover {
          background: rgba(168,85,247,0.12);
          transform: translateY(-2px);
        }

        /* ── content card ── */
        .content-card {
          background: rgba(255,255,255,0.03);
          border: 1px solid rgba(255,255,255,0.05);
          border-radius: 16px;
          overflow: hidden;
          transition: all 0.3s ease;
        }
        .content-card:hover {
          background: rgba(255,255,255,0.06);
          border-color: rgba(139,92,246,0.3);
          transform: translateY(-2px);
        }

        /* ── no-scrollbar ── */
        .no-scrollbar::-webkit-scrollbar { display: none; }
        .no-scrollbar { -ms-overflow-style: none; scrollbar-width: none; }

        /* ── settings card ── */
        .settings-section {
          background: rgba(255,255,255,0.03);
          border: 1px solid rgba(255,255,255,0.07);
          border-radius: 20px;
          padding: 24px;
        }

        /* ── input override ── */
        .profile-input {
          width: 100%;
          background: rgba(255,255,255,0.05);
          border: 1px solid rgba(255,255,255,0.10);
          border-radius: 12px;
          padding: 10px 14px;
          color: #fff;
          font-family: 'DM Sans', sans-serif;
          font-size: 14px;
          outline: none;
          transition: border-color 0.2s;
        }
        .profile-input:focus { border-color: #a855f7; }
        .profile-input::placeholder { color: rgba(255,255,255,0.3); }

        /* ── badge ── */
        .verified-badge {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 20px;
          height: 20px;
          background: #3b82f6;
          border-radius: 9999px;
          border: 2px solid #0f0c1b;
          flex-shrink: 0;
        }
      `}</style>

      <div
        className="profile-root min-h-screen"
        style={{
          background: 'linear-gradient(160deg, #0f0c1b 0%, #130e24 50%, #0a0814 100%)',
          color: '#fff',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        {/* Background blobs */}
        <div className="profile-blob" style={{ top: '-8%', left: '-12%', width: '55%', height: '45%', background: 'rgba(99,60,210,0.12)' }} />
        <div className="profile-blob" style={{ top: '35%', right: '-15%', width: '50%', height: '55%', background: 'rgba(192,90,240,0.10)' }} />
        <div className="profile-blob" style={{ bottom: '-10%', left: '20%', width: '50%', height: '40%', background: 'rgba(139,92,246,0.10)' }} />

        <div className="relative z-10 max-w-5xl mx-auto px-4 py-10 sm:px-6 lg:px-8">

          {/* ═══════════════════════════════════════
              PROFILE HEADER CARD
          ═══════════════════════════════════════ */}
          <div
            style={{
              background: 'rgba(255,255,255,0.03)',
              border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: '28px',
              padding: '36px 28px 28px',
              marginBottom: '24px',
              backdropFilter: 'blur(20px)',
              position: 'relative',
              overflow: 'hidden',
            }}
          >
            {/* inner accent line */}
            <div style={{
              position: 'absolute', top: 0, left: '10%', right: '10%', height: '1px',
              background: 'linear-gradient(90deg, transparent, rgba(168,85,247,0.5), transparent)',
            }} />

            <div className="flex flex-col sm:flex-row items-center gap-6 sm:gap-8">
              {/* Avatar */}
              <div className="flex-shrink-0">
                <div className="avatar-ring" style={{ display: 'inline-block' }}>
                  <div style={{
                    width: 96, height: 96, borderRadius: '9999px',
                    border: '3px solid #0f0c1b',
                    overflow: 'hidden',
                    background: '#1e1a2e',
                  }}>
                    {user?.avatar ? (
                      <img src={user.avatar} alt={user.username} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <UserCircleIcon style={{ width: 64, height: 64, color: 'rgba(255,255,255,0.2)' }} />
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Info */}
              <div className="flex-1 text-center sm:text-left">
                {editMode ? (
                  <div style={{ maxWidth: 320 }} className="space-y-3">
                    <input
                      type="text"
                      value={profileForm.username}
                      onChange={(e) => setProfileForm({ ...profileForm, username: e.target.value })}
                      className="profile-input"
                      placeholder="Username"
                    />
                    <input
                      type="url"
                      value={profileForm.avatar}
                      onChange={(e) => setProfileForm({ ...profileForm, avatar: e.target.value })}
                      className="profile-input"
                      placeholder="URL avatar"
                    />
                    <div className="flex gap-3 mt-2">
                      <button onClick={() => updateMutation.mutate(profileForm)} className="px-6 py-2 bg-gradient-to-r from-violet-600 to-fuchsia-600 rounded-full font-semibold shadow-[0_4px_20px_rgba(139,92,246,0.3)] hover:scale-105 transition-all text-sm" disabled={updateMutation.isLoading}>Lưu</button>
                      <button onClick={() => setEditMode(false)} className="px-6 py-2 bg-white/5 border border-white/10 rounded-full font-medium text-gray-300 hover:bg-white/10 transition-all text-sm">Hủy</button>
                    </div>
                  </div>
                ) : (
                  <>
                    <div className="flex items-center justify-center sm:justify-start gap-2 mb-1">
                      <h1 style={{ fontSize: 24, fontWeight: 800, letterSpacing: '-0.3px' }}>{user?.username}</h1>
                      <span className="verified-badge">
                        <svg viewBox="0 0 12 12" fill="none" style={{ width: 9, height: 9 }}>
                          <path d="M2 6l2.5 2.5L10 3" stroke="white" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                        </svg>
                      </span>
                    </div>
                    <p style={{ color: 'rgba(255,255,255,0.45)', fontSize: 13, marginBottom: 4 }}>{user?.email}</p>
                    <p style={{ color: 'rgba(255,255,255,0.25)', fontSize: 12 }}>
                      Tham gia: {formatDate(user?.created_at)}
                    </p>
                  </>
                )}
              </div>

              {/* Stats — desktop row */}
              <div className="flex gap-3 sm:gap-4">
                {[
                  { label: 'Bookmarks', value: profile?.bookmarks?.length ?? 0, icon: '🔖' },
                  { label: 'Playlists', value: profile?.playlists?.length ?? 0, icon: '🎵' },
                ].map((s) => (
                  <div key={s.label} className="stat-card" style={{ padding: '14px 20px', textAlign: 'center', minWidth: 90 }}>
                    <div style={{ fontSize: 20, marginBottom: 4 }}>{s.icon}</div>
                    <div style={{ fontSize: 22, fontWeight: 800, fontFamily: 'Sora, sans-serif' }}>{s.value}</div>
                    <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>{s.label}</div>
                  </div>
                ))}
              </div>
            </div>

            {/* Action buttons row */}
            {!editMode && (
              <div className="flex flex-wrap gap-3 mt-8 justify-center sm:justify-start">
                <button onClick={() => setEditMode(true)} className="px-6 py-2.5 bg-gradient-to-r from-violet-600 to-fuchsia-600 rounded-full font-semibold shadow-[0_4px_20px_rgba(139,92,246,0.3)] hover:scale-105 transition-all flex items-center gap-2 text-sm">
                  ✏️ Chỉnh sửa
                </button>
                <Link to="/dashboard" className="px-6 py-2.5 bg-white/5 border border-white/10 rounded-full font-medium text-gray-300 hover:bg-white/10 transition-all flex items-center gap-2 text-sm">
                  <Squares2X2Icon style={{ width: 18, height: 18 }} /> Dashboard
                </Link>
                <button onClick={handleLogout} className="px-6 py-2.5 bg-red-500/10 border border-red-500/20 text-red-400 rounded-full font-medium hover:bg-red-500/20 transition-all flex items-center gap-2 text-sm">
                  <ArrowRightOnRectangleIcon style={{ width: 18, height: 18 }} /> Đăng xuất
                </button>
              </div>
            )}
          </div>

          {/* ═══════════════════════════════════════
              TAB STRIP
          ═══════════════════════════════════════ */}
          <div className="flex bg-white/[0.03] p-1.5 rounded-full border border-white/5 mb-8 w-fit mx-auto sm:mx-0">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex-1 min-w-[100px] sm:min-w-[120px] px-4 py-2.5 rounded-full font-medium transition-all duration-300 whitespace-nowrap text-sm ${
                  activeTab === tab.id
                    ? 'bg-gradient-to-r from-violet-600 to-fuchsia-600 text-white shadow-lg'
                    : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* ═══════════════════════════════════════
              BOOKMARKS
          ═══════════════════════════════════════ */}
          {activeTab === 'bookmarks' && (
            <div>
              <h2 style={{ fontSize: 18, fontWeight: 700, marginBottom: 16 }} className="grad-text">Đã đánh dấu</h2>
              {profile?.bookmarks?.length > 0 ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {profile.bookmarks.map((bookmark, index) => (
                    <div key={bookmark.content_id || index} className="content-card group">
                      <Link
                        to={bookmark.content_type === 'video' ? `/videos/${bookmark.content_id}` : `/comics/${bookmark.content_id}`}
                        className="block"
                      >
                        <div style={{ position: 'relative', aspectRatio: '16/9', background: 'rgba(255,255,255,0.05)' }}>
                          {bookmark.thumbnail ? (
                            <img
                              src={bookmark.thumbnail}
                              alt={bookmark.title}
                              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                              onError={(e) => { e.target.src = '/placeholder.png' }}
                            />
                          ) : (
                            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                              {bookmark.content_type === 'video'
                                ? <FilmIcon style={{ width: 40, height: 40, color: 'rgba(255,255,255,0.15)' }} />
                                : <BookOpenIcon style={{ width: 40, height: 40, color: 'rgba(255,255,255,0.15)' }} />
                              }
                            </div>
                          )}
                          <span style={{
                            position: 'absolute', top: 10, left: 10,
                            background: 'rgba(124,58,237,0.85)', backdropFilter: 'blur(6px)',
                            borderRadius: 8, padding: '3px 10px', fontSize: 11, fontWeight: 600, color: '#fff',
                          }}>
                            {bookmark.content_type === 'video' ? 'Video' : 'Truyện'}
                          </span>
                        </div>
                      </Link>
                      <div style={{ padding: '14px 16px' }}>
                        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 8 }}>
                          <Link
                            to={bookmark.content_type === 'video' ? `/videos/${bookmark.content_id}` : `/comics/${bookmark.content_id}`}
                            style={{ flex: 1, textDecoration: 'none' }}
                          >
                            <p style={{ color: '#fff', fontWeight: 600, fontSize: 14, lineHeight: '1.4', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
                              {bookmark.title || 'Không có tiêu đề'}
                            </p>
                            <p style={{ color: 'rgba(255,255,255,0.35)', fontSize: 12, marginTop: 4 }}>
                              {formatDate(bookmark.created_at)}
                            </p>
                          </Link>
                          <button
                            onClick={() => removeBookmarkMutation.mutate(bookmark.content_id)}
                            disabled={removeBookmarkMutation.isPending}
                            style={{
                              padding: '7px', borderRadius: 10, border: 'none', cursor: 'pointer',
                              background: 'transparent', color: 'rgba(255,255,255,0.3)',
                              transition: 'color 0.2s, background 0.2s', flexShrink: 0,
                            }}
                            onMouseEnter={e => { e.currentTarget.style.color = '#f87171'; e.currentTarget.style.background = 'rgba(239,68,68,0.12)' }}
                            onMouseLeave={e => { e.currentTarget.style.color = 'rgba(255,255,255,0.3)'; e.currentTarget.style.background = 'transparent' }}
                          >
                            <TrashIcon style={{ width: 17, height: 17 }} />
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState icon="🔖" title="Chưa có bookmark nào" desc="Đánh dấu video hoặc truyện để xem sau" />
              )}
            </div>
          )}

          {/* ═══════════════════════════════════════
              LIKED
          ═══════════════════════════════════════ */}
          {activeTab === 'liked' && (
            <div>
              <h2 style={{ fontSize: 18, fontWeight: 700, marginBottom: 16 }} className="grad-text">Đã thích</h2>
              {profile?.liked?.length > 0 ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {profile.liked.map((item, index) => (
                    <div key={item.id || index} className="content-card group">
                      <Link
                        to={item.content_type === 'video' ? `/videos/${item.id}` : `/comics/${item.id}`}
                        className="block"
                      >
                        <div style={{ position: 'relative', aspectRatio: '16/9', background: 'rgba(255,255,255,0.05)' }}>
                          {item.thumbnail ? (
                            <img
                              src={item.thumbnail}
                              alt={item.title}
                              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                              onError={(e) => { e.target.src = '/placeholder.png' }}
                            />
                          ) : (
                            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                              {item.content_type === 'video'
                                ? <FilmIcon style={{ width: 40, height: 40, color: 'rgba(255,255,255,0.15)' }} />
                                : <BookOpenIcon style={{ width: 40, height: 40, color: 'rgba(255,255,255,0.15)' }} />
                              }
                            </div>
                          )}
                          <span style={{
                            position: 'absolute', top: 10, left: 10,
                            background: 'rgba(236,72,153,0.8)', backdropFilter: 'blur(6px)',
                            borderRadius: 8, padding: '3px 10px', fontSize: 11, fontWeight: 600, color: '#fff',
                          }}>
                            {item.content_type === 'video' ? 'Video' : 'Truyện'}
                          </span>
                        </div>
                      </Link>
                      <div style={{ padding: '14px 16px' }}>
                        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 8 }}>
                          <Link
                            to={item.content_type === 'video' ? `/videos/${item.id}` : `/comics/${item.id}`}
                            style={{ flex: 1, textDecoration: 'none' }}
                          >
                            <p style={{ color: '#fff', fontWeight: 600, fontSize: 14, lineHeight: '1.4', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
                              {item.title || 'Không có tiêu đề'}
                            </p>
                            {item.content_type === 'video' && item.duration && (
                              <p style={{ color: 'rgba(255,255,255,0.4)', fontSize: 12, marginTop: 4 }}>{formatDuration(item.duration)}</p>
                            )}
                            {item.views !== undefined && (
                              <p style={{ color: 'rgba(255,255,255,0.25)', fontSize: 12, marginTop: 2 }}>{item.views} lượt xem</p>
                            )}
                          </Link>
                          <button
                            onClick={() => removeLikeMutation.mutate({ type: item.content_type, contentId: item.id })}
                            disabled={removeLikeMutation.isPending}
                            style={{
                              padding: '7px', borderRadius: 10, border: 'none', cursor: 'pointer',
                              background: 'transparent', color: 'rgba(255,255,255,0.3)',
                              transition: 'color 0.2s, background 0.2s', flexShrink: 0,
                            }}
                            onMouseEnter={e => { e.currentTarget.style.color = '#f87171'; e.currentTarget.style.background = 'rgba(239,68,68,0.12)' }}
                            onMouseLeave={e => { e.currentTarget.style.color = 'rgba(255,255,255,0.3)'; e.currentTarget.style.background = 'transparent' }}
                          >
                            <HeartIcon style={{ width: 17, height: 17 }} />
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState icon="❤️" title="Chưa thích nội dung nào" desc="Thích video hoặc truyện để xem lại sau" />
              )}
            </div>
          )}

          {/* ═══════════════════════════════════════
              SETTINGS
          ═══════════════════════════════════════ */}
          {activeTab === 'settings' && (
            <div style={{ maxWidth: 600 }}>
              <h2 style={{ fontSize: 18, fontWeight: 700, marginBottom: 20 }} className="grad-text">Cài đặt tài khoản</h2>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>

                {/* Telegram Sync */}
                <div className="settings-section">
                  <h3 style={{ fontSize: 15, fontWeight: 700, marginBottom: 4, display: 'flex', alignItems: 'center', gap: 8 }}>
                    <CloudArrowDownIcon style={{ width: 18, height: 18, color: '#60a5fa' }} />
                    Đồng bộ Telegram
                  </h3>
                  <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', marginBottom: 16, lineHeight: '1.6' }}>
                    Quét và tự động thêm video từ kênh Telegram vào thư viện.
                  </p>

                  <div style={{
                    background: 'rgba(255,255,255,0.03)', borderRadius: 14, border: '1px solid rgba(255,255,255,0.06)',
                    padding: '12px 16px', marginBottom: 14, display: 'flex', alignItems: 'center', gap: 10,
                  }}>
                    {telegramStatus?.telegram_connected
                      ? <CheckCircleIcon style={{ width: 20, height: 20, color: '#4ade80', flexShrink: 0 }} />
                      : <ExclamationCircleIcon style={{ width: 20, height: 20, color: '#facc15', flexShrink: 0 }} />
                    }
                    <div>
                      <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.7)' }}>{telegramStatus?.message || 'Đang kiểm tra...'}</p>
                      <p style={{ fontSize: 12, color: telegramStatus?.telegram_connected ? '#4ade80' : '#facc15', marginTop: 2 }}>
                        {telegramStatus?.telegram_connected ? 'Đã kết nối' : 'Chưa kết nối'}
                      </p>
                    </div>
                  </div>

                  <div className="flex gap-3 mt-4">
                    <button
                      onClick={() => syncTelegramMutation.mutate({})}
                      disabled={syncTelegramMutation.isPending || !telegramStatus?.telegram_connected}
                      className="px-6 py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded-full font-medium inline-flex items-center gap-2 shadow-[0_4px_16px_rgba(59,130,246,0.25)] hover:scale-105 transition-all disabled:opacity-50 disabled:hover:scale-100 disabled:cursor-not-allowed text-sm"
                    >
                      <ArrowPathIcon style={{ width: 18, height: 18 }} className={syncTelegramMutation.isPending ? 'animate-spin' : ''} />
                      {syncTelegramMutation.isPending ? 'Đang đồng bộ...' : 'Đồng bộ ngay'}
                    </button>
                    <button onClick={() => refetchTelegramStatus()} className="px-6 py-2.5 bg-white/5 border border-white/10 text-gray-300 rounded-full font-medium hover:bg-white/10 transition-all text-sm">
                      Kiểm tra
                    </button>
                  </div>

                  {syncTelegramMutation.isSuccess && (
                    <div style={{
                      marginTop: 14, padding: '12px 16px',
                      background: 'rgba(74,222,128,0.08)', border: '1px solid rgba(74,222,128,0.2)',
                      borderRadius: 12, fontSize: 13,
                    }}>
                      <p style={{ color: '#4ade80' }}>✓ Đồng bộ thành công! Đã thêm {syncTelegramMutation.data?.data?.new_videos_count || 0} video mới.</p>
                      {syncTelegramMutation.data?.data?.skipped_count > 0 && (
                        <p style={{ color: 'rgba(255,255,255,0.4)', marginTop: 4, fontSize: 12 }}>
                          Bỏ qua: {syncTelegramMutation.data?.data?.skipped_count} video đã có
                        </p>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

/* ── Reusable empty state ── */
const EmptyState = ({ icon, title, desc }) => (
  <div style={{ textAlign: 'center', padding: '64px 0' }}>
    <div style={{
      width: 80, height: 80, borderRadius: '24px',
      background: 'rgba(255,255,255,0.04)',
      border: '1px solid rgba(255,255,255,0.08)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: 32, margin: '0 auto 20px',
    }}>
      {icon}
    </div>
    <p style={{ color: 'rgba(255,255,255,0.6)', fontWeight: 600, fontSize: 15, marginBottom: 6 }}>{title}</p>
    <p style={{ color: 'rgba(255,255,255,0.3)', fontSize: 13 }}>{desc}</p>
  </div>
)

export default Profile