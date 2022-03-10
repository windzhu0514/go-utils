package crypto

//// des加密
//func DesEncrypt(plainTxt, key []byte, padding Padding) ([]byte, error) {
//	block, err := des.NewCipher(key)
//	if err != nil {
//		return nil, err
//	}
//
//	if padding != nil {
//		plainTxt = padding.Padding(plainTxt, block.BlockSize())
//	}
//
//	// origData = ZeroPadding(origData, block.BlockSize())
//	blockMode := cipher.NewCBCEncrypter(block, key)
//	crypted := make([]byte, len(plainTxt))
//	// 根据CryptBlocks方法的说明，如下方式初始化crypted也可以
//	// crypted := origData
//	blockMode.CryptBlocks(crypted, plainTxt)
//	return crypted, nil
//}
//
//// des解密
//func DesDecrypt(cipherTxt, key []byte, padding Padding) ([]byte, error) {
//	block, err := des.NewCipher(key)
//	if err != nil {
//		return nil, err
//	}
//
//	if len(cipherTxt)%block.BlockSize() != 0 {
//		return nil, errors.New("gocrypto/cipher: input not full blocks")
//	}
//
//	blockMode := cipher.NewCBCDecrypter(block, key)
//
//	dst := make([]byte, len(cipherTxt))
//	blockMode.CryptBlocks(dst, cipherTxt)
//
//	if padding != nil {
//		dst = padding.UnPadding(dst)
//	}
//
//	return dst, nil
//}
//
//func TripleCBCEncrypt(plainTxt, key []byte, padding Padding) ([]byte, error) {
//	block, err := des.NewTripleDESCipher(key)
//	if err != nil {
//		return nil, err
//	}
//
//	if padding != nil {
//		plainTxt = padding.Padding(plainTxt, block.BlockSize())
//	}
//
//	// The IV needs to be unique, but not secure. Therefore it's common to
//	// include it at the beginning of the ciphertext.
//	dst := make([]byte, block.BlockSize()+len(plainTxt))
//	iv := dst[:block.BlockSize()]
//	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
//		return nil, err
//	}
//
//	blockMode := cipher.NewCBCEncrypter(block, iv)
//	blockMode.CryptBlocks(dst, plainTxt)
//
//	return dst, nil
//}
//
//func TripleCBCDecrypt(cipherTxt, key []byte, padding Padding) ([]byte, error) {
//	block, err := des.NewTripleDESCipher(key)
//	if err != nil {
//		return nil, err
//	}
//
//	if len(cipherTxt) < block.BlockSize() {
//		return nil, errors.New("ciphertext too short")
//	}
//
//	if len(cipherTxt)%block.BlockSize() != 0 {
//		return nil, errors.New("gocrypto/cipher: input not full blocks")
//	}
//
//	iv := cipherTxt[:block.BlockSize()]
//	cipherTxt = cipherTxt[block.BlockSize():]
//
//	blockMode := cipher.NewCBCDecrypter(block, iv)
//
//	dst := make([]byte, len(cipherTxt))
//	blockMode.CryptBlocks(dst, cipherTxt)
//
//	if padding != nil {
//		dst = padding.UnPadding(dst)
//	}
//
//	return dst, nil
//}
