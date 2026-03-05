import { useRef, useState, useEffect, useCallback } from 'react'
import ReactPlayer from 'react-player'
import {
  PlayIcon,
  PauseIcon,
  SpeakerWaveIcon,
  SpeakerXMarkIcon,
  ArrowsPointingOutIcon,
  ArrowsPointingInIcon,
  Cog6ToothIcon,
  ForwardIcon,
  BackwardIcon,
} from '@heroicons/react/24/solid'
import { usePlayerStore } from '../store/playerStore'

// Player component với resume playback và controls đầy đủ
const Player = ({ url, videoId, poster, onProgress: onProgressCallback }) => {
  const playerRef = useRef(null)
  const containerRef = useRef(null)
  
  const [playing, setPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)
  const [seeking, setSeeking] = useState(false)
  const [fullscreen, setFullscreen] = useState(false)
  const [showControls, setShowControls] = useState(true)
  const [showSettings, setShowSettings] = useState(false)
  const [buffered, setBuffered] = useState(0)
  
  // Double-tap state
  const [doubleTapSide, setDoubleTapSide] = useState(null) // 'left' | 'right' | null
  const lastTapRef = useRef({ time: 0, side: null })

  const { 
    volume, 
    muted, 
    playbackRate,
    setVolume, 
    toggleMute,
    setPlaybackRate,
    saveVideoProgress, 
    getVideoProgress 
  } = usePlayerStore()

  // Resume playback từ saved progress
  useEffect(() => {
    const savedProgress = getVideoProgress(videoId)
    if (savedProgress && playerRef.current) {
      // Seek đến vị trí đã lưu
      playerRef.current.seekTo(savedProgress.currentTime, 'seconds')
    }
  }, [videoId, getVideoProgress])

  // Auto hide controls
  useEffect(() => {
    let timeout
    if (playing && showControls) {
      timeout = setTimeout(() => setShowControls(false), 3000)
    }
    return () => clearTimeout(timeout)
  }, [playing, showControls])

  // Handle progress
  const handleProgress = useCallback((state) => {
    if (!seeking) {
      setProgress(state.playedSeconds)
      setBuffered(state.loadedSeconds)
      
      // Save progress mỗi 5 giây
      if (Math.floor(state.playedSeconds) % 5 === 0) {
        saveVideoProgress(videoId, state.playedSeconds, duration)
      }
      
      onProgressCallback?.(state)
    }
  }, [seeking, videoId, duration, saveVideoProgress, onProgressCallback])

  // Handle seek
  const handleSeek = (e) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - rect.left
    const percent = x / rect.width
    const seekTime = percent * duration
    playerRef.current?.seekTo(seekTime, 'seconds')
  }

  // Skip forward/backward
  const skip = (seconds) => {
    const newTime = Math.max(0, Math.min(duration, progress + seconds))
    playerRef.current?.seekTo(newTime, 'seconds')
  }

  // Handle double tap to seek
  const handleDoubleTap = (e, side) => {
    const now = Date.now()
    const lastTap = lastTapRef.current
    
    // Check if double tap (within 300ms)
    if (now - lastTap.time < 300 && lastTap.side === side) {
      // Double tap detected
      const seekAmount = side === 'left' ? -10 : 10
      skip(seekAmount)
      
      // Show visual feedback
      setDoubleTapSide(side)
      setTimeout(() => setDoubleTapSide(null), 500)
      
      lastTapRef.current = { time: 0, side: null }
    } else {
      lastTapRef.current = { time: now, side }
    }
  }

  // Toggle fullscreen
  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      containerRef.current?.requestFullscreen()
      setFullscreen(true)
    } else {
      document.exitFullscreen()
      setFullscreen(false)
    }
  }

  // Format time
  const formatTime = (seconds) => {
    if (!seconds || isNaN(seconds)) return '0:00'
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    if (mins >= 60) {
      const hours = Math.floor(mins / 60)
      const remainMins = mins % 60
      return `${hours}:${remainMins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const playbackRates = [0.5, 0.75, 1, 1.25, 1.5, 2]

  return (
    <div 
      ref={containerRef}
      className="relative bg-black rounded-lg overflow-hidden group"
      onMouseMove={() => setShowControls(true)}
      onMouseLeave={() => playing && setShowControls(false)}
    >
      {/* React Player */}
      <ReactPlayer
        ref={playerRef}
        url={url}
        playing={playing}
        volume={volume}
        muted={muted}
        playbackRate={playbackRate}
        width="100%"
        height="100%"
        className="aspect-video"
        onDuration={setDuration}
        onProgress={handleProgress}
        onEnded={() => setPlaying(false)}
        config={{
          file: {
            attributes: {
              poster: poster,
            }
          }
        }}
      />

      {/* Double-tap zones for seeking */}
      <div className="absolute inset-0 flex pointer-events-auto">
        {/* Left zone - tap to go back 10s */}
        <div 
          className="w-1/2 h-full cursor-pointer"
          onClick={(e) => handleDoubleTap(e, 'left')}
        >
          {doubleTapSide === 'left' && (
            <div className="absolute left-1/4 top-1/2 -translate-x-1/2 -translate-y-1/2 flex items-center justify-center animate-pulse">
              <div className="bg-black/60 backdrop-blur-sm rounded-full p-4">
                <BackwardIcon className="w-8 h-8 text-white" />
              </div>
              <span className="absolute -bottom-8 text-white text-sm font-medium">-10s</span>
            </div>
          )}
        </div>
        
        {/* Right zone - tap to go forward 10s */}
        <div 
          className="w-1/2 h-full cursor-pointer"
          onClick={(e) => handleDoubleTap(e, 'right')}
        >
          {doubleTapSide === 'right' && (
            <div className="absolute right-1/4 top-1/2 translate-x-1/2 -translate-y-1/2 flex items-center justify-center animate-pulse">
              <div className="bg-black/60 backdrop-blur-sm rounded-full p-4">
                <ForwardIcon className="w-8 h-8 text-white" />
              </div>
              <span className="absolute -bottom-8 text-white text-sm font-medium">+10s</span>
            </div>
          )}
        </div>
      </div>

      {/* Play button overlay */}
      {!playing && (
        <div 
          className="absolute inset-0 flex items-center justify-center cursor-pointer"
          onClick={() => setPlaying(true)}
        >
          <div className="w-20 h-20 bg-white/90 rounded-full flex items-center justify-center hover:scale-110 transition-transform">
            <PlayIcon className="w-10 h-10 text-gray-900 ml-1" />
          </div>
        </div>
      )}

      {/* Controls */}
      <div 
        className={`absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/90 to-transparent p-4 transition-opacity duration-300 ${
          showControls ? 'opacity-100' : 'opacity-0'
        }`}
      >
        {/* Progress bar */}
        <div 
          className="relative h-1 bg-gray-600 rounded cursor-pointer mb-3 group/progress"
          onClick={handleSeek}
          onMouseDown={() => setSeeking(true)}
          onMouseUp={() => setSeeking(false)}
        >
          {/* Buffered */}
          <div 
            className="absolute h-full bg-gray-500 rounded"
            style={{ width: `${(buffered / duration) * 100}%` }}
          />
          {/* Played */}
          <div 
            className="absolute h-full bg-primary-500 rounded"
            style={{ width: `${(progress / duration) * 100}%` }}
          >
            <div className="absolute right-0 top-1/2 -translate-y-1/2 w-3 h-3 bg-white rounded-full opacity-0 group-hover/progress:opacity-100 transition-opacity" />
          </div>
        </div>

        {/* Controls row */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            {/* Play/Pause */}
            <button 
              onClick={() => setPlaying(!playing)}
              className="p-1 text-white hover:text-primary-400 transition-colors"
            >
              {playing ? (
                <PauseIcon className="w-6 h-6" />
              ) : (
                <PlayIcon className="w-6 h-6" />
              )}
            </button>

            {/* Skip buttons */}
            <button 
              onClick={() => skip(-10)}
              className="p-1 text-white hover:text-primary-400 transition-colors"
              title="Lùi 10s"
            >
              <BackwardIcon className="w-5 h-5" />
            </button>
            <button 
              onClick={() => skip(10)}
              className="p-1 text-white hover:text-primary-400 transition-colors"
              title="Tiến 10s"
            >
              <ForwardIcon className="w-5 h-5" />
            </button>

            {/* Volume */}
            <div className="flex items-center space-x-1 group/volume">
              <button 
                onClick={toggleMute}
                className="p-1 text-white hover:text-primary-400 transition-colors"
              >
                {muted || volume === 0 ? (
                  <SpeakerXMarkIcon className="w-5 h-5" />
                ) : (
                  <SpeakerWaveIcon className="w-5 h-5" />
                )}
              </button>
              <input
                type="range"
                min="0"
                max="1"
                step="0.1"
                value={muted ? 0 : volume}
                onChange={(e) => setVolume(parseFloat(e.target.value))}
                className="w-0 group-hover/volume:w-20 transition-all duration-200 accent-primary-500"
              />
            </div>

            {/* Time */}
            <span className="text-white text-sm">
              {formatTime(progress)} / {formatTime(duration)}
            </span>
          </div>

          <div className="flex items-center space-x-3">
            {/* Settings */}
            <div className="relative">
              <button 
                onClick={() => setShowSettings(!showSettings)}
                className="p-1 text-white hover:text-primary-400 transition-colors"
              >
                <Cog6ToothIcon className="w-5 h-5" />
              </button>
              
              {showSettings && (
                <div className="absolute bottom-full right-0 mb-2 bg-gray-800 rounded-lg shadow-lg py-2 min-w-[120px]">
                  <div className="px-3 py-1 text-xs text-gray-400">Tốc độ phát</div>
                  {playbackRates.map((rate) => (
                    <button
                      key={rate}
                      onClick={() => {
                        setPlaybackRate(rate)
                        setShowSettings(false)
                      }}
                      className={`w-full px-3 py-1 text-left text-sm hover:bg-gray-700 ${
                        playbackRate === rate ? 'text-primary-400' : 'text-white'
                      }`}
                    >
                      {rate}x
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Fullscreen */}
            <button 
              onClick={toggleFullscreen}
              className="p-1 text-white hover:text-primary-400 transition-colors"
            >
              {fullscreen ? (
                <ArrowsPointingInIcon className="w-5 h-5" />
              ) : (
                <ArrowsPointingOutIcon className="w-5 h-5" />
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Player
