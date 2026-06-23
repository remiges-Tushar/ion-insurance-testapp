package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// CallbackManager manages the async→sync bridging via Redis pub/sub.
//
// Key format:     Sync#{route}#{txnId}      (35s TTL)
// Channel format: Callback#{route}#{txnId}
//
// We key only on txnId (not msgId) because onix-bpp generates a fresh
// message_id for on_* responses, so the response msgId never matches the
// request msgId that was registered.
type CallbackManager struct {
	rdb *redis.Client
}

// NewCallbackManager creates a CallbackManager connected to the given Redis address.
func NewCallbackManager(redisAddr string) *CallbackManager {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &CallbackManager{rdb: rdb}
}

func (cm *CallbackManager) pendingKey(route, txnId string) string {
	return fmt.Sprintf("Sync#%s#%s", route, txnId)
}

func (cm *CallbackManager) callbackChannel(route, txnId string) string {
	return fmt.Sprintf("Callback#%s#%s", route, txnId)
}

// Register marks a pending request in Redis with a 35-second TTL.
// Must be called before forwarding the request to onix-bap so that the
// subscriber is ready when the on_* webhook arrives.
func (cm *CallbackManager) Register(ctx context.Context, route, txnId, msgId string) error {
	key := cm.pendingKey(route, txnId)
	log.Printf("[CallbackManager] Register key=%s", key)

	metadata := map[string]string{
		"transaction_id": txnId,
		"message_id":     msgId,
		"created_at":     time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	return cm.rdb.Set(ctx, key, data, 35*time.Second).Err()
}

// Wait subscribes to the callback channel and blocks until a message arrives
// or the timeout elapses.  It returns the raw body bytes published by Publish.
func (cm *CallbackManager) Wait(ctx context.Context, route, txnId, msgId string, timeout time.Duration) ([]byte, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	channel := cm.callbackChannel(route, txnId)
	log.Printf("[CallbackManager] Wait channel=%s timeout=%v", channel, timeout)

	pubsub := cm.rdb.Subscribe(waitCtx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	select {
	case msg := <-ch:
		log.Printf("[CallbackManager] Received on channel=%s", channel)
		return []byte(msg.Payload), nil
	case <-waitCtx.Done():
		return nil, fmt.Errorf("timeout waiting for callback on channel %s", channel)
	}
}

// Publish delivers body to the waiting frontend handler.  It checks that the
// pending key exists (so we don't publish to an unregistered route), then
// publishes on the channel and deletes the key.
func (cm *CallbackManager) Publish(ctx context.Context, route, txnId, msgId string, body []byte) error {
	key := cm.pendingKey(route, txnId)
	log.Printf("[CallbackManager] Publish key=%s", key)

	exists, err := cm.rdb.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("checking pending key: %w", err)
	}
	if exists == 0 {
		keys, _ := cm.rdb.Keys(ctx, "Sync#*").Result()
		log.Printf("[CallbackManager] WARN: key not found (%s); existing Sync keys: %v", key, keys)
		return fmt.Errorf("no pending request found for key %s", key)
	}

	channel := cm.callbackChannel(route, txnId)
	n, err := cm.rdb.Publish(ctx, channel, string(body)).Result()
	if err != nil {
		return fmt.Errorf("publish to channel %s: %w", channel, err)
	}
	log.Printf("[CallbackManager] Published to %d subscriber(s) on channel=%s", n, channel)

	if err := cm.rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("[CallbackManager] WARN: failed to delete key %s: %v", key, err)
	}
	return nil
}
