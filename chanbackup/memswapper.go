package chanbackup

import (
	"os"
	"sync"

	"github.com/lightningnetwork/lnd/keychain"
)

// MemorySwapper stores the latest packed multi-channel backup in memory.
// It is useful for environments where direct filesystem access is unavailable.
type MemorySwapper struct {
	mu     sync.Mutex
	backup PackedMulti
}

// NewMemorySwapper creates an in-memory channel backup swapper.
func NewMemorySwapper() *MemorySwapper {
	return &MemorySwapper{}
}

// UpdateAndSwap stores the latest packed multi-channel backup.
func (m *MemorySwapper) UpdateAndSwap(newBackup PackedMulti) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.backup = append(m.backup[:0], newBackup...)
	return nil
}

// ExtractMulti returns the currently stored multi-channel backup.
func (m *MemorySwapper) ExtractMulti(keyChain keychain.KeyRing) (*Multi, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.backup) == 0 {
		return nil, os.ErrNotExist
	}

	return m.backup.Unpack(keyChain)
}
