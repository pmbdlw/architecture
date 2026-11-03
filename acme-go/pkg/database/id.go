package database

import (
	"hash/fnv"
	"os"
	"sync"

	"github.com/sony/sonyflake/v2"
)

var (
	flake     *sonyflake.Sonyflake
	flakeOnce sync.Once
)

// NextID 生成全局唯一、趋势递增的 int64 ID（Sonyflake）。
// 机器号默认取 Pod 私有 IP 的低 16 位；取不到时用主机名哈希。
// 注意：int64 超过 JS 安全整数，对外 JSON 一律以字符串输出（protojson 默认如此）。
func NextID() int64 {
	flakeOnce.Do(func() {
		var err error
		flake, err = sonyflake.New(sonyflake.Settings{})
		if err != nil {
			flake, err = sonyflake.New(sonyflake.Settings{MachineID: hostnameMachineID})
			if err != nil {
				panic(err)
			}
		}
	})
	id, err := flake.NextID()
	if err != nil {
		panic(err)
	}
	return id
}

func hostnameMachineID() (int, error) {
	name, _ := os.Hostname()
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int(h.Sum32() & 0xFFFF), nil
}
