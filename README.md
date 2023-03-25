# bamfaRemote
通过 巴法云平台接口 实现小爱语音 开关机

config.yaml.template 改为 config.yaml 后配置自己的信息


```
bamfa:
  #巴法平台私钥
  uid: 22cc12345677777ccc
  topic: PC001
wol:
  #延时唤醒  配合智能插座使用，可设置延迟，单位秒
  delay: 0
  #需要控制的局域网ip
  ip: 192.168.31.47
  #需要控制的电脑mac
  mac: 00-00-00-00-00-00
  #是否使用Etherwake唤醒，如果使用Etherwake，以下面的参数为准
  isEtherwake: false
  #网络接口,可以参考Etherwake使用说明
  ifname: br-lan
  #mac或ip,例如 p: 192.168.1.1
  p: 00:22:44:66:88:aa

```

如果Magic packet不生效，可用安装Etherwake，并设置isEtherwake: true

关机需要另外一个程序配合

[远程休眠服务](https://github.com/xv-chang/remoteShutdown) 




### 注意：

>1.自行设置网络唤醒，可参考 [WOL网络唤醒](https://www.jianshu.com/p/95e1a22d1e9f)

>2.巴法平台 创建 TCP创客云 topic名称 PC001 ,记得设置昵称，这个昵称是小爱上用来识别的，我这里叫电脑，小爱上可通过：
“打开电脑”
“关闭电脑”
语音来控制


>3.小爱音响->智能家居->右上+号->其他平台设备->巴法->登录绑定你的设备

>4.程序请放在一个一直运行的设备上，我这里是放在小米路由器上运行的

