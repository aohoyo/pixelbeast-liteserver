//go:build windows

package panel

import (
	"os"
	"syscall"
	"unsafe"
)

// isAdmin 检测当前进程是否以管理员权限运行
func isAdmin() bool {
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	procAllocateAndInitializeSid := advapi32.NewProc("AllocateAndInitializeSid")
	procCheckTokenMembership := advapi32.NewProc("CheckTokenMembership")
	procFreeSid := advapi32.NewProc("FreeSid")

	// SID_IDENTIFIER_AUTHORITY: 6 bytes, NT AUTHORITY = {0,0,0,0,0,5}
	type sidAuth struct{ Value [6]byte }
	var ntAuth sidAuth
	ntAuth.Value[5] = 5

	var sid uintptr
	ret, _, _ := procAllocateAndInitializeSid.Call(
		uintptr(unsafe.Pointer(&ntAuth)),
		2,
		32,  // SECURITY_BUILTIN_DOMAIN_RID
		544, // DOMAIN_ALIAS_RID_ADMINS
		0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&sid)),
	)
	if ret == 0 {
		return false
	}
	defer procFreeSid.Call(sid)

	var isMember int32
	ret, _, _ = procCheckTokenMembership.Call(
		0,
		sid,
		uintptr(unsafe.Pointer(&isMember)),
	)
	return ret != 0 && isMember != 0
}

// canWriteDir 检测是否可写入指定目录
func canWriteDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".cleanup_test_*")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}
