package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AbsPath 转换相对路径为绝对路径
func AbsPath(target string) string {
	var basePath, _ = filepath.Abs(target)
	return strings.Replace(basePath, "\\", "/", -1)
}

// GetAppPath 返回应用程序当前路径
func GetAppPath() string {
	var curPath, _ = exec.LookPath(os.Args[0])

	return strings.Replace(filepath.Dir(AbsPath(curPath)), "\\", "/", -1)
}

// IsFile returns true if given path is a file,
// or returns false when it's a directory or does not exist.
func IsFile(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// CopyFile 复制文件
func CopyFile(dst string, src string) (written int64, err error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}

	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// FileGetContents Get bytes to file.
// if non-exist, create this file.
func FileGetContents(filename string) (data []byte, e error) {
	var f *os.File

	f, e = os.OpenFile(filename, os.O_RDONLY, os.ModePerm)

	if e != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()

	stat, e := f.Stat()
	if e != nil {
		return
	}
	data = make([]byte, stat.Size())
	result, e := f.Read(data)
	if e != nil || int64(result) != stat.Size() {
		return nil, e
	}

	return
}

// FilePutContents Put bytes to file.
// if non-exist, create this file.
func FilePutContents(filename string, content []byte, append bool) error {
	var flag int
	if append {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	fp, err := os.OpenFile(filename, flag, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()

	_, err = fp.Write(content)
	return err
}

func main() {
	var err error
	var msg, xmlFile string
	var logData []byte
	var appPath = GetAppPath() + "/x2t.bak.exe"
	var logFile = GetAppPath() + "/x2t.trace.log"
	var cmd = exec.Command(appPath, os.Args[1:]...)

	rand.Seed(time.Now().UnixNano())

	if IsFile(logFile) {
		logData, _ = FileGetContents(logFile)
	}
	if nil == logData {
		logData = make([]byte, 0, 100)
	}

	msg = strings.Join(os.Args, " ") + "\n"
	if strings.HasSuffix(os.Args[1], ".xml") {
		xmlFile = GetAppPath() + "/" + strconv.FormatInt(time.Now().UnixNano(), 10) + "." + strconv.FormatInt(rand.Int63n(10000), 10) + ".xml"
		CopyFile(xmlFile, os.Args[1])
		msg = msg + "copy to: " + xmlFile + "\n"
	}

	msg = msg + "\n"
	logData = append(logData, []byte(msg)...)
	err = FilePutContents(logFile, logData, true)

	if err = cmd.Run(); nil == err {
		os.Exit(0)
	}

	os.Exit(0)
}
