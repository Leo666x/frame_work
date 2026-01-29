package powerai

import (
	"encoding/json"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/env"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/etcd"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/minio"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/pgsql"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/redis"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/weaviate"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xenv"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strconv"
	"time"
)

func initMinio(etcd *etcd_mw.Etcd) (*minio_mw.Minio, error) {
	ev, err := etcd.GetByPrefix(GetServiceInstancePrefixKey("power-ai-minio"))
	if err != nil {
		return nil, fmt.Errorf("通过etcd获取minio服务信息,err: %v", err)
	}
	if len(ev) == 0 {
		return nil, fmt.Errorf("通过etcd获取minio服务信息成功，但是value为空")
	}
	var ip, port, ak, sk string
	for _, v := range ev {
		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			continue
		}
		ip = etcdValue["ip"]
		port = etcdValue["port"]
		ak = etcdValue["username"]
		sk = etcdValue["password"]
		// 优先调用本地的
		if ip == xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1") {
			break
		}
	}

	// 不为空，则替换掉环境变量中数据
	if ip != "" && port != "" && ak != "" && sk != "" {
		env.G.MinioConfig.Ip = ip
		env.G.MinioConfig.Port = port
		env.G.MinioConfig.Ak = ak
		env.G.MinioConfig.Sk = sk
	}
	return minio_mw.New(&minio_mw.Config{
		Ip:   env.G.MinioConfig.Ip,
		Port: env.G.MinioConfig.Port,
		Ak:   env.G.MinioConfig.Ak,
		Sk:   env.G.MinioConfig.Sk,
	})
}

func initPgSql(etcd *etcd_mw.Etcd) (*pgsql_mw.PgSql, error) {
	ev, err := etcd.GetByPrefix(GetServiceInstancePrefixKey("power-ai-postgres"))
	if err != nil {
		return nil, fmt.Errorf("通过etcd获取pgsql服务信息,err: %v", err)
	}
	if len(ev) == 0 {
		return nil, fmt.Errorf("通过etcd获取pgsql服务信息成功，但是value为空")
	}
	var ip, port, password string
	for _, v := range ev {
		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			xlog.LogErrorF("10000", "postgres", "init", fmt.Sprintf("etcd-postgres obj, key : %s,反序列化", v.Key), err)
			continue
		}
		ip = etcdValue["ip"]
		port = etcdValue["port"]
		password = etcdValue["password"]
		if ip == xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1") {
			break
		}
	}
	// 不为空，则替换掉环境变量中数据
	if ip != "" && port != "" && password != "" {
		env.G.PgsqlConfig.Host = ip
		env.G.PgsqlConfig.Port = port
		env.G.PgsqlConfig.Password = password
	}

	return pgsql_mw.New(&pgsql_mw.Config{
		Host:     env.G.PgsqlConfig.Host,
		Port:     env.G.PgsqlConfig.Port,
		Password: env.G.PgsqlConfig.Password,
		Database: env.G.PgsqlConfig.Database,
		Username: env.G.PgsqlConfig.Username,
	})
}

func initRedis(etcd *etcd_mw.Etcd) (*redis_mw.Redis, error) {
	ev, err := etcd.GetByPrefix(GetServiceInstancePrefixKey("power-ai-redis"))
	if err != nil {
		return nil, fmt.Errorf("通过etcd获取redis服务信息,err: %v", err)
	}
	if len(ev) == 0 {
		return nil, fmt.Errorf("通过etcd获取redis服务信息成功，但是value为空")
	}
	var ip, port, password string
	for _, v := range ev {
		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			xlog.LogErrorF("10000", "redis", "init", fmt.Sprintf("etcd-redis obj, key : %s,反序列化", v.Key), err)
			continue
		}
		ip = etcdValue["ip"]
		port = etcdValue["port"]
		password = etcdValue["password"]
		if ip == xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1") {
			break
		}
	}
	if ip != "" && port != "" && password != "" {
		env.G.RedisConfig.Addr = fmt.Sprintf("%s:%s", ip, port)
		env.G.RedisConfig.Password = password
	}
	return redis_mw.New(&redis_mw.Config{
		Addr:     env.G.RedisConfig.Addr,
		Password: env.G.RedisConfig.Password,
	})
}

func initWeaviate(etcd *etcd_mw.Etcd) (*weaviate_mw.Weaviate, error) {
	ev, err := etcd.GetByPrefix(GetServiceInstancePrefixKey("power-ai-weaviate"))
	if err != nil {
		return nil, fmt.Errorf("通过etcd获取weaviate服务信息,err: %v", err)
	}
	if len(ev) == 0 {
		return nil, fmt.Errorf("通过etcd获取weaviate服务信息成功，但是value为空")
	}
	var ip, httpPort, password string
	for _, v := range ev {
		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			xlog.LogErrorF("10000", "weaviate", "init", fmt.Sprintf("etcd-weaviate obj, key : %s,反序列化", v.Key), err)
			continue
		}
		ip = etcdValue["ip"]
		httpPort = etcdValue["http_port"]
		password = etcdValue["password"]
		// 优先调用本地的
		if ip == xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1") {
			// 如果一个都没命中，那么就取最后一个用的
			break
		}
	}

	// 不为空，则替换掉环境变量中数据
	if ip != "" && httpPort != "" && password != "" {
		env.G.WeaviateConfig.Host = fmt.Sprintf("%s:%s", ip, httpPort)
		env.G.WeaviateConfig.ApiKey = password
	}
	return weaviate_mw.New(&weaviate_mw.Config{
		Host:   env.G.WeaviateConfig.Host,
		Scheme: env.G.WeaviateConfig.Scheme,
		ApiKey: env.G.WeaviateConfig.ApiKey,
	})
}

func initMilvus(etcd *etcd_mw.Etcd) (*milvus_mw.Milvus, error) {
	ev, err := etcd.GetByPrefix(GetServiceInstancePrefixKey("power-ai-milvus"))
	if err != nil {
		return nil, fmt.Errorf("通过etcd获取milvus服务信息,err: %v", err)
	}
	if len(ev) == 0 {
		return nil, fmt.Errorf("通过etcd获取milvus服务信息成功，但是value为空")
	}
	var ip, port, password, username, timeout string
	for _, v := range ev {
		etcdValue := make(map[string]string)
		if err = json.Unmarshal([]byte(v.Value), &etcdValue); err != nil {
			xlog.LogErrorF("10000", "milvus", "init", fmt.Sprintf("etcd-milvus obj, key : %s,反序列化", v.Key), err)
			continue
		}
		ip = etcdValue["ip"]
		port = etcdValue["port"]
		password = etcdValue["password"]
		username = etcdValue["username"]
		timeout = etcdValue["timeout"]
		// 优先调用本地的
		if ip == xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1") {
			// 如果一个都没命中，那么就取最后一个用的
			break
		}
	}

	if timeout == "" {
		timeout = "10"
	}
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		timeoutInt = 10
	}

	// 不为空，则替换掉环境变量中数据
	if ip != "" && port != "" {
		env.G.MilvusConfig.Addr = fmt.Sprintf("%s:%s", ip, port)
		env.G.MilvusConfig.Password = password
		env.G.MilvusConfig.Username = username
		env.G.MilvusConfig.Timeout = time.Duration(timeoutInt) * time.Second
	}
	return milvus_mw.New(&milvus_mw.Config{
		Addr:     env.G.MilvusConfig.Addr,
		Username: env.G.MilvusConfig.Username,
		Password: env.G.MilvusConfig.Password,
		Timeout:  env.G.MilvusConfig.Timeout,
	})
}

func initEtcd() (*etcd_mw.Etcd, error) {
	return etcd_mw.New(&etcd_mw.Config{
		Endpoints: env.G.EtcdConfig.Endpoints,
	})
}
func initManifest(manifest string) (*Manifest, error) {
	if manifest == "" {
		return nil, fmt.Errorf("manifest is empty")
	}
	mf := &Manifest{}
	err := json.Unmarshal([]byte(manifest), mf)
	if err != nil {
		return nil, fmt.Errorf("init manifest err:%s", err.Error())
	}

	if mf.Code == "" {
		return nil, fmt.Errorf("manifest code is empty")
	}
	if mf.Name == "" {
		return nil, fmt.Errorf("manifest name is empty")
	}
	if mf.Version == "" {
		return nil, fmt.Errorf("manifest version is empty")
	}
	if mf.Description == "" {
		return nil, fmt.Errorf("manifest description is empty")
	}
	return mf, nil
}
