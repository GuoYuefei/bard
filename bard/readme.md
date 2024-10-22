### TODO 列表

+ [x] 基础代理功能
	- [x] TCP代理
	- [x] UDP代理
	- [x] timeoout 判断连
	- [x] 代理权限认证	 		
+ [x] 日志和debug输出控制
+ [x] 插件系统建立——前期有预留方法		// 下一个分支
+ [ ] 非移动端的客户端（往后可带web界面）	// 下下分支
+ [ ] 移动端 全平台 使用flutter框架 			
+ [ ] 更新系统	
+ [ ] 加密等争对数据流的插件			//在基础客户端完成后 新开项目
+ [ ]  反向代理实现					// 可能新开项目 
------
> 基本功能建立后发布第一个beta版本v0.1 or alpha v0.0.1

---
插件方面的相关设计草稿
## 插件

### 1. 混淆与加密插件

#### 1.1 x 不实现 修改socks握手协议 socks-b1

**不区分用户来商议接下来的通讯插件**

第一次握手协议无安全保障，此内容与协议内容无关。下方提供对第一次握手安全通讯保障并非协议内容。实现时可自行思考。 本协议只用于如何使c/s使用相同插件，以保证接下来可以正常交流。

+ client

  根据配置文件中的用户的插件列表，发送消息。第一条消息格式如下

  |       插件数量（byte） pn or socks协议版本 Ver        |                 插件id长度+插件id(变长) pid   变长可不存在                 |    支持的验证数量（byte） n    | 验证方式（变长）method |
  | :---------------------------------------------------: | :----------------------------------------------------------: | :--: | :---------------------------: |
  | 表示插件数量，一般只有一个。0x1X代表一个，0x2X代表2个 | 第一个字节是后面插件id的长度。当pn为多个时该字段会重读pn次。当pn为0x0X，该字段不存在。 |    socks协议，客户端支持验证数量 |   socks协议验证方式    |
  |                         0x25                          |             0x02 0x01 0x00, 0x03 0x00 0x00 0x00              |            0x02              |       0x00 0x02        |
  |                         0x05                          |                                                              |                  0x02              |       0x00 0x02        |
  

以上以socks5协议为例子，并且该协议可以兼容socks5协议。当pn字段为0x0X，就为X版本socks，此时pn字段就是ver字段。

客户端发的消息是裸露的，所以第一条消息应该使用加密方式。客户端和服务器端在此之前应该配备一个公私钥，用于第一条消息的非对称加密。

客户端的公钥可以由配置文件给出文件位置信息。若不配置，则认为是明文传输（为了兼容socks协议）。

+ server

  在握手的第一次连接时确认使用的插件。但是此次通讯没有安全保障，由于信息量较小，可由配置文件给出私钥文件位置，若不给出配置信息，则认为是明文传输（为了兼容socks协议）。

  |                 插件数量pn or Ver                  |                            method                            |
  | :------------------------------------------------: | :----------------------------------------------------------: |
  | 回复插件数量和版本信息 结构于客户端握手pn字段相同  | 服务器回复状态  遵照socks5协议 rfc1928   根据协议内容在method的0x80-0xfe可以用户自定义。 0xfe做客户端发送来的插件和服务器端拥有的不匹配回复 |
  | 0x25 客户端请求需要要两个插件， 使用使用socks5握手 |                0x02 username/password验证方式                |
  |                        0x25                        |                  0xfe 插件无法匹配错误回复                   |
  |                        0x15                        |                    0xff 无此验证方式回复                     |
  |                        0x05                        |                      0x00 NO ACCEPTABLE                      |

#### 1.2 修改user/password子协议 up-b2

**区分用户来商议接下来的用户通讯插件**

这种情况需要修改权限认证子协议。修改子协议为0x02，即username/password方式。

+ 客户端

  |                          pn \| Ver                           | **插件id长度+插件id(变长) pid   长度可为0 | len of username |     username     | len of password | password  |
  | :----------------------------------------------------------: | :---------------------------------------: | :-------------: | :--------------: | :-------------: | :-------: |
  | 插件数量 or 子协议版本 0x01 前四位代表插件长度，后四位代表子协议版本 |          格式： len，len*{0x00}           | 子协议 rfc1929  |     Rfc1929      |     rfc1929     |  rfc1929  |
  |                             0x21                             | 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00  |      0x03       | 0x00, 0x00, 0x00 |      0x0c       | 12*{0x00} |

  

兼容rfc1929，兼容原理与1. 1中所述相同。

+ 服务器端

  |     pn \| Ver      |                            status                            |
  | :----------------: | :----------------------------------------------------------: |
  | 与客户端发来的相同 | 0x00代表成功， 按rfc1929，非0x00都为拒绝。这里0xfe表示插件无法匹配错误 |
  |        0x21        |                             0xfe                             |

   

### 2. 传输控制子协议 

该协议与混淆与加密插件自由绑定，并且应该由插件形式自由给出。

但是刚开始握手过程应该有一个默认协议控制传输过程。（由配置文件配置， 如果握手时有加密，则必须配置控制子协议）

暂时想到需要的控制信息为一次发送数据块长度。



### 3. 自定义权限认证协议

1. 第一次握手 say hello

+ client

| 子协议版本 | 非对称算法代号 |
| ---------- | -------------- |
| 1 byte     | 1 byte         |

 

+ server

| 子协议版本 | status                                                |
| ---------- | ----------------------------------------------------- |
| 1 byte     | 0x00 有此算法支持, 0xfe 无此算法支持, 0xff 无此子协议 |

2. 第二次握手 验证

+ client

| 子协议版本 | 对应算法和秘钥的加密块大小的数据 数据全为0x00的填充 |
| ---------- | --------------------------------------------------- |
| 1 byte     |                                                     |

+ server

| 子协议版本 | status               |
| ---------- | -------------------- |
| 1 byte     | 0x00 成功 0x0ff 拒绝 |

非对称秘钥作为验证的钥匙

具体实现应该比之上面的协议，靠后



### 4. 插件配置

```yaml
# Global Plugin Config
plugin: 
  - xxx
  - xxx
TCSP: xxx
# User Config
users:
  - 
    username: xxx
    password: xxx
    plugin:
      - xxx
      - xxx
    TCSP: xxx
  - 
    username: xxx
    password: xxx
    plugin:
      - xxx
      - xxx
    TCSP: xxx

```
### 草稿

先找到所有插件，形成一个集合。集合类型已存在bard.Plugins.

要使用时，根据plugin的id值找到plugin，形成一个小集合 类型依旧为bard.Plugins