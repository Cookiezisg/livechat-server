# livechat-server
A Go practice project implementing a simple multi-user network chatroom.

实现功能

1.上线下线
2.聊天，其他人，自己可以看到聊天消息
3.查询当前聊天室用户名字
4.可以修改自己名字
5.超时踢出

技术点分析：
a. socket tcp编程
b. map结构存储用户，遍历，删除
c. go 程，channel
d. select （超时退出，主动退出）
d. timer定时器

实现基础

1.思路分析 