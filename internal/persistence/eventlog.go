package persistence

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxEventSize = 4 << 20

// Event 是追加日志中的一条可校验记录。Sequence 和摘要字段由 Store 统一填写，
// 调用方只需提供 Type、Payload 和可选的 At。
type Event struct {
	Sequence       int64     `json:"sequence"`
	Type           string    `json:"type"`
	Payload        any       `json:"payload"`
	At             time.Time `json:"at"`
	PreviousDigest string    `json:"previousDigest,omitempty"`
	Digest         string    `json:"digest"`
}

type eventDigestInput struct {
	Sequence       int64     `json:"sequence"`
	Type           string    `json:"type"`
	Payload        any       `json:"payload"`
	At             time.Time `json:"at"`
	PreviousDigest string    `json:"previousDigest,omitempty"`
}

func digestEvent(event Event) (string, error) {
	data, err := json.Marshal(eventDigestInput{
		Sequence:       event.Sequence,
		Type:           event.Type,
		Payload:        event.Payload,
		At:             event.At,
		PreviousDigest: event.PreviousDigest,
	})
	if err != nil {
		return "", fmt.Errorf("编码事件摘要输入: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// AppendEvent 在 Store 的单写入锁内分配连续序号、连接摘要链并同步落盘。
// 如果已有日志不完整或摘要不匹配，函数会拒绝继续追加，避免掩盖损坏。
func (s *Store) AppendEvent(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event.Type = strings.TrimSpace(event.Type)
	if event.Type == "" {
		return errors.New("事件类型不能为空")
	}
	events, err := s.readEventsLocked()
	if err != nil {
		return err
	}
	nextSequence := int64(1)
	previousDigest := ""
	if len(events) > 0 {
		last := events[len(events)-1]
		nextSequence = last.Sequence + 1
		previousDigest = last.Digest
	}
	if event.Sequence != 0 && event.Sequence != nextSequence {
		return fmt.Errorf("事件序号不连续：期望 %d，得到 %d", nextSequence, event.Sequence)
	}
	event.Sequence = nextSequence
	event.PreviousDigest = previousDigest
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	} else {
		event.At = event.At.UTC()
	}
	event.Digest, err = digestEvent(event)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("创建事件日志目录: %w", err)
	}
	path := filepath.Join(s.dir, "events.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开事件日志: %w", err)
	}
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("编码事件: %w", err)
	}
	data = append(data, '\n')
	written, err := file.Write(data)
	if err != nil {
		return fmt.Errorf("追加事件日志: %w", err)
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("同步事件日志: %w", err)
	}
	return nil
}

// Events 返回已经通过序号及摘要链校验的日志副本。
func (s *Store) Events() ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readEventsLocked()
}

func (s *Store) readEventsLocked() ([]Event, error) {
	path := filepath.Join(s.dir, "events.jsonl")
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Event{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("打开事件日志: %w", err)
	}
	defer file.Close()

	events := []Event{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxEventSize)
	line := 0
	for scanner.Scan() {
		line++
		var event Event
		if err = json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("解析事件日志第 %d 行: %w", line, err)
		}
		if err = validateEvent(event, events); err != nil {
			return nil, fmt.Errorf("校验事件日志第 %d 行: %w", line, err)
		}
		events = append(events, event)
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取事件日志: %w", err)
	}
	return events, nil
}

func validateEvent(event Event, previous []Event) error {
	expectedSequence := int64(len(previous) + 1)
	if event.Sequence != expectedSequence {
		return fmt.Errorf("序号不连续：期望 %d，得到 %d", expectedSequence, event.Sequence)
	}
	if strings.TrimSpace(event.Type) == "" {
		return errors.New("事件类型为空")
	}
	if event.At.IsZero() {
		return errors.New("事件时间为空")
	}
	expectedPrevious := ""
	if len(previous) > 0 {
		expectedPrevious = previous[len(previous)-1].Digest
	}
	if event.PreviousDigest != expectedPrevious {
		return errors.New("前序摘要不匹配")
	}
	expectedDigest, err := digestEvent(event)
	if err != nil {
		return err
	}
	if event.Digest != expectedDigest {
		return errors.New("事件摘要不匹配")
	}
	return nil
}
