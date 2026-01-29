package etcd_mw

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type Etcd struct {
	client *clientv3.Client
	config *Config
}
type Config struct {
	Endpoints []string
}

type EtcdValue struct {
	Key   string
	Value string
}

func New(c *Config) (*Etcd, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Etcd{client: client, config: c}, nil
}

func (e *Etcd) Close() {
	if e.client != nil {
		_ = e.client.Close()
		e.client = nil
	}
}

func (e *Etcd) check() error {
	if e.client == nil {
		return fmt.Errorf("etcd初始化失败,endpoints：%s", e.config.Endpoints)
	}
	return nil
}

// Get 根据指定KEY查询
func (e *Etcd) Get(key string) (*EtcdValue, error) {
	v := &EtcdValue{Key: key}
	if err := e.check(); err != nil {
		return v, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return v, fmt.Errorf("key '%s' invoke err: %v", key, err)
	}

	if len(resp.Kvs) == 0 {
		return v, fmt.Errorf("key '%s' value is empty ", key) // 键不存在
	}

	value := resp.Kvs[0].Value
	if value == nil {
		return v, fmt.Errorf("key '%s' value is empty ", key) // 此情况不会发生，防御性编程， 冗余判断
	}

	v.Value = string(value)
	return v, nil
}

// GetByPrefix 按前缀查询key
func (e *Etcd) GetByPrefix(prefix string) ([]*EtcdValue, error) {
	var v []*EtcdValue

	if err := e.check(); err != nil {
		return v, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return v, fmt.Errorf("prefix '%s' invoke err: %v", prefix, err)
	}

	if len(resp.Kvs) == 0 {
		return v, fmt.Errorf("prefix '%s' value is empty ", prefix) // 键不存在
	}
	value := resp.Kvs[0].Value
	if value == nil {
		return v, fmt.Errorf("prefix '%s' value is empty ", prefix) // 此情况不会发生，防御性编程， 冗余判断
	}
	for _, kv := range resp.Kvs {
		v = append(v, &EtcdValue{
			Key:   string(kv.Key),
			Value: string(kv.Value),
		})
	}
	return v, nil
}

// Set 将值存入etcd
func (e *Etcd) Set(key, value string) error {
	if err := e.check(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := e.client.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("set key '%s' err: %v", key, err)
	}
	return nil
}

func (e *Etcd) Watch(key string) (clientv3.WatchChan, error) {
	if err := e.check(); err != nil {
		return nil, err
	}
	ch := e.client.Watch(context.Background(), key)
	return ch, nil
}

func (e *Etcd) WatchPrefixKey(prefix string) (clientv3.WatchChan, error) {
	if err := e.check(); err != nil {
		return nil, err
	}
	ch := e.client.Watch(context.Background(), prefix, clientv3.WithPrefix())
	return ch, nil
}

func (e *Etcd) GrantAndSet(ttl int64, key, value string) (clientv3.LeaseID, error) {
	if err := e.check(); err != nil {
		return 0, err
	}
	leaseResp, err := e.client.Grant(context.Background(), ttl)
	if err != nil {
		return 0, err
	}
	_, err = e.client.Put(context.Background(), key, value, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return 0, nil
	}
	return leaseResp.ID, nil
}

func (e *Etcd) KeepAliveOnce(leaseId clientv3.LeaseID) error {
	if err := e.check(); err != nil {
		return err
	}
	_, err := e.client.KeepAliveOnce(context.Background(), leaseId)
	if err != nil {
		return err
	}
	return nil
}
func (e *Etcd) KeepAlive(leaseId clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	if err := e.check(); err != nil {
		return nil, err
	}
	resp, err := e.client.KeepAlive(context.Background(), leaseId)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
