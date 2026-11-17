package filex

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// POSIX replacement keeps existing shared readers on the previous version.
// MoveFileEx (used by os.Rename) cannot provide that behavior on Windows.
func renameAtomic(source, destination string) error {
	sourceName, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	name, err := windows.UTF16FromString(destination)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(sourceName, windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	type renameInfo struct {
		Flags          uint32
		RootDirectory  windows.Handle
		FileNameLength uint32
		FileName       [1]uint16
	}
	var layout renameInfo
	size := int(unsafe.Offsetof(layout.FileName)) + len(name)*2
	if size < int(unsafe.Sizeof(layout)) {
		size = int(unsafe.Sizeof(layout))
	}
	buffer := make([]byte, size)
	info := (*renameInfo)(unsafe.Pointer(&buffer[0]))
	info.Flags = windows.FILE_RENAME_REPLACE_IF_EXISTS | windows.FILE_RENAME_POSIX_SEMANTICS
	info.FileNameLength = uint32((len(name) - 1) * 2)
	copy(unsafe.Slice(&info.FileName[0], len(name)), name)
	err = windows.SetFileInformationByHandle(handle, windows.FileRenameInfoEx, &buffer[0], uint32(len(buffer)))
	// Older Windows/filesystems may not implement this information class.
	// Preserve the legacy atomic operation, never delete the destination first.
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) || errors.Is(err, windows.ERROR_NOT_SUPPORTED) || errors.Is(err, windows.ERROR_INVALID_FUNCTION) {
		return os.Rename(source, destination)
	}
	return err
}

// openShared permits atomic replacement while this reader retains the old file.
func openShared(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(handle), path), nil
}
