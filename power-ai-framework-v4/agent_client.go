package powerai

import (
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/etcd"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xcache"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strings"
	"sync/atomic"
	"time"
)

// ***************************************************************************************************************
//
//	AgentClient 用于调用智能体
//  智能体etcd注册说明
//  key: /service/instance/{智能体编号}
//  value : {
//	 			"ip":"",
//				"port":"",
//              "version":"",
//              "code":"",
//              "name":""
//			}
// ***************************************************************************************************************

type AgentClient struct {
	etcd      *etcd_mw.Etcd                   //即用方式,无需循环etcd状态
	instances *xcache.Cache[string, []string] // key:agent_code,value:ip:port
	counters  *xcache.Cache[string, uint64]
}

func newAgentClient(etcd *etcd_mw.Etcd) *AgentClient {
	a := &AgentClient{
		etcd:      etcd,
		instances: xcache.NewCache[string, []string](),
		counters:  xcache.NewCache[string, uint64](),
	}
	a.loadAll()
	go a.watch()
	return a
}

func (i *AgentClient) loadAll() {

	// 判断etcd是否初始化
	if i.etcd == nil {
		return
	}

	// 调用etcd，通过前缀获取所有的注册实例
	ev, err := i.etcd.GetByPrefix(AgentInstancePrefixKey)
	// 判断etcd是否调用成功
	if err != nil {
		return
	}

	// 判断返回结果是否为空
	if len(ev) == 0 {
		return
	}

	// 对结果进行循环，如果发现是本地服务，那么优先使用，逻辑：优先调用本地智能体，如果一个都没命中，那么就取最后一个用的
	for _, v := range ev {

		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			continue
		}
		agentCode := etcdValue["code"]
		addrs, _ := i.instances.Get(agentCode)
		addrs = append(addrs, fmt.Sprintf("%s:%s", etcdValue["ip"], etcdValue["port"]))
		i.instances.Set(agentCode, addrs)
	}

	return
}

func (i *AgentClient) get(agentCode string) (string, error) {

	addr := i.next(agentCode)
	if addr != "" {
		return addr, nil
	}

	// 判断etcd是否初始化
	if i.etcd == nil {
		return "", fmt.Errorf("etcd 未初始化")
	}

	// 获取etcd和本地缓存存储的key
	key := GetServiceInstancePrefixKey(agentCode)
	// 调用etcd，通过前缀获取
	ev, err := i.etcd.GetByPrefix(key)
	// 判断etcd是否调用成功
	if err != nil {
		return "", err
	}

	// 判断返回结果是否为空
	if len(ev) == 0 {
		return "", fmt.Errorf("etcd 返回结果为空")
	}

	var etcdAddrs []string
	// 对结果进行循环，如果发现是本地服务，那么优先使用，逻辑：优先调用本地智能体，如果一个都没命中，那么就取最后一个用的
	for _, v := range ev {
		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			continue
		}
		etcdAddrs = append(etcdAddrs, fmt.Sprintf("%s:%s", etcdValue["ip"], etcdValue["port"]))
	}

	i.instances.Set(agentCode, etcdAddrs)

	addr = i.next(agentCode)
	if addr == "" {
		return "", fmt.Errorf("未发现可用地址")
	}

	return addr, nil
}

// next 选择一个合理的服务地址,轮询方案
func (i *AgentClient) next(agentCode string) string {
	// 获取etcd和本地缓存存储的key
	//key := getEtcdServiceInstancePrefixKey(agentCode)
	// 优先从缓存中获取地址列表
	addrs, ok := i.instances.Get(agentCode)
	if !ok || len(addrs) <= 0 {
		return ""
	}
	// 存在地址,轮询访问
	idx := i.getAndIncrementCounter(agentCode) % uint64(len(addrs))
	return addrs[idx]
}

// getAndIncrementCounter 获取并递增服务的计数器（线程安全）
func (i *AgentClient) getAndIncrementCounter(agentCode string) uint64 {
	// 先尝试直接获取并递增
	if count, ok := i.counters.Get(agentCode); ok {
		// 使用原子操作确保线程安全
		newCount := atomic.AddUint64(&count, 1)
		// 更新缓存中的值
		i.counters.Set(agentCode, newCount)
		return count // 返回递增前的值作为索引
	}

	// 首次访问，初始化为1（因为已经使用了一次）
	i.counters.Set(agentCode, 1)
	return 0 // 首次调用返回0
}

func (i *AgentClient) register(ip, port, agentCode, agentName, agentVersion string) {
	m := map[string]string{
		"ip":      ip,
		"port":    port,
		"version": agentVersion,
		"code":    agentCode,
		"name":    agentName,
	}
	b, _ := json.Marshal(&m)
	for {
		key := GetServiceInstanceFullKey(agentCode, ip, port)
		//  put key 并且 创建续租
		leaseID, err := i.etcd.GrantAndSet(60, key, string(b))
		if err != nil {
			xlog.LogErrorF("10000", "agent-register", "register", fmt.Sprintf("[%s]注册服务信息并且创建续约失败，5秒之后继续", key), err)
			time.Sleep(5 * time.Second)
			continue
		}
		//  启动自动续约
		keepAliveChan, err := i.etcd.KeepAlive(leaseID)
		if err != nil {
			xlog.LogErrorF("10000", "agent-register", "register", fmt.Sprintf("[%s]启动KeepAlive失败，5秒后重试", key), err)
			time.Sleep(5 * time.Second)
			continue
		}

		xlog.LogInfoF("10000", "agent-register", "register", fmt.Sprintf("[%s]注册成功并开始自动续约", key))

		//  监听续约响应
		for resp := range keepAliveChan {
			if resp == nil {
				xlog.LogErrorF("10000", "agent-register", "register", fmt.Sprintf("[%s]KeepAlive通道关闭，准备重新注册", key), nil)
				break // 退出循环，重新注册
			}
		}
		//  如果到这里，说明租约失效或网络异常，重新注册
		time.Sleep(2 * time.Second)
	}
}

func (i *AgentClient) update(value []byte) {
	etcdValue := make(map[string]string)
	if err := json.Unmarshal(value, &etcdValue); err != nil {
		return
	}
	v, ok := i.instances.Get(etcdValue["code"])
	if !ok {
		// 不存在表示 此服务没有访问过这个服务，不进行存储
		return
	}
	// update地址
	updateAddr := fmt.Sprintf("%s:%s", etcdValue["ip"], etcdValue["port"])

	for _, addr := range v {
		if addr == updateAddr {
			//这个地址存在则直接返回
			return
		}
	}
	v = append(v, updateAddr)
	i.instances.Set(etcdValue["code"], v)
}

func (i *AgentClient) delete(key string) {
	sfx := strings.ReplaceAll(key, AgentInstancePrefixKey, "")
	codeAndAddr := strings.Split(sfx, "/")
	if len(codeAndAddr) < 2 {
		return
	}
	delCode := codeAndAddr[0]
	delAddr := codeAndAddr[1]

	addrs, _ := i.instances.Get(delCode)
	var newAddrs []string
	for _, addr := range addrs {
		if addr != delAddr {
			newAddrs = append(newAddrs, addr)
		}
	}
	i.instances.Set(delCode, newAddrs)
}

func (i *AgentClient) watch() {
	rch, err := i.etcd.WatchPrefixKey(AgentInstancePrefixKey)
	if err != nil {
		xlog.LogErrorF("10000", "agent-client", "watch", fmt.Sprintf("监听[%s]失败", AgentInstancePrefixKey), err)
		return
	}
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if ev.Type == clientv3.EventTypePut {
				i.update(ev.Kv.Value)
			} else if ev.Type == clientv3.EventTypeDelete {
				i.delete(string(ev.Kv.Key))
			}
		}
	}
}
