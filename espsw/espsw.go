package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

var endpoint = flag.String("e", "", "TCP端点")
var command = flag.String("c", "", "命令代码")
var showVer = flag.Bool("v", false, "显示应用版本信息并退出")
var showHelp = flag.Bool("h", false, "显示应用帮助信息并退出")

// commands 继电器开关命令
var commands = map[string][]byte{
	"po": []byte{0xA0, 0x01, 0x01, 0xA2},
	"pc": []byte{0xA0, 0x01, 0x00, 0xA1},
	"ro": []byte{0xA0, 0x02, 0x01, 0xA3},
	"rc": []byte{0xA0, 0x02, 0x00, 0xA2},
}

// sendCMD 发送命令
func sendCMD(addr string, cmd1 []byte, cmd2 []byte, delay int64) error {
	var err error
	var conn net.Conn
	var buf = make([]byte, 1024)
	fmt.Println("addr:", addr)
	if conn, err = net.Dial("tcp", addr); nil != err {
		return err
	}

	defer conn.Close()
	if _, err = conn.Write(cmd1); nil != err {
		return err
	}

	conn.SetReadDeadline((time.Now().Add(time.Second * time.Duration(delay))))
	conn.Read(buf)

	if nil != cmd2 {
		if _, err = conn.Write(cmd2); nil != err {
			return err
		}
	}

	conn.SetReadDeadline((time.Now().Add(time.Second * time.Duration(delay))))
	conn.Read(buf)

	return err
}

// triggerDefault 重置继电器为默认常闭状态
func triggerDefault(addr string) error {
	return sendCMD(addr, commands["pc"], commands["rc"], 1)
}

// triggerPowerOn 触发电源开关
func triggerPowerOn(addr string) error {
	return sendCMD(addr, commands["po"], commands["pc"], 1)
}

// triggerReset 触发重启开关
func triggerReset(addr string) error {
	return sendCMD(addr, commands["ro"], commands["rc"], 1)
}

// triggerPowerForce 触发强制关闭电源
func triggerPowerForce(addr string) error {
	return sendCMD(addr, commands["po"], commands["pc"], 6)
}

func main() {
	var err error
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "使用说明:", filepath.Base(os.Args[0]), "参数选项")
		fmt.Fprintln(os.Stderr, "欢迎使用基于 ESP 双路继电器开关工具")
		fmt.Fprintln(os.Stderr, "更多信息：http://github.com/csg800/tools/tree/master/espsw")
		fmt.Fprintln(os.Stderr, "")

		fmt.Fprintln(os.Stderr, "参数选项:")
		flag.PrintDefaults()

		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "命令代码：")
		fmt.Fprintln(os.Stderr, "          reset     重启")
		fmt.Fprintln(os.Stderr, "          power     开机")
		fmt.Fprintln(os.Stderr, "          forceOff  强制关机")
	}

	flag.Parse()

	if *showHelp || "" == *endpoint || "" == *command {
		flag.Usage()
		return
	}

	switch *command {
	case "power":
		err = triggerPowerOn(*endpoint)
		triggerDefault(*endpoint)
	case "reset":
		err = triggerReset(*endpoint)
		triggerDefault(*endpoint)
	case "forceOff":
		err = triggerPowerForce(*endpoint)
		triggerDefault(*endpoint)
	default:
		err = errors.New("不支持的命令")
	}

	if nil != err {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
