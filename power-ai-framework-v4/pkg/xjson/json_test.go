package xjson

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	json := `{
	"head":{
		"charset":"utf-8",
		"encrypt_type":"AES",
		"enterprise_id":"orgine",
		"language":"zh_CN",
		"method":"orgine.powerempi.service.api.externalservice.listEmpiUser",
		"sign":"",
		"sign_type":"md5",
		"sys_track_code":"8e3470d5b13a4a5a9413d125f7f2ca89",
		"timestamp":"1605431894505",
		"version":"1.0",
		"access_token":"1231231231231",
		"app_id":"f75f4ab345bb4034a6d377e135a9baf5"
	},
	"body":{
		"info":{
			"system_code":null,
			"identifier_type":"01",
			"mobile":null,
			"name":null,
			"system_user_id":null,
			"identifier_no":"360124199410200312",
			"mpi_id":null
		}
	}
}`

	r := Get(json, "body.info.identifier_no")
	fmt.Println(r.String())
}
func TestSet(t *testing.T) {
	json := `{
	"head":{
		"charset":"utf-8",
		"encrypt_type":"AES",
		"enterprise_id":"orgine",
		"language":"zh_CN",
		"method":"orgine.powerempi.service.api.externalservice.listEmpiUser",
		"sign":"",
		"sign_type":"md5",
		"sys_track_code":"8e3470d5b13a4a5a9413d125f7f2ca89",
		"timestamp":"1605431894505",
		"version":"1.0",
		"access_token":"1231231231231",
		"app_id":"f75f4ab345bb4034a6d377e135a9baf5"
	},
	"body":{
		"info":{
			"system_code":null,
			"identifier_type":"01",
			"mobile":null,
			"name":null,
			"system_user_id":null,
			"identifier_no":"360124199410200312",
			"mpi_id":null
		}
	}
}`

	r, err := Set(json, "body.info.abc", "这是我新增的一个字段")
	if err != nil {
		panic(err)
	}
	fmt.Println(r)
}
