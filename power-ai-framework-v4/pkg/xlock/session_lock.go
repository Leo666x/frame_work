package xlock

import (
	"sync"
)

// ============================================================================
// 会话级并发锁管理
// ============================================================================

// SessionLockManager 会话锁管理器
// 用于防止同一会话的并发写入冲突，确保数据一致性
type SessionLockManager struct {
	locks sync.Map // map[conversationID]*sync.Mutex
}

// NewSessionLockManager 创建会话锁管理器
func NewSessionLockManager() *SessionLockManager {
	return &SessionLockManager{
		locks: sync.Map{},
	}
}

// GetLock 获取指定会话的锁
// 参数:
//   - conversationID: 会话ID
// 返回:
//   - *sync.Mutex: 互斥锁指针
//
// 使用场景:
//   在需要修改会话状态的地方使用
//
// 示例:
//   lock := lockManager.GetLock("conv_123")
//   lock.Lock()
//   defer lock.Unlock()
//   // 修改会话状态...
func (m *SessionLockManager) GetLock(conversationID string) *sync.Mutex {
	lock, _ := m.locks.LoadOrStore(conversationID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// LockWith 在锁保护下执行函数
// 参数:
//   - conversationID: 会话ID
//   - fn: 要执行的函数
//
// 使用场景:
//   简化锁的使用，避免忘记释放锁
//
// 示例:
//   err := lockManager.LockWith("conv_123", func() {
//       // 修改会话状态...
//   })
func (m *SessionLockManager) LockWith(conversationID string, fn func()) error) error {
	lock := m.GetLock(conversationID)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

// LockWithVal 在锁保护下执行函数并返回值
// 参数:
//   - conversationID: 会话ID
//   - fn: 要执行的函数
// 返回:
//   - T: 函数返回值
//
// 使用场景:
//   简化锁的使用，避免忘记释放锁，并支持返回值
//
// 示例:
//   result, err := lockManager.LockWithVal("conv_123", func() (string, error) {
//       return "result", nil
//   })
func (m *SessionLockManager) LockWithVal[T any](conversationID string, fn func() (T, error)) (T, error) {
	lock := m.GetLock(conversationID)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}
