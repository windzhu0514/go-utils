# gocrypto
go crypto library

## 加密算法
AES支持三种长度的密钥：128位，192位，256位，即AES128，AES192，AES256
AES256安全性最高
AES128性能最高

- [ ] IV支持不传入值，默认生成且返回
- [ ] 生成随机key，根据不同算法生成

## IV
// 在除ECB以外的所有加密方式中，都需要用到IV对加密结果进行随机化。在使用同一种加密同一个密钥时不应该使用相同的IV，否则会失去一定甚至全部的安全性。如果到这里还不明白的话没关系，后面还会继续讲到

## BlockMode
### ECB 电码本模式    Electronic Codebook Book
ECB模式有一个显著的安全问题：如果使用相同的密钥，
 那么相同的明文块就会生成相同的密文块，不能很好的隐藏数据模式，因此，在密码协议中不建议使用ECB模式

### CBC 密码分组链接模式    Cipher Block Chaining
// BlockModeCBC CBC模式加密过程是串行的，不能并行化，速度比较慢，但是解密可以并行。另外，如果密文的某一位被修改了，只会使这个密文块所对应的明文块完全改变并且改变下一个明文块的对应位，安全性仍然有一定的欠缺。

### CFB 密码反馈模式    Cipher FeedBack
### CTR 计算器模式    Counter
### OFB 输出反馈模式    Output FeedBack


参考资料：
- [AES加密(1): 基本AES算法] (https://zhuanlan.zhihu.com/p/131324301)
- [什么是AES算法？（整合版）—— 漫画算法](https://www.cxyxiaowu.com/3239.html)