### 网址响应状态与时间监控工具

持续批量监控配置的网址响应时间与状态，输出为CSV文件，借助Excel的图表可以方便观察网络稳定性。

## 命令参数：
```bash
使用说明: hrt  参数选项
欢迎使用 HTTP 请求时间监视工具
更多信息：http://github.com/csg800/tools

参数选项:
  -c string  配置文件路径 (default "hrt.json")
  -h         显示应用帮助信息并退出
  -i uint    请求间隔时间（秒） (default 60)
  -o string  数据输出路径 (default "hrt.csv")
  -t uint    请求超时时间（秒） (default 5)
```

## hrt.json 配置文件
配置文件采用JSON格式，每个要监控的网址用一个对象表示，目前没有限制最多监控的网址数量，但过多的网址会影响程序整体响应。

配置文件格式实例：
```json
[
    {"label":"网址标题", "url":"要监控的网址", "param":"POST方式请求参数，如果为空将用GET请求"}
]
```

## 输出结果
```csv
请求时间,大鱼(时间),大鱼(状态),1688(时间),1688(状态),天猫(时间),天猫(状态),淘宝(时间),淘宝(状态),百度(时间),百度(状态),京东(时间),京东(状态)
2020-02-10 13:49:16,0.448,400,5.000,200,5.001,0,5.000,0,5.001,200,5.000,200
2020-02-10 13:49:18,3.721,400,5.001,0,5.001,0,5.000,0,5.001,0,5.001,0
2020-02-10 13:49:20,5.001,0,5.001,0,5.001,0,5.001,0,5.001,0,5.001,0
```