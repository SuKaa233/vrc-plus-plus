//go:build !windows

package updater

import (
	"errors"
	"syscall"
)

func hiddenProcessAttributes() *syscall.SysProcAttr { return nil }
func verifyInstaller(string) error {
	return errors.New("当前系统不支持 Windows 安装程序更新")
}
