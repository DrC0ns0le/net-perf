package system

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoKey = fmt.Errorf("key not found")
)

type StateTable struct {
	kv   map[stateKey]*stateValue
	lock sync.RWMutex
}

type stateValue struct {
	value     string
	timestamp time.Time
}

type stateKey struct {
	key       string
	namespace string
}

func NewStateTable() *StateTable {
	return &StateTable{
		kv:   make(map[stateKey]*stateValue),
		lock: sync.RWMutex{},
	}
}

// Get returns the value associated with the key in the package that
// called this function. The second return value is the timestamp when
// the value was set. If the key is not found, the second return value is
// the zero Time.
func (s *StateTable) Get(key string) (string, time.Time, error) {
	packageName := getCaller()

	stateKey := stateKey{
		key:       key,
		namespace: packageName,
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if val, ok := s.kv[stateKey]; ok {
		return val.value, val.timestamp, nil
	}
	return "", time.Time{}, ErrNoKey
}

func (s *StateTable) GetFromNamespace(key, namespace string) (string, time.Time, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if val, ok := s.kv[stateKey{key: key, namespace: namespace}]; ok {
		return val.value, val.timestamp, nil
	}
	return "", time.Time{}, ErrNoKey
}

func (s *StateTable) GetAll(namespace string) map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	m := make(map[string]string)
	for k, v := range s.kv {
		if k.namespace == namespace {
			m[k.key] = v.value
		}
	}
	return m
}

func (s *StateTable) Set(key, value string) {
	packageName := getCaller()

	stateKey := stateKey{
		key:       key,
		namespace: packageName,
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.kv[stateKey] = &stateValue{
		value:     value,
		timestamp: time.Now(),
	}
}

func (s *StateTable) Modify(key, value string) error {
	packageName := getCaller()

	stateKey := stateKey{
		key:       key,
		namespace: packageName,
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if val, ok := s.kv[stateKey]; ok {
		val.value = value
		val.timestamp = time.Now()
	} else {
		return ErrNoKey
	}

	return nil
}

func (s *StateTable) Delete(key string) error {
	packageName := getCaller()

	stateKey := stateKey{
		key:       key,
		namespace: packageName,
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.kv[stateKey]; ok {
		delete(s.kv, stateKey)
	} else {
		return ErrNoKey
	}

	return nil
}

// getCaller returns the calling function's package, file, and line number
func getCaller() string {
	pc, _, _, ok := runtime.Caller(3) // Skip four frames to get the actual caller
	if !ok {
		return "unknown"
	}

	// Get the function name
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	// Extract package name from full function name
	// Format is "package/path.function"
	fullName := fn.Name()
	if lastDot := strings.LastIndexByte(fullName, '.'); lastDot >= 0 {
		if lastSlash := strings.LastIndexByte(fullName[:lastDot], '/'); lastSlash >= 0 {
			return strings.Split(fullName[lastSlash+1:lastDot], ".")[0]
		}
		return strings.Split(fullName[:lastDot], ".")[0]
	}
	return strings.Split(fullName, ".")[0]
}
