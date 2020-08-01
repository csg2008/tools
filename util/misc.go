package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	mrand "math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// seq 序列号
var seq int64

// lts 最近一次的时间
var lts int64

// DatePatterns 格式标识映射
var DatePatterns = []string{
	// time zone
	"CST", "CST",
	"GMT", "GMT",
	"T", "MST",
	"P", "-07:00",
	"O", "-0700",

	// year
	"Y", "2006", // A full numeric representation of a year, 4 digits   Examples: 1999 or 2003
	"y", "06", //A two digit representation of a year   Examples: 99 or 03

	// month
	"m", "01", // Numeric representation of a month, with leading zeros 01 through 12
	"n", "1", // Numeric representation of a month, without leading zeros   1 through 12
	"M", "Jan", // A short textual representation of a month, three letters Jan through Dec
	"F", "January", // A full textual representation of a month, such as January or March   January through December

	// day
	"d", "02", // Day of the month, 2 digits with leading zeros 01 to 31
	"j", "2", // Day of the month without leading zeros 1 to 31

	// week
	"D", "Mon", // A textual representation of a day, three letters Mon through Sun
	"l", "Monday", // A full textual representation of the day of the week  Sunday through Saturday

	// time
	"g", "3", // 12-hour format of an hour without leading zeros    1 through 12
	"G", "15", // 24-hour format of an hour without leading zeros   0 through 23
	"h", "03", // 12-hour format of an hour with leading zeros  01 through 12
	"H", "15", // 24-hour format of an hour with leading zeros  00 through 23

	"a", "pm", // Lowercase Ante meridiem and Post meridiem am or pm
	"A", "PM", // Uppercase Ante meridiem and Post meridiem AM or PM

	"i", "04", // Minutes with leading zeros    00 to 59
	"s", "05", // Seconds, with leading zeros   00 through 59

	// RFC 2822
	"r", time.RFC1123Z,
}

// DatetimeFormat 日期时间格式转换
// 已经将时区调为 UTC 0，如需要时区可以在格式化串中传入
// var t, err = DatetimeFormat("2017/11/13 10:11:02 GMT+0800", "Y/m/d H:i:s GMT+0800", "timestamp")
// fmt.Println("out:", t, err)
// out: 1510539062 <nil>
// var t1, e1 = DatetimeFormat(1510539062, "timestamp", "Y/m/d H:i:s GMT+0800")
// fmt.Println("out:", t1, e1)
// out: 2017/11/13 10:11:02 GMT+0800 <nil>
// var t2, e2 = DatetimeFormat(1510539062, "timestamp", "Y/m/d H:i:s")
// out: 2017/11/13 10:11:02 <nil>
func DatetimeFormat(in interface{}, inFormat string, outFormat string) (interface{}, error) {
	// 把日期转换为时间
	replacer := strings.NewReplacer(DatePatterns...)
	if "timestamp" == outFormat {
		inFormat = replacer.Replace(inFormat)
		tm, err := time.ParseInLocation(inFormat, in.(string), time.Local)
		if nil != err {
			return nil, err
		}

		return tm.UTC().Unix(), nil
	} else {
		var t time.Time
		outFormat = replacer.Replace(outFormat)

		if v, ok := in.(string); ok {
			t = time.Unix(StrToInt(v), 0)
		} else if v, ok := in.(int); ok {
			t = time.Unix(int64(v), 0)
		} else if v, ok := in.(uint); ok {
			t = time.Unix(int64(v), 0)
		} else if v, ok := in.(int64); ok {
			t = time.Unix(v, 0)
		} else if v, ok := in.(uint64); ok {
			t = time.Unix(int64(v), 0)
		} else if v, ok := in.(int32); ok {
			t = time.Unix(int64(v), 0)
		} else if v, ok := in.(uint32); ok {
			t = time.Unix(int64(v), 0)
		} else {
			return nil, errors.New("unknown timestamp value")
		}

		return t.Format(outFormat), nil
	}
}

// TimestampToDate 将时间戳转换为日期
func TimestampToDate(in int64) int {
	var t = time.Unix(in, 0)
	var y, m, d = t.Date()

	return y*10000 + int(m)*100 + d
}

// FindInStringSlice 从 slice 中查找指定的字符串并返回相应的索引值，找不到返回-1
func FindInStringSlice(str string, s []string) int {
	for i, e := range s {
		if e == str {
			return i
		}
	}
	return -1
}

// NewUUID 生成UUID字符串
func NewUUID() string {
	u := [16]byte{}
	rand.Read(u[:])
	u[8] = (u[8] | 0x40) & 0x7F
	u[6] = (u[6] & 0xF) | (4 << 4)
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}

// RandomMAC 生成MAC地址
func RandomMAC() string {
	mrand.Seed(time.Now().UnixNano())
	mac := [6]byte{0x80, 0x3f,
		byte(mrand.Intn(0x7F)), byte(mrand.Intn(0x7F)),
		byte(mrand.Intn(0x7F)), byte(mrand.Intn(0x7F))}

	return hex.EncodeToString(mac[:])
}

// SubString 字符串截取
func SubString(source string, start int, end int) string {
	var r = []rune(source)
	length := len(r)
	if start < 0 || end > length || start > end {
		return ""
	}
	if start == 0 && end == length {
		return source
	}
	return string(r[start:end])
}

// GetSeqNo 返回一个可用的序列号
func GetSeqNo() int64 {
	var v = atomic.AddInt64(&seq, 1)
	if v >= 9999 {
		atomic.StoreInt64(&seq, 1)
		atomic.StoreInt64(&lts, time.Now().Unix()*10000)
	} else if 1 == v {
		atomic.StoreInt64(&lts, time.Now().Unix()*10000)
	}

	return atomic.LoadInt64(&lts) + atomic.LoadInt64(&seq)
}

// LoadJSON 从文件加载 JSON 到变量
func LoadJSON(file string, in interface{}) error {
	var err error
	var data []byte

	if data, err = FileGetContents(file); nil == err {
		err = json.Unmarshal(data, in)
	}

	return err
}

// SaveJSON 保存数据到 JSON 文件
func SaveJSON(file string, in interface{}) error {
	var bj, _ = json.Marshal(in)

	return FilePutContents(file, bj, false)
}

// StrToInt 字符串转数字
func StrToInt(in string) int64 {
	in = strings.TrimSpace(in)
	ret, err := strconv.ParseInt(in, 10, 64)

	if nil != err {
		ret = 0
	}

	return ret
}

// SafeFileName replace all illegal chars to a underline char
func SafeFileName(fileName string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(`/\:*?"><|`, r) != -1 {
			return '_'
		}
		return r
	}, fileName)
}

// GetDirFiles 获取指定文件夹文件列表
func GetDirFiles(dirPath string, stripExt bool) []string {
	var ret []string

	if files, err := ioutil.ReadDir(dirPath); nil == err && nil != files {
		ret = make([]string, 0, len(files))
		for _, v := range files {
			if !v.IsDir() {
				if stripExt {
					var tmp = v.Name()
					var idx = strings.LastIndexByte(v.Name(), '.')
					if idx > 0 {
						ret = append(ret, strings.Trim(tmp[0:idx], " "))
					} else {
						ret = append(ret, v.Name())
					}
				} else {
					ret = append(ret, v.Name())
				}
			}
		}
	}

	return ret
}

// WalkRelFiles 遍历文件，可指定后缀，返回相对路径
func WalkRelFiles(target string, suffixes ...string) (files []string) {
	if !filepath.IsAbs(target) {
		target, _ = filepath.Abs(target)
	}
	err := filepath.Walk(target, func(retpath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if len(suffixes) == 0 {
			files = append(files, RelPath(retpath, target))
			return nil
		}
		_retpath := RelPath(retpath, target)
		for _, suffix := range suffixes {
			if strings.HasSuffix(_retpath, suffix) {
				files = append(files, _retpath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("function.WalkRelFiles: %v\n", err)
		return
	}

	return
}

// WalkRelDirs 遍历目录，可指定后缀，返回相对路径
func WalkRelDirs(target string, suffixes ...string) (dirs []string) {
	if !filepath.IsAbs(target) {
		target, _ = filepath.Abs(target)
	}
	err := filepath.Walk(target, func(retpath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() {
			return nil
		}
		if len(suffixes) == 0 {
			dirs = append(dirs, RelPath(retpath, target))
			return nil
		}
		_retpath := RelPath(retpath, target)
		for _, suffix := range suffixes {
			if strings.HasSuffix(_retpath, suffix) {
				dirs = append(dirs, _retpath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("utils.WalkRelDirs: %v\n", err)
		return
	}

	return
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

// IsDir returns true if given path is a directory,
func IsDir(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return f.IsDir()
}

// IsExist checks whether a file or directory exists.
// It returns false when the file or directory does not exist.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

/**
 * 判断指定文件是否存在
 */
func IsFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

/**
 * 判断指定目录是否存在
 */
func IsDirExist(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	} else {
		return fi.IsDir()
	}
	return true
}

// Dirname 返回指定路径的文件夹名
func Dirname(target string) string {
	idx := strings.LastIndex(strings.TrimRight(target, "/"), "/")

	if -1 != idx {
		return target[:idx]
	}

	return ""
}

// MkDir 创建文件夹
func MkDir(path string) error {
	if !IsDir(path) {
		return os.Mkdir(path, os.ModePerm)
	}

	return nil
}

// RelPath 转相对路径
func RelPath(target string, basePath string) string {
	//basePath, _ := filepath.Abs("./")
	rel, _ := filepath.Rel(basePath, target)
	return strings.Replace(rel, "\\", "/", -1)
}

// AbsPath 转换相对路径为绝对路径
func AbsPath(target string) string {
	basePath, _ := filepath.Abs(target)
	return strings.Replace(basePath, "\\", "/", -1)
}

// FileExt 返回文件扩展名
func FileExt(path string) string {
	return filepath.Ext(path)
}

// GetAppPath 返回应用程序当前路径
func GetAppPath() string {
	var curPath, _ = exec.LookPath(os.Args[0])

	return strings.Replace(filepath.Dir(AbsPath(curPath)), "\\", "/", -1)
}

// FileGetContents Get bytes to file.
// if non-exist, create this file.
func FileGetContents(filename string) (data []byte, e error) {
	f, e := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
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

// GetFileList Gets a list of files under the folder
func GetFileList(src string) ([]string, error) {
	var fs []string
	dir, err := ioutil.ReadDir(src)
	if err != nil {
		return nil, err
	}
	for _, v := range dir {
		fs = append(fs, v.Name())
	}
	return fs, nil
}

// GetFileListCount Gets the number of files in the folder
func GetFileListCount(src string) (int, error) {
	dir, err := ioutil.ReadDir(src)
	if err != nil {
		return 0, err
	}
	var res int
	for range dir {
		res++
	}
	return res, nil
}

// StringSliceDifference concatenates slices together based on its index and
// returns an individual string array
func StringSliceDifference(slice1 []string, slice2 []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff = append(diff, s1)
			}
		}
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}
	return diff
}

// ByteFill 字节数组填充
func ByteFill(src []byte, n int) []byte {
	var ret []byte

	if len(src) <= n {
		ret = make([]byte, n)
		for k, v := range src {
			ret[k] = v
		}
	} else {
		ret = src
	}

	return ret
}

// ByteToUInt16 字节数组转16位整数
func ByteToUInt16(src []byte) uint16 {
	return binary.LittleEndian.Uint16(ByteFill(src, 2))
}

// ByteToUInt32 字节数组转32位整数
func ByteToUInt32(src []byte) uint32 {
	return binary.LittleEndian.Uint32(ByteFill(src, 4))
}

// ByteToUInt64 字节数组转64位整数
func ByteToUInt64(src []byte) uint64 {
	return binary.LittleEndian.Uint64(ByteFill(src, 8))
}

// ByteToFloat32 字节转 flat32
func ByteToFloat32(bytes []byte) float32 {
	return math.Float32frombits(ByteToUInt32(bytes))
}

// ByteToFloat64 字节转 flat64
func ByteToFloat64(bytes []byte) float64 {
	return math.Float64frombits(ByteToUInt64(bytes))
}

// RoundFloat rounds your floating point number to the desired decimal place
func RoundFloat(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	intermed += .5
	x = .5
	if frac < 0.0 {
		x = -.5
		intermed--
	}
	if frac >= x {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
}

// ToString 把数据转换为字符串
func ToString(in interface{}) string {
	var ret string

	if v, ok := in.(string); ok {
		ret = v
	} else if v, ok := in.(int); ok {
		ret = strconv.FormatInt(int64(v), 10)
	} else if v, ok := in.(uint); ok {
		ret = strconv.FormatUint(uint64(v), 10)
	} else if v, ok := in.(uint64); ok {
		ret = strconv.FormatUint(v, 10)
	} else if v, ok := in.(int64); ok {
		ret = strconv.FormatInt(v, 10)
	} else if v, ok := in.(float64); ok {
		ret = strconv.FormatFloat(v, 'f', 0, 64)
	} else if v, ok := in.(float32); ok {
		ret = strconv.FormatFloat(float64(v), 'f', 0, 32)
	}

	return ret
}

// ToInt 把数据转换为数值
func ToInt(in interface{}) int64 {
	var t int64
	if v, ok := in.(string); ok {
		t, _ = strconv.ParseInt(v, 10, 64)
	} else if v, ok := in.(int); ok {
		t = int64(v)
	} else if v, ok := in.(uint); ok {
		t = int64(v)
	} else if v, ok := in.(int64); ok {
		t = v
	} else if v, ok := in.(uint64); ok {
		t = int64(v)
	} else if v, ok := in.(int32); ok {
		t = int64(v)
	} else if v, ok := in.(uint32); ok {
		t = int64(v)
	} else if v, ok := in.(float32); ok {
		t = int64(v)
	} else if v, ok := in.(float64); ok {
		t = int64(v)
	} else {
		t = 0
	}

	return t
}

// ToFloat 把数据转换为浮点数
func ToFloat(in interface{}) float64 {
	var t float64
	if v, ok := in.(string); ok {
		t, _ = strconv.ParseFloat(v, 64)
	} else if v, ok := in.(int); ok {
		t = float64(v)
	} else if v, ok := in.(uint); ok {
		t = float64(v)
	} else if v, ok := in.(int64); ok {
		t = float64(v)
	} else if v, ok := in.(uint64); ok {
		t = float64(v)
	} else if v, ok := in.(int32); ok {
		t = float64(v)
	} else if v, ok := in.(uint32); ok {
		t = float64(v)
	} else if v, ok := in.(float32); ok {
		t = float64(v)
	} else if v, ok := in.(float64); ok {
		t = v
	} else {
		t = 0
	}

	return t
}

// GetArgVal 读取参数值
func GetArgVal(name string, inputs []string) []string {
	var flag = false
	var args = make([]string, 0, len(inputs))

	for _, arg := range inputs {
		if name == arg {
			flag = true

			continue
		}
		if flag && strings.HasPrefix(arg, "-") {
			break
		}
		if flag {
			args = append(args, arg)
		}
	}

	return args
}

// GetMapVal 读取 map 中指定路径数据
func GetMapVal(in map[string]interface{}, keys ...string) (interface{}, bool) {
	if len(keys) > 0 {
		var idx int
		var cnt = len(keys)
		var val map[string]interface{}

		val = in
		for _, key := range keys {
			if tmp, ok := val[key]; ok {
				idx++

				if idx == cnt {
					return tmp, true
				}

				if v, ok := tmp.(map[string]interface{}); ok {
					val = v
				}
			}
		}
	}

	return nil, false
}

// MultiMapPromote 提升多维 map 为一维
func MultiMapPromote(in *map[string]interface{}) map[string]interface{} {
	var tmp map[string]interface{}
	var ret = make(map[string]interface{})

	for k, v := range *in {
		if vv, ok := v.(map[string]interface{}); ok {
			tmp = MultiMapPromote(&vv)
			for k1, v1 := range tmp {
				ret[k1] = v1
			}
		} else {
			ret[k] = v
		}
	}

	return ret
}

// MD5 返回字节序列的 md5 值
func MD5(in []byte) string {
	h := md5.New()
	h.Write(in)

	return hex.EncodeToString(h.Sum(nil))
}
