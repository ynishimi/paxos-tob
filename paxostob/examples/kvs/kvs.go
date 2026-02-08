package kvs

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/ynishimi/paxos-tob/paxostob"
)

// keyValue stores key and value. it can be sent as a payload of Message
type keyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (kv *keyValue) Serialize() ([]byte, error) {
	return json.Marshal(kv)
}

func Deserialize(data []byte) (kv *keyValue, err error) {
	err = json.Unmarshal(data, &kv)
	return kv, err
}

func (kv *keyValue) String() string {
	return fmt.Sprintln(kv.Key, kv.Value)
}

type kvMessage struct {
	src    string
	byteKv []byte
}

func (m *kvMessage) Src() string {
	return m.src
}
func (m *kvMessage) Payload() []byte {
	return m.byteKv
}

func (m *kvMessage) String() string {
	keyValue, err := Deserialize(m.byteKv)
	if err != nil {
		log.Error().Err(err).Msg("failed to deserialize")
		return fmt.Sprintf("[src:%s] ?", m.src)
	}
	return fmt.Sprintf("[src:%s] %s", m.src, keyValue.String())
}

func (kvs *simpleKvs) NewKVMessage(k, v string) *kvMessage {
	kv := keyValue{
		Key:   k,
		Value: v,
	}
	byteKv, err := kv.Serialize()
	if err != nil {
		log.Error().Err(err).Msg("failed to serialize")
	}
	return &kvMessage{
		src:    kvs.GetAddress(),
		byteKv: byteKv,
	}
}

type kvs interface {
	GetAddress() string
	Put(k, v string) <-chan error
	Get(k string) (v string, err error)
}

type simpleKvs struct {
	tob     paxostob.TotalOrderBroadcast
	storage map[string]string
	// keeps pending kv (for notifying completion of Put())
	pending map[keyValue]chan error
	mu      sync.RWMutex
}

func NewSimpleKvs(tob paxostob.TotalOrderBroadcast) *simpleKvs {
	kvs := &simpleKvs{
		tob:     tob,
		storage: make(map[string]string),
		pending: make(map[keyValue]chan error),
	}

	// recv delivered msgs

	go func() {
		for deliveredMsg := range tob.Deliver() {
			log.Debug().Msg("delivered")
			kv, err := Deserialize(deliveredMsg.Payload())
			log.Debug().Msg("deserialized")
			if err != nil {
				log.Error().Err(err).Msg("failed to deserialize")
			}
			log.Debug().Str(kv.Key, kv.Value).Msg("delivered result")
			kvs.store(kv)
			kvs.notifyPending(kv)
		}
	}()

	return kvs

}

func (kvs *simpleKvs) GetAddress() string {
	return kvs.tob.GetAddress()
}

func (kvs *simpleKvs) Put(k, v string) <-chan error {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	msg := kvs.NewKVMessage(k, v)
	log.Debug().Str("kv", msg.String()).Msg("newKVMessage")
	kvs.tob.Broadcast(msg)

	// todo: err handling of broadcast
	done := make(chan error)
	kvs.pending[keyValue{Key: k, Value: v}] = done
	return done
}

func (kvs *simpleKvs) Get(k string) (v string, err error) {
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()

	v, ok := kvs.storage[k]
	if !ok {
		return "", fmt.Errorf("value not available for %s", k)
	}

	return v, nil
}

func (kvs *simpleKvs) store(kv *keyValue) {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	kvs.storage[kv.Key] = kv.Value
}

func (kvs *simpleKvs) notifyPending(kv *keyValue) {
	kvs.mu.Lock()
	ch := kvs.pending[*kv]
	kvs.mu.Unlock()
	close(ch)
}
