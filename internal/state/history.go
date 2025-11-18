package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
	"gorm.io/gorm"
)

// HistoryEntry represents a state change history entry
type HistoryEntry struct {
	ID         uuid.UUID              `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	EntityType string                 `gorm:"type:varchar(50);not null;index:idx_state_history_entity" json:"entity_type"`
	EntityID   uuid.UUID              `gorm:"type:uuid;not null;index:idx_state_history_entity" json:"entity_id"`
	OldState   *string                `gorm:"type:varchar(50)" json:"old_state"`
	NewState   string                 `gorm:"type:varchar(50);not null" json:"new_state"`
	ChangedAt  time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_state_history_changed_at" json:"changed_at"`
	Metadata   map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

// TableName specifies the table name for HistoryEntry
func (HistoryEntry) TableName() string {
	return "state_history"
}

// HistoryTracker tracks state changes to a database
type HistoryTracker struct {
	db *gorm.DB
}

// NewHistoryTracker creates a new history tracker
func NewHistoryTracker(db *gorm.DB) *HistoryTracker {
	return &HistoryTracker{db: db}
}

// Record records a state change to the history table
func (h *HistoryTracker) Record(ctx context.Context, entityType, entityID string, oldState, newState models.State, metadata map[string]interface{}) error {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return fmt.Errorf("invalid entity ID: %w", err)
	}

	var oldStateStr *string
	if oldState != "" {
		str := string(oldState)
		oldStateStr = &str
	}

	entry := HistoryEntry{
		EntityType: entityType,
		EntityID:   entityUUID,
		OldState:   oldStateStr,
		NewState:   string(newState),
		ChangedAt:  time.Now().UTC(),
		Metadata:   metadata,
	}

	if err := h.db.WithContext(ctx).Create(&entry).Error; err != nil {
		return fmt.Errorf("failed to record state history: %w", err)
	}

	return nil
}

// GetHistory retrieves state history for an entity
func (h *HistoryTracker) GetHistory(ctx context.Context, entityType, entityID string, limit int) ([]HistoryEntry, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	var entries []HistoryEntry
	query := h.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityUUID).
		Order("changed_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to get state history: %w", err)
	}

	return entries, nil
}

// GetRecentHistory retrieves recent state changes across all entities
func (h *HistoryTracker) GetRecentHistory(ctx context.Context, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	query := h.db.WithContext(ctx).
		Order("changed_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent history: %w", err)
	}

	return entries, nil
}

// HistoryPublisher publishes state changes to the history tracker
type HistoryPublisher struct {
	tracker *HistoryTracker
}

// NewHistoryPublisher creates a new history publisher
func NewHistoryPublisher(db *gorm.DB) *HistoryPublisher {
	return &HistoryPublisher{
		tracker: NewHistoryTracker(db),
	}
}

// Publish records a state change event to the history
func (p *HistoryPublisher) Publish(event TransitionEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oldState := event.OldState
	if oldState == "" {
		oldState = models.State("") // Empty state for new entities
	}

	return p.tracker.Record(ctx, event.EntityType, event.EntityID, oldState, event.NewState, event.Metadata)
}

// MarshalJSON implements custom JSON marshaling for metadata
func (h *HistoryEntry) MarshalJSON() ([]byte, error) {
	type Alias HistoryEntry
	return json.Marshal(&struct {
		*Alias
		Metadata string `json:"metadata"`
	}{
		Alias:    (*Alias)(h),
		Metadata: fmt.Sprintf("%v", h.Metadata),
	})
}
