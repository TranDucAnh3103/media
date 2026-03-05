import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import App from './App.jsx'
import './index.css'

// ========================================
// PROTECTION: Chặn copy, context menu, phím tắt
// ========================================

// Chặn chuột phải (context menu)
document.addEventListener('contextmenu', (e) => {
  e.preventDefault()
  return false
})

// Chặn các phím tắt copy/save
document.addEventListener('keydown', (e) => {
  // Ctrl+C, Ctrl+V, Ctrl+X, Ctrl+S, Ctrl+U, Ctrl+Shift+I, F12
  if (
    (e.ctrlKey && ['c', 'v', 'x', 's', 'u', 'p'].includes(e.key.toLowerCase())) ||
    (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'i') 
  ) {
    // Cho phép trong input/textarea
    const tagName = e.target.tagName.toLowerCase()
    if (tagName === 'input' || tagName === 'textarea') {
      return true
    }
    e.preventDefault()
    return false
  }
})

// Chặn kéo thả
document.addEventListener('dragstart', (e) => {
  e.preventDefault()
  return false
})

// React Query client cho data fetching & caching
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 phút
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
        <Toaster 
          position="top-right"
          toastOptions={{
            style: {
              background: '#1f2937',
              color: '#fff',
            },
          }}
        />
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>,
)
