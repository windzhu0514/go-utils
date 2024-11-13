package signature

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	uuid "github.com/satori/go.uuid"

	"github.com/windzhu0514/go-utils/utils"
)

type Option func(*options)

type options struct {
	signKey string

	tcSignKey   string
	tcPartnerID string
}

func WithSignKey(key string) Option {
	return func(o *options) {
		o.signKey = key
	}
}

func WithTCPartnerID(partnerID string) Option {
	return func(o *options) {
		o.tcPartnerID = partnerID
	}
}

func WithTCSignKey(key string) Option {
	return func(o *options) {
		o.tcSignKey = key
	}
}

func Check(opts ...Option) middleware.Middleware {
	var op options
	for _, opt := range opts {
		opt(&op)
	}

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				authUser := tr.RequestHeader().Get("Auth-User")
				method := tr.RequestHeader().Get("Method")
				timestamp := tr.RequestHeader().Get("Timestamp")
				signature := tr.RequestHeader().Get("Signature")

				reqBody := utils.JsonMarshalString(req)
				sign := utils.MD5(authUser + method + timestamp + reqBody + op.signKey)
				if sign != signature {
					return nil, errors.New("签名校验失败")
				}
			}

			return next(ctx, req)
		}
	}
}

// FilterCheck 内部服务签名验证
// 和WithSignKey一起使用
func FilterCheck(logger log.Logger, opts ...Option) http.FilterFunc {
	var op options
	for _, opt := range opts {
		opt(&op)
	}

	log := log.NewHelper(logger, log.WithMessageKey("message"))

	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if strings.HasPrefix(r.RequestURI, "/train") { // TC
				jsonStr := r.FormValue("jsonStr")
				log.Debug("train jsonStr:" + jsonStr)

				//var tcReq v1.TCQCommonRequest
				//if err := json.Unmarshal([]byte(jsonStr), &tcReq); err != nil {
				//	log.Errorf("train jsonStr unmarshal error: %s", err.Error())
				//
				//	resp := &v1.TCQCommonReply{
				//		Code: tcdefine.ErrTCQOrder999.Code,
				//		Msg:  err.Error(),
				//	}
				//
				//	_, _ = w.Write(utils.JsonMarshalByte(resp))
				//	return
				//}

				//if tcReq.Partnerid != "test" {
				//	sign := utils.MD5(tcReq.Partnerid + tcReq.Method + tcReq.Reqtime + utils.MD5(op.tcSignKey))
				//	if sign != tcReq.Sign {
				//		log.Errorf("train sign error partnerid:%s method:%s reqtime:%s",
				//			tcReq.Partnerid, tcReq.Method, tcReq.Reqtime)
				//
				//		resp := &v1.TCQCommonReply{
				//			Code: tcdefine.ErrTCQOrder999.Code,
				//			Msg:  "签名验证失败",
				//		}
				//
				//		_, _ = w.Write(utils.JsonMarshalByte(resp))
				//
				//		return
				//	}
				//}
			} else { // 内部服务
				//bodyBytes, err := ioutil.ReadAll(r.Body)
				//log.Debug("jsonStr:" + string(bodyBytes))
				//
				//_ = r.Body.Close()
				//if err != nil {
				//	log.Errorf("signature read body error: %s", err.Error())
				//
				//	resp := &v1.CommonReply{
				//		StatusCode:   define.ErrInternalError.Code,
				//		StatusReason: define.ErrInternalError.Msg,
				//	}
				//
				//	_, _ = w.Write(utils.JsonMarshalByte(resp))
				//	return
				//}
				//
				//r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				//
				//authUser := r.Header.Get("Auth-User")
				//method := r.Header.Get("Method")
				//timestamp := r.Header.Get("Timestamp")
				//signature := r.Header.Get("Signature")
				//
				//traceId := r.Header.Get("Trace-Id")
				//w.Header().Set("Trace-Id", traceId)
				//
				//if authUser != "test" {
				//	sign := utils.MD5(authUser + method + timestamp + string(bodyBytes) + op.signKey)
				//	if sign != signature {
				//		log.Errorf("sign error Trace-Id:%s Auth-User:%s Method:%s Timestamp:%s Signature:%s body:%s",
				//			traceId, authUser, method, timestamp, signature, string(bodyBytes))
				//
				//		resp := &v1.CommonReply{
				//			StatusCode:   define.ErrSignatureError.Code,
				//			StatusReason: define.ErrSignatureError.Msg,
				//		}
				//
				//		_, _ = w.Write(utils.JsonMarshalByte(resp))
				//		return
				//	}
				//}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AddHttpHead(user, key string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				if ht, ok := tr.(http.Transporter); ok {
					header := ht.RequestHeader()
					method := func() string {
						words := strings.Split(ht.PathTemplate(), "/")
						if len(words) > 0 {
							return words[len(words)-1]
						}
						return ht.PathTemplate()
					}()

					body := utils.JsonMarshalString(req)

					timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
					sign := utils.MD5(user + method + timestamp + body + key)
					header.Set("Auth-User", user)
					header.Set("Method", method)
					header.Set("Timestamp", timestamp)
					header.Set("Signature", sign)
					header.Set("Trace-Id", uuid.NewV4().String())
				}
			}
			return handler(ctx, req)
		}
	}
}
