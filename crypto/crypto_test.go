package crypto

//func TestAESECBEncrypt(t *testing.T) {
//	cipher := Cipher{
//		Algorithm:   AlgorithmAES,
//		BlockMode:   BlockModeECB,
//		PaddingMode: PaddingModePKCS7,
//		Key:         []byte("0123456789012345"),
//	}
//
//	cipherTxt, err := cipher.Encrypt([]byte("hello,go"))
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println(hex.EncodeToString(cipherTxt))
//}
//
//func TestAESECBDecrypt(t *testing.T) {
//	cipher := Cipher{
//		Algorithm:   AlgorithmAES,
//		BlockMode:   BlockModeECB,
//		PaddingMode: PaddingModePKCS7,
//		Key:         []byte("0123456789012345"),
//	}
//
//	cipherTxt, _ := hex.DecodeString("4b5dfeae9217c720218bc88c59125cf5")
//	plainTxt, err := cipher.Decrypt(cipherTxt)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	fmt.Println(string(plainTxt))
//}
//
//func TestDesECBEncrypt(t *testing.T) {
//	cipher := Cipher{
//		Algorithm:   AlgorithmDES,
//		BlockMode:   BlockModeECB,
//		PaddingMode: PaddingModePKCS7,
//		Key:         []byte("01234567"),
//	}
//
//	cipherTxt, err := cipher.Encrypt([]byte("hello,go"))
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println(hex.EncodeToString(cipherTxt))
//}
//
//func TestDESECBDecrypt(t *testing.T) {
//	cipher := Cipher{
//		Algorithm:   AlgorithmDES,
//		BlockMode:   BlockModeECB,
//		PaddingMode: PaddingModePKCS7,
//		Key:         []byte("01234567"),
//	}
//
//	cipherTxt, _ := hex.DecodeString("1089000d26269cd808bb5db6b37c06d7")
//	plainTxt, err := cipher.Decrypt(cipherTxt)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	fmt.Println(string(plainTxt))
//}
//
//func TestTripleDESEncrypt(t *testing.T) {
//	cipher := Cipher{
//		Algorithm:   AlgorithmTripleDES,
//		BlockMode:   BlockModeECB,
//		PaddingMode: PaddingModePKCS7,
//		Key:         []byte("012345670123456701234567"),
//	}
//
//	cipherTxt, err := cipher.Encrypt([]byte("hello,go"))
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println(hex.EncodeToString(cipherTxt))
//}
//
//func TestTripleDESDecrypt(t *testing.T) {
//	cipher := Cipher{
//		Algorithm:   AlgorithmTripleDES,
//		BlockMode:   BlockModeECB,
//		PaddingMode: PaddingModePKCS7,
//		Key:         []byte("012345670123456701234567"),
//	}
//
//	cipherTxt, _ := hex.DecodeString("1089000d26269cd808bb5db6b37c06d7")
//	plainTxt, err := cipher.Decrypt(cipherTxt)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	fmt.Println(string(plainTxt))
//}
