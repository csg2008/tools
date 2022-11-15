package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var ip = flag.String("i", "192.168.1.1", "路由器IP地址")
var uname = flag.String("u", "user", "登录用户名")
var passwd = flag.String("p", "", "密码")
var showHelp = flag.Bool("h", false, "显示应用帮助信息并退出")

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "使用说明:", strings.TrimRight(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])), " 参数选项")
		fmt.Fprintln(os.Stderr, "欢迎使用华为路由器自动重启工具")
		fmt.Fprintln(os.Stderr, "更多信息：https://github.com/csg2008/tools/tree/master/router")
		fmt.Fprintln(os.Stderr, "")

		fmt.Fprintln(os.Stderr, "参数选项:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	if "" == *ip || "" == *uname || "" == *passwd {
		fmt.Println("路由器IP，登录用户名，密码不能为空")
		return
	}

	var err error
	var token, session string

	if token, err = getToken(*ip); nil == err {
		if session, err = doLogin(*ip, *uname, *passwd, token); nil == err {
			if restart(*ip, session) {
				fmt.Println("restart ok")
			}
		}
	}

	fmt.Println("test:", err, token, session)
}

func getToken(ip string) (string, error) {
	var idx int
	var url = "http://" + ip + "/asp/GetRandCount.asp"
	var header = [][2]string{
		{"User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:91.0) Gecko/20100101 Firefox/91.0"},
		{"Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
		{"X-Requested-With", "XMLHttpRequest"},
	}

	var _, data, err = httpDo(http.MethodPost, url, "", header)
	if nil == err && len(data) > 3 {
		for k, v := range data {
			if v < 128 {
				idx = k

				break
			}
		}

		return string(data[idx:]), err
	}

	return "", err
}

func doLogin(ip string, user string, passwd string, token string) (string, error) {
	var uri = "http://" + ip + "/login.cgi"
	var param = "UserName=" + user + "&PassWord=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(passwd))) + "&x.X_HW_Token=" + token

	var header = [][2]string{
		{"Host", ip},
		{"User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:91.0) Gecko/20100101 Firefox/91.0"},
		{"Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"},
		{"Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
		{"Content-Type", "application/x-www-form-urlencoded"},
		{"Cookie", "Cookie=body:Language:chinese:id=-1"},
	}

	var session string
	var resp, _, err = httpDo(http.MethodPost, uri, param, header)
	if nil == err && nil != resp {
		if 200 == resp.StatusCode {
			session = resp.Header.Get("Set-Cookie")
		} else {
			err = errors.New("login not ok")
		}
	}

	return session, err
}

func restart(ip string, cookie string) bool {
	var url = "http://" + ip + "/html/ssmp/common/refreshTime.asp"
	var header = [][2]string{
		{"User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:91.0) Gecko/20100101 Firefox/91.0"},
		{"Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
		{"Accept-Encoding", "gzip, deflate"},
		{"Cookie", cookie},
	}

	var resp, _, err = httpDo(http.MethodGet, url, "", header)

	return nil == err && nil != resp && 200 == resp.StatusCode
}

func httpDo(method string, url string, data string, header [][2]string) (*http.Response, []byte, error) {
	var err error
	var ret, dump []byte
	var req *http.Request
	var resp *http.Response
	var client = &http.Client{}

	if req, err = http.NewRequest(method, url, strings.NewReader(data)); err != nil {
		return nil, nil, err
	}

	for _, v := range header {
		req.Header.Set(v[0], v[1])
	}

	if dump, err = httputil.DumpRequest(req, true); nil == err {
		fmt.Println("request:", string(dump))
	}

	if resp, err = client.Do(req); nil != err {
		return nil, nil, err
	}

	if dump, err = httputil.DumpResponse(resp, true); nil == err {
		fmt.Println("response:", string(dump))
	}

	defer resp.Body.Close()
	ret, err = ioutil.ReadAll(resp.Body)

	return resp, ret, err
}
