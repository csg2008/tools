package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/csg2008/tools/util"
)

var configPath = flag.String("c", "hrt.json", "配置文件路径")
var savePath = flag.String("o", "hrt.csv", "数据输出路径")
var timeout = flag.Uint("t", 5, "请求超时时间（秒）")
var interval = flag.Uint("i", 60, "请求间隔时间（秒）")
var showHelp = flag.Bool("h", false, "显示应用帮助信息并退出")

// request 进行 WEB 请求
func request(config map[string]string, idx int, timeout uint, wg *sync.WaitGroup, mux *sync.Mutex, times *map[int]int64) {
	var err error
	var code int64
	var req *http.Request
	var resp *http.Response
	var start = time.Now().UnixNano()
	var client = http.Client{Timeout: time.Duration(timeout) * time.Second}

	if "" == config["param"] {
		req, err = http.NewRequest(http.MethodGet, config["url"], strings.NewReader(""))
	} else {
		req, err = http.NewRequest(http.MethodPost, config["url"], strings.NewReader(config["param"]))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}

	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.87 Safari/537.36")

	//dump, _ := httputil.DumpRequest(req, true)

	resp, err = client.Do(req)
	if nil == err {
		defer resp.Body.Close()

		code = int64(resp.StatusCode)

		_, err = ioutil.ReadAll(resp.Body)

		// if strings.Index(config["url"], "aliyuncs") > 0 {
		// 	fmt.Println(string(dump))
		// 	fmt.Println(string(ret))
		// }
	}

	mux.Lock()
	(*times)[idx*2+1] = code
	(*times)[idx*2] = time.Now().UnixNano() - start
	mux.Unlock()

	wg.Done()
}

// batchRequest 批量请求
func batchRequest(configs []map[string]string, out *os.File, timeout uint) {
	var wg sync.WaitGroup
	var mux = new(sync.Mutex)
	var use = make(map[int]int64)
	var log = time.Now().Format("2006-01-02 15:04:05")

	for k, v := range configs {
		wg.Add(1)

		go request(v, k, timeout, &wg, mux, &use)
	}

	wg.Wait()

	for k := range configs {
		log = log + "," + strconv.FormatFloat(float64(use[k*2])/1000000000.0, 'f', 3, 64) + "," + strconv.FormatInt(use[k*2+1], 10)
	}

	log = log + "\n"

	out.WriteString(log)
}

// monitor 监视请求响应
func monitor(configs []map[string]string, out *os.File, timeout uint, interval uint) {
	var timer = time.NewTicker(time.Duration(interval) * time.Second)

	out.WriteString("请求时间,")
	for k, v := range configs {
		out.WriteString(v["label"] + "(时间)")
		out.WriteString(",")
		out.WriteString(v["label"] + "(状态)")

		if k != len(configs)-1 {
			out.WriteString(",")
		}
	}

	out.WriteString("\n")

	for {
		<-timer.C

		go batchRequest(configs, out, timeout)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "使用说明:", strings.TrimRight(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])), " 参数选项")
		fmt.Fprintln(os.Stderr, "欢迎使用 HTTP 请求时间监视工具")
		fmt.Fprintln(os.Stderr, "更多信息：https://github.com/csg2008/tools/tree/master/hrt")
		fmt.Fprintln(os.Stderr, "")

		fmt.Fprintln(os.Stderr, "参数选项:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	if 0 == *timeout || 0 == *interval {
		fmt.Println("请求超时时间与请求间隔时间必须大于零")
	}
	if !util.IsFile(*configPath) {
		fmt.Println("配置文件", *configPath, "不存在")
		return
	}

	var err error
	var outFile *os.File
	var configs = make([]map[string]string, 0, 100)

	if err = util.LoadJSON(*configPath, &configs); nil != err {
		fmt.Println("加载配置文件", *configPath, "失败", err)
		return
	}

	if "" == *savePath {
		outFile = os.Stdout
	} else {
		var flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		if outFile, err = os.OpenFile(*savePath, flag, os.ModePerm); err != nil {
			return
		}

		defer outFile.Close()
	}

	monitor(configs, outFile, *timeout, *interval)
}
