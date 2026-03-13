import { Fragment } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { XMarkIcon } from '@heroicons/react/24/outline'

const NotificationModal = ({ open, onClose, notifications = [] }) => {
  return (
    <Transition show={open} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-200"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-150"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/40" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Transition.Child
              as={Fragment}
              enter="transform transition ease-out duration-200"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="transform transition ease-in duration-150"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md bg-gray-900/95 backdrop-blur-lg rounded-2xl p-4 border border-white/10">
                <div className="flex items-center justify-between mb-2">
                  <Dialog.Title className="text-lg font-medium text-white">Thông báo</Dialog.Title>
                  <button onClick={onClose} className="p-2 text-gray-400 hover:text-white rounded-md">
                    <XMarkIcon className="w-5 h-5" />
                  </button>
                </div>

                <div className="divide-y divide-white/5 max-h-72 overflow-auto">
                  {notifications.length === 0 ? (
                    <p className="py-6 text-center text-sm text-gray-400">Bạn không có thông báo nào</p>
                  ) : (
                    notifications.map((n, i) => (
                      <div key={i} className="py-3 text-sm text-gray-300 px-1">
                        <p className="font-medium text-white">{n.title}</p>
                        <p className="text-xs text-gray-400">{n.message}</p>
                      </div>
                    ))
                  )}
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  )
}

export default NotificationModal
