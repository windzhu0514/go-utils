package httpclient

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"testing"
)

func TestGet(t *testing.T) {
	code, resp, err := Get("http://www.baidu.com")

	fmt.Println(code, string(resp), err)
}

func TestGet2(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://www.baidu.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("fake-head", "11111")
	req.Header.Add("fake-head", "2222")
	dump, err := httputil.DumpRequest(req, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(dump))
}

func TestSetCookie(t *testing.T) {
	client := NewClient()
	var cookies []*http.Cookie
	cookies = append(cookies, &http.Cookie{Name: "Channel", Value: "Android"})
	cookies = append(cookies, &http.Cookie{Name: "lang", Value: "Android"})
	cookies = append(cookies, &http.Cookie{Name: "issu", Value: "Android"})
	cookies = append(cookies, &http.Cookie{Name: "issu", Value: "Android"})
	client.SetCookies(cookies)
	dumplicatMap := make(map[string]bool)
	for _, cc := range client.cookies {
		fmt.Println("name: ", cc.Name)
		if dumplicatMap[cc.Name] {
			t.Fail()
		}
		dumplicatMap[cc.Name] = true
	}
}
