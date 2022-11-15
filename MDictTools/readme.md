### MDict 词典源文件整理工具
辅助 MDict 词典优化的小工具，当前实现的功能：  
* 词典源文件整理：清理没用的空格与换行、清理不要的标签、自动关闭没有关闭的标签  
* 词典引用的 CSS 整理：根据词典源文件中的标签名、ID、className，从源CSS文件生成一份被用到的精简版CSS文件  
* 合并两本词典（未完成）：合并两本词典源的源文件，当前是定制开发，没有通用性，不能直接使用  

命令参数：
```bash
使用说明: tidy -e <启动入口> -c <配置文件>
欢迎使用 MDict 词典源文件整理工具
更多信息：https://github.com/csg2008/tools/tree/master/MDictTools

参数选项:
  -c string 整理规则文件路径
  -e string 工具启动入口
  -h        显示应用帮助信息并退出

启动入口：
        tidy     词典源文件整理
        css      词典引用的 CSS 整理
        merge    合并两本词典
```

## tidy 词典源文件整理
实现的功能：   
* 替换掉指定的内容  
* 清理不需要的标签  
* 清理不正确关闭的标签  
* 自动关闭未关闭的标签  

tidy.json 配置实例：
```json
{
    "DumpWord": false,
    "Input": "漢字音形義字典20191017.txt",
    "Output": "",
    "SkipWord": null,
    "SkipContent": null,
    "Prepare": [
        [" alt=\"\"", ""],
        [" title=\"\"", ""],
        [" class=\" \"", ""],
        [" style=\"\"", ""]
    ],
    "Post": [
        ["<A", "<a"]
    ],
    "Rules": [{
        "Selector": "",
        "Action": "Tidy",
        "Param": {
            "SkipComment": true,
            "EscapeBracket": false,
            "Drop": ["textarea"],
            "UnWrap": ["html", "body", "head", "meta", "noscript"]
        }
    }]
}
```

配置文件说明：  
Input: 词典源文件路径  
Output: 输出的词典源文件路径 ，如果为空自动在输入源文件扩展名前加上 new 作为新文件  
SkipWord: 需要忽略的词头关键词,  
SkipContent: 需要忽略的正文关键词,  
Prepare: 规则执行前的关键词替换  
Post: 规则执行后的关键词替换  
Rules: [{  
Selector: CSS选择器  
Action: Tidy,  
Param: {  
    SkipComment: 是否忽略HTML注释,  
    EscapeBracket: 是否转义HTML符号,  
    Drop: 标签删除规则，匹配此规则的标签及子标签都会被删除,  
    UnWrap: 标签删除规则，匹配此规则的标签会被删除但子标签会被保留，   
}  
}]  

## css 词典引用的 CSS 整理
实现的功能：  
* 清理未被使用的 CSS 样式  
* 生成词典源文件标签概览  

css.json 配置实例：
```json
{
    "Source": "Thesaurus.new.txt",
    "CSS": "Thesaurus.css"
}
```

配置文件说明：  
Source   词典源文件路径  
CSS        词典样式文件路径  
Output   输出的CSS文件路径 ，如果为空自动在输入源CSS文件扩展名前加上 new 作为新文件  