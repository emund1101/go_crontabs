# go_crontabs
go crontab 秒级定时器,代替linux下的crontab

#### 文件说明
```
--config 		配置文件
--crontab 		二进制执行文件
--crontab.go 	源代码
```
#### 应用参数说明
```
crontab -h
  -c string
        指定配置文件 (default "D:\\go\\demo\\crontab/config")
  -d    是否守护进程执行
  -h    帮助信息
  -kill
        杀掉守护进程
  -s    是否调试输出
  -stat 查看实时统计应用程序信息   http://localhost:6060/debug/statsviz/ 
  ```
  
#### 配置设置如下
```
# 1行里面 /1  只能出现1次
# * * 23-7 23点至7点,每小时1次
# * */5 * * * * * 每5分钟执行一次
# * 6,10 * * * * *  // 6和10分钟里的每一秒都执行,所以最好指定秒数, 最好是 0 6,10 * * * *
# 2 10-20/5 * * * * *  // 10-20 每5分钟 的的第二秒执行
# * 1,2,3 * * * * * //逢1,2,3分钟执行
# Example of job definition:
# .---------------- second (0 - 59)     */1 - */59  每隔1-59s
# |  .---------------- minute (0 - 59)  */1 - */59  每隔1-59m
# |  |  .------------- hour (0 - 23)     */1 - */23  每隔1-23h
# |  |  |  .---------- day of month (1 - 31)    */1 - */30
# |  |  |  |  .------- month (1 - 12) OR jan,feb,mar,apr ...     */1 - */11
# |  |  |  |  |  .---- day of week (0 - 6) (Sunday=0 or 7) OR sun,mon,tue,wed,thu,fri,sat   */1- */52 每隔1周
# |  |  |  |  |  |  .--year
# *  *  *  *  *  *  *
#50-20 * * * * * * echo asd

````