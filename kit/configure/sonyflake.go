package configure

import (
	"errors"
	"sync"

	"github.com/sony/sonyflake/v2"
)

var ErrSonyflakeMachineID = errors.New("需要配置 sonyflake_machine_id，范围为 0~65535，且在同一 ID 空间内每个进程唯一")

var sonyflakeInstances = struct {
	sync.Mutex
	instances map[int]*sonyflake.Sonyflake
}{instances: make(map[int]*sonyflake.Sonyflake)}

// Sonyflake 获取按进程编号共享的 ID 生成器
func (c Configure) Sonyflake() (instance *sonyflake.Sonyflake, err error) {
	if c.SonyflakeMachineID == nil || *c.SonyflakeMachineID < 0 || *c.SonyflakeMachineID > 65535 {
		err = ErrSonyflakeMachineID
		return
	}

	machineID := *c.SonyflakeMachineID

	sonyflakeInstances.Lock()
	defer sonyflakeInstances.Unlock()

	instance = sonyflakeInstances.instances[machineID]
	if instance != nil {
		return
	}

	instance, err = sonyflake.New(sonyflake.Settings{
		MachineID: func() (int, error) { return machineID, nil },
	})
	if err == nil {
		sonyflakeInstances.instances[machineID] = instance
	}

	return
}
