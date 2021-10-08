package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/spf13/cast"
	"github.com/zh-five/xdaemon"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Timestr struct {
	Value string
	Type  int
}

type Crontab_str struct {
	sec   Timestr
	min   Timestr
	hour  Timestr
	day   Timestr
	month Timestr
	week  Timestr
	year  Timestr
}

var (
	help       bool
	s          *bool
	f          *string
	d          *bool
	kill       *bool
	dir        string
	conf_finfo fs.FileInfo //配置文件信息
)

func init() {
	dir, _ = os.Getwd()
	flag.BoolVar(&help, "h", false, "帮助信息")
	s = flag.Bool("s", false, "是否调试输出")
	f = flag.String("c", dir+"/config", "指定配置文件")
	d = flag.Bool("d", false, "是否守护进程执行")
	kill = flag.Bool("kill", false, "杀掉守护进程")
}

func main() {
	flag.Parse()
	defer dealPanic() //捕捉异常并处理

	if help {
		readme()
		return
	}
	if _, err := os.Lstat(*f); err != nil {
		panic("配置文件不存在")
		return
	}
	if *kill {
		byte_pid, _ := os.ReadFile(dir + "/pid")
		kcmd := exec.Command("kill", bytes.NewBuffer(byte_pid).String())
		err := kcmd.Run()
		if err != nil {
			panic(err)
		}
		return
	}

	conf_finfo, _ = os.Stat(*f)
	arr := parse_conf(*f)

	//守护进程执行
	if *d {
		_, _ = xdaemon.Background(dir+"/log", true) //启动一个子进程后主程序退出
		pid := os.Getpid()
		_ = ioutil.WriteFile(dir+"/pid", bytes.NewBufferString(cast.ToString(pid)).Bytes(), os.ModePerm)
	}

	clocker(arr)
}

//处理配置文件的内容
func parse_conf(file_path string) []string {
	byte, err := ioutil.ReadFile(file_path)
	//添加文件热更新重载crontab
	if err != nil {
		panic(err)
	}
	cfg := cast.ToString(byte)
	//处理#
	reg := regexp.MustCompile(`#(.*)\s?`)
	st := reg.ReplaceAllString(cfg, "")
	//处理空行
	arr := strings.Split(strings.TrimSpace(st), "\r\n")
	return arr
}

//解析字段内的数据
func parse(cron []string, ntime time.Time) {
	if len(cron) > 0 {

		for _, val := range cron {
			param := strings.Split(val, " ")
			second := param[0] //秒数
			minute := param[1] //分钟
			hour := param[2]   //小时
			day := param[3]    //日
			month := param[4]  //月
			week := param[5]   //周
			year := param[6]   //年
			//判断是否在执行时间内
			if convert(second, minute, hour, day, month, week, year, ntime) {
				command := param[7]                 //执行命令
				args := append(param[8:])           //执行命令的参数
				if runtime.NumGoroutine() > 10000 { //执行协程不能超过10000个
					panic("超出协程数处理")
				} else {
					go execute(command, args)
				}
			}
		}
	}
}

//转换crontab的时间格式
func convert(sec, min, hour, day, month, week, year string, ntime time.Time) bool {
	str_sec := preg(sec)   //检查秒数参数是否在正常数值内
	str_min := preg(min)   //检查分钟数参数是否在正常数值内
	str_hour := preg(hour) //检查小时数参数是否在正常数值内
	str_day := preg(day)   //检查天数
	str_week := preg(week) //检查星期几
	str_mon := preg(month) //检查月数
	str_year := preg(year) //检查年数
	str := Crontab_str{str_sec, str_min, str_hour, str_day, str_mon, str_week, str_year}
	str.sec = str_sec
	//validate,err := tdeal(str, ntime, interval)
	validate, err := str.cronhandle(ntime)

	if err != nil {
		fmt.Println(err) //输出错误给用户睇
		return false
	}

	//验证时间是否满足
	if validate == 1 {
		return true
	}

	return false
}

//处理每个时间字段的结果
func (str *Crontab_str) cronhandle(ntime time.Time) (int, error) {
	var interval int = 0
	var arr = []int{1, 2, 6}
	var sec_func func(string) bool
	var min_func func(string) bool
	var hour_func func(string) bool
	var day_func func(string) bool
	var month_func func(string) bool
	var week_func func(string) bool
	var year_func func(string) bool

	//处理/*
	if inArray(str.sec.Type, arr) {
		sec_func = func(arg string) bool { return ntime.Second()%cast.ToInt(arg) == 0 }
	}

	if inArray(str.min.Type, arr) {
		min_func = func(arg string) bool {
			var sec_operate bool
			switch str.sec.Type {
			case 1:
				sec_operate = ntime.Second()%cast.ToInt(arg) == 0
			default:
				sec_operate = ntime.Second() == cast.ToInt(str.sec.Value)
			}
			s := sec_operate
			return ntime.Minute()%cast.ToInt(arg) == 0 && sec_operate && s == true
		}
	}

	if inArray(str.hour.Type, arr) {
		hour_func = func(arg string) bool {
			var sec_operate bool
			var min_operate bool
			switch str.sec.Type {
			case 1:
				sec_operate = ntime.Second()%cast.ToInt(arg) == 0
			case 2:
				sec_operate = ntime.Second() == cast.ToInt(str.sec.Value)
			}

			switch str.min.Type {
			case 1:
				min_operate = ntime.Minute()%cast.ToInt(arg) == 0
			case 2:
				min_operate = ntime.Minute() == cast.ToInt(str.min.Value)
			}
			return ntime.Hour()%cast.ToInt(arg) == 0 && sec_operate && min_operate
		}
	}

	if inArray(str.day.Type, arr) {
		day_func = func(arg string) bool {
			var sec_operate bool
			var min_operate bool
			var hour_operate bool
			switch str.sec.Type {
			case 1:
				sec_operate = ntime.Second()%cast.ToInt(arg) == 0
			case 2:
				sec_operate = ntime.Second() == cast.ToInt(str.sec.Value)
			}
			switch str.min.Type {
			case 1:
				min_operate = ntime.Minute()%cast.ToInt(arg) == 0
			case 2:
				min_operate = ntime.Minute() == cast.ToInt(str.min.Value)
			}
			switch str.hour.Type {
			case 1:
				hour_operate = ntime.Hour()%cast.ToInt(arg) == 0
			case 2:
				hour_operate = ntime.Hour() == cast.ToInt(str.hour.Value)
			}

			return ntime.Day()%cast.ToInt(str.day.Value) == 0 && sec_operate && min_operate && hour_operate
		}
	}

	if inArray(str.month.Type, arr) {
		var sec_operate bool
		var min_operate bool
		var hour_operate bool
		var day_operate bool
		month_func = func(arg string) bool {
			return int(ntime.Month())%cast.ToInt(str.month.Value) == 0 && sec_operate && min_operate && hour_operate && day_operate
		}
	}

	if inArray(str.week.Type, arr) {
		var sec_operate bool
		var min_operate bool
		var hour_operate bool
		var day_operate bool
		var month_operate bool

		week_func = func(arg string) bool {
			return int(ntime.Weekday())%cast.ToInt(str.week.Value) == 0 && sec_operate && min_operate && hour_operate && day_operate && month_operate
		}
	}

	if inArray(str.year.Type, arr) {
		var sec_operate bool
		var min_operate bool
		var hour_operate bool
		var day_operate bool
		var month_operate bool
		var week_operate bool
		year_func = func(arg string) bool {
			return ntime.Year()%cast.ToInt(str.year.Value) == 0 && sec_operate && min_operate && hour_operate && day_operate && month_operate && week_operate
		}
	}

	//省点时间而已.
	v, err := str.sec.judge(1, 0, 59, ntime.Second(), sec_func, &interval)
	v, err = str.min.judge(v, 0, 59, ntime.Minute(), min_func, &interval)
	v, err = str.hour.judge(v, 0, 23, ntime.Hour(), hour_func, &interval)
	v, err = str.day.judge(v, 1, 31, ntime.Day(), day_func, &interval)
	v, err = str.month.judge(v, 1, 12, int(ntime.Month()), month_func, &interval)
	v, err = str.week.judge(v, 0, 6, int(ntime.Weekday()), week_func, &interval)
	v, err = str.year.judge(v, 1, 9999, ntime.Year(), year_func, &interval)

	//每隔时间戳只能出现最多1次,否则会出错
	if interval > 1 {
		v = 0
		err = errors.New("不能同时有2个/*")
	}

	return v, err
}

func (str *Timestr) judge(validate, min, max, timestr int, closure func(arg string) bool, num *int) (v int, err error) { //参数带有函数
	switch str.Type {
	case 1: //处理*
		v = validate * 1
	case 2: //处理 */5
		if cast.ToInt(str.Value) == 0 {
			err = errors.New("没有*/0的写法")
		} else {
			if valite(str.Value, min, max) {
				if closure(str.Value) == true {
					v = validate * 1 //
				} else {
					v = validate * 0
				}
				*num++
			} else {
				err = errors.New("/*数值不在合适范围")
			}
		}
	case 3: //处理指定数
		if valite(str.Value, min, max) {
			if timestr == cast.ToInt(str.Value) {
				v = validate * 1
			} else {
				v = validate * 0
			}
		} else {
			err = errors.New("a数值不在合适范围")
		}
	case 4: //处理 指定时间段  10-23 时 ,兼容 50-10 50至10秒的写法
		arr := strings.Split(str.Value, "-")
		if valite(arr[0], min, max) && valite(arr[1], min, max) {
			if cast.ToInt(arr[0]) < cast.ToInt(arr[1]) { //10-20
				if timestr >= cast.ToInt(arr[0]) && timestr <= cast.ToInt(arr[1]) {
					v = validate * 1
				} else {
					v = validate * 0
				}
			} else { //23-10
				if (max >= timestr && timestr >= cast.ToInt(arr[0])) || (min <= timestr && timestr <= cast.ToInt(arr[1])) {
					v = validate * 1
				} else {
					v = validate * 0
				}
			}
		} else {
			err = errors.New("a-b数值不在合适范围")
		}
	case 5: //处理  10,20,30 的指定多个数值
		arr := unique(strings.Split(str.Value, ","))
		//数组要去重
		for _, val := range arr {
			if valite(val, min, max) && timestr == cast.ToInt(val) {
				v = validate * 1
				break //其中1个满足就跳出条件循环
			} else {
				v = validate * 0
			}
		}
	case 6: //处理 10-23/3 , 10-23 点 每隔3小时
		args := strings.Split(str.Value, "/")
		arr := strings.Split(args[0], "-")
		if valite(arr[0], min, max) && valite(arr[1], min, max) && valite(args[1], min, max) && closure(args[1]) == true {
			if cast.ToInt(arr[0]) < cast.ToInt(arr[1]) { //10-20
				if timestr >= cast.ToInt(arr[0]) && timestr <= cast.ToInt(arr[1]) {
					v = validate * 1
				} else {
					v = validate * 0
				}
			} else { //23-10
				if (max >= timestr && timestr >= cast.ToInt(arr[0])) || (min <= timestr && timestr <= cast.ToInt(arr[1])) {
					v = validate * 1
				} else {
					v = validate * 0
				}
			}
		} else {
			v = validate * 0
		}
	default: //不符合要求的
		err = errors.New("格式要求不正确")
	}
	return v, err
}

//验证值是否在合理范围
func valite(vstr string, min, max int) bool {
	value := cast.ToInt(vstr)
	if value <= max && value >= min {
		return true
	} else {
		return false
	}
}

//将时间字符串转成特定类型识别处理的数据
func preg(str string) Timestr {
	var rs bool

	rs, _ = regexp.MatchString("^\\*$", str)
	if rs {
		return Timestr{Type: 1, Value: ""} //每隔多少时间戳
	}
	rs, _ = regexp.MatchString("^\\*\\/\\d+$", str)
	if rs {
		reg, _ := regexp.Compile("\\d+")
		return Timestr{Type: 2, Value: reg.FindString(str)} //每隔指定时间戳
	}
	rs, _ = regexp.MatchString("^\\d+$", str)
	if rs {
		reg, _ := regexp.Compile("\\d+")
		return Timestr{Type: 3, Value: reg.FindString(str)} //第几 时间戳
	}
	rs, _ = regexp.MatchString("^\\d+-\\d+$", str)
	if rs {
		reg, _ := regexp.Compile("\\d+-\\d+")
		return Timestr{Type: 4, Value: reg.FindString(str)} //处理连续时间段10-20 和50 -20等的连续时间段
	}
	rs, _ = regexp.MatchString("^(\\d+,)+\\d+", str)
	if rs {
		reg, _ := regexp.Compile("^(\\d+,)+\\d+")
		return Timestr{Type: 5, Value: reg.FindString(str)} //处理1,2,3,4 等自定义的时间
	}

	rs, _ = regexp.MatchString("\\d+-\\d+/\\d+", str)
	if rs {
		reg, _ := regexp.Compile("^\\d+-\\d+/\\d+")
		return Timestr{Type: 6, Value: reg.FindString(str)} //处理8-20/4 等自定义的时间
	}

	return Timestr{Type: 0}
}

//数组去重
func unique(arrs []string) []string {
	for i := 0; i < len(arrs); i++ {
		for j := len(arrs) - 1; j > i; j-- {
			if arrs[i] == arrs[j] {
				arrs = append(arrs[:j], arrs[j+1:]...) //删除j下表的数组
			}
		}
	}
	return arrs
}

//任务执行
func execute(command string, args []string) {
	if *s {
		start := time.Now()
		fmt.Println("执行时间:", start)
		cmd := exec.Command(command, args...) //执行代码
		output, _ := cmd.CombinedOutput()     //输出的数据
		fmt.Println("完成时间", time.Now(), time.Now().Sub(start), string(output))
	} else {
		cmd := exec.Command(command, args...) //执行代码
		_, _ = cmd.CombinedOutput()           //不论执行结果
	}
}

//秒级计时器
func clocker(cron []string) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			//检查配置文件是否变更了,
			fileinfo, _ := os.Stat(*f)
			if fileinfo.ModTime().Unix() > conf_finfo.ModTime().Unix() { //比对修改时间,监测到配置文件变更重新加载
				conf_finfo = fileinfo
				cron = parse_conf(*f) //重新读取配置内容
				fmt.Println("重载配置")
			}
			parse(cron, time.Now())

		}
	}
}

//cli 的帮助文档
func readme() {
	fmt.Fprintf(os.Stderr, `
Usage: gocron [-s]
Options:
`)
	flag.PrintDefaults()
}

func inArray(t int, array []int) bool {
	for _, v := range array {
		if t == v {
			return true
		}
	}
	return false
}

func dealPanic() {
	if p := recover(); p != nil { //捕捉异常并处理
		switch p.(type) {
		case runtime.Error:

		case error:
			// 普通错误类型异常
			return
		case string: //普通字符
			fmt.Println(p)
		}
	}
}
