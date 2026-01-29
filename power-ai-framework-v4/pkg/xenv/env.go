package xenv

import (
	"net"
	"os"
	"strconv"
)

// GetEnvOrDefault 获取环境变量的值，如果不存在则返回默认值
func GetEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
func GetEnvOrDefaultInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	valueInt, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return valueInt
}

func GetEnvOrDefaultBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	valueBool, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return valueBool
}

// GetEnv 获取环境变量的值，如果不存在则返回默认值
func GetEnv(key string) string {
	value := os.Getenv(key)
	return value
}

// GetInternalIp return internal ipv4.
func GetInternalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err.Error())
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip != nil && (ip.IsLinkLocalUnicast() || ip.IsGlobalUnicast()) {
			continue
		}

		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}

	return ""
}
