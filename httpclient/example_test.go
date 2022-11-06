package httpclient_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/windzhu0514/go-utils/httpclient"
)

func ExamplePathEscape() {
	v := url.Values{}
	v.Set("name", "Ava")
	v.Add("friend", "Jess")

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.FormValue("name"), r.FormValue("friend"))
		_, _ = w.Write([]byte("OK"))
	}))

	statusCode, resp, err := httpclient.Post(s.URL, httpclient.MIMEPOSTForm, v.Encode())
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(statusCode, string(resp))
	// Output:
	// Ava Jess
	// 200 OK
}
