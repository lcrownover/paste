package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

func GetPaste(rdb *redis.Client, pasteID string) (*Paste, bool, error) {
	result, err := rdb.Get(pasteID).Result()
	if err == redis.Nil {
		slog.Info("Paste not found in Redis", "id", pasteID)
		return nil, false, nil
	} else if err != nil {
		slog.Error("Failed to retrieve paste from Redis", "error", err, "id", pasteID)
		return nil, false, fmt.Errorf("failed to get paste: %v", err)
	}

	var rPaste redisPaste
	if err := json.Unmarshal([]byte(result), &rPaste); err != nil {
		slog.Error("Failed to unmarshal paste", "error", err, "id", pasteID, "raw", result)
		return nil, false, fmt.Errorf("failed to unmarshal paste: %v", err)
	}
	ttl, err := rdb.TTL(pasteID).Result()
	if err != nil {
		slog.Error("Failed to get TTL for paste", "error", err, "id", pasteID, "raw", ttl)
		return nil, false, fmt.Errorf("failed to get TTL for key: %v", err)
	}
	paste := Paste{
		ID:              rPaste.ID,
		Content:         rPaste.Content,
		LifetimeSeconds: int64(ttl.Seconds()),
	}

	return &paste, true, nil
}

func CreatePaste(rdb *redis.Client, content string, lifetimeSeconds int64) (*Paste, error) {
	pasteID := uuid.New().String()

	p := redisPaste{
		ID:      pasteID,
		Content: content,
	}
	pj, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal paste in creation: %v", err)
	}

	err = rdb.Set(pasteID, pj, time.Duration(lifetimeSeconds)*time.Second).Err()
	if err != nil {
		slog.Error("Failed to save paste to Redis", "error", err)
		return nil, fmt.Errorf("failed to create paste in redis: %v", err)
	}

	createdPaste, found, err := GetPaste(rdb, pasteID)
	if err != nil || !found {
		slog.Error("Failed to verify paste was saved", "id", pasteID, "error", err)
	} else {
		slog.Info("Paste saved and verified in Redis", "id", pasteID)
	}

	return createdPaste, nil
}

func DeletePaste(rdb *redis.Client, pasteID string) error {
	exists, err := rdb.Exists(pasteID).Result()
	if err != nil {
		slog.Error("Error checking if paste exists", "id", pasteID, "error", err)
		return fmt.Errorf("failed to find existing pastes: %v", err)
	}

	if exists == 0 {
		return nil
	}

	_, err = rdb.Del(pasteID).Result()
	if err != nil {
		slog.Error("Error deleting paste", "id", pasteID, "error", err)
		return fmt.Errorf("failed to delete existing paste: %v", err)
	}

	return nil
}
