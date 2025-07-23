package services

import (
	"context"
	"sync"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
)

// TaskDecisionManager manages real-time decision channels for blocking webhook handlers
type TaskDecisionManager struct {
	decisions map[string]chan domain.ActionType
	mutex     sync.RWMutex
}

// NewTaskDecisionManager creates a new decision manager
func NewTaskDecisionManager() *TaskDecisionManager {
	return &TaskDecisionManager{
		decisions: make(map[string]chan domain.ActionType),
	}
}

// CreateDecisionChannel creates a new decision channel for a task
func (m *TaskDecisionManager) CreateDecisionChannel(taskID string) chan domain.ActionType {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	decisionChan := make(chan domain.ActionType, 1)
	m.decisions[taskID] = decisionChan
	return decisionChan
}

// SendDecision sends a decision to the waiting channel
func (m *TaskDecisionManager) SendDecision(taskID string, decision domain.ActionType) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if decisionChan, exists := m.decisions[taskID]; exists {
		select {
		case decisionChan <- decision:
			return true
		default:
			// Channel full or closed
			return false
		}
	}
	return false
}

// RemoveDecisionChannel removes and closes a decision channel
func (m *TaskDecisionManager) RemoveDecisionChannel(taskID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if decisionChan, exists := m.decisions[taskID]; exists {
		close(decisionChan)
		delete(m.decisions, taskID)
	}
}

// WaitForDecision waits for a user decision with timeout
func (m *TaskDecisionManager) WaitForDecision(ctx context.Context, taskID string, timeout time.Duration) (domain.ActionType, error) {
	decisionChan := m.CreateDecisionChannel(taskID)
	defer m.RemoveDecisionChannel(taskID)

	select {
	case decision := <-decisionChan:
		return decision, nil
	case <-time.After(timeout):
		return "", ErrDecisionTimeout
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// GetActiveDecisions returns the number of active decision channels
func (m *TaskDecisionManager) GetActiveDecisions() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.decisions)
}

// HasPendingDecision checks if a task has a pending decision
func (m *TaskDecisionManager) HasPendingDecision(taskID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.decisions[taskID]
	return exists
}

// CleanupExpiredChannels removes channels that haven't been used (emergency cleanup)
// This should rarely be needed as channels are cleaned up in defer statements
func (m *TaskDecisionManager) CleanupExpiredChannels() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Close and remove all channels
	for taskID, decisionChan := range m.decisions {
		close(decisionChan)
		delete(m.decisions, taskID)
	}
}