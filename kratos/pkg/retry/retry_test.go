package retry

import (
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestDecodeBody(t *testing.T) {
	r := Retry{
		log: log.NewHelper(log.DefaultLogger),
	}

	var msg CallbackMsg
	msg.Body = []byte("ZGF0YT0lN0IlMjJzdWNjZXNzJTIyJTNBJTIyZmFsc2UlMjIlMkMlMjJjb2RlJTIyJTNBJTIyOTk5JTIyJTJDJTIybXNnJTIyJTNBJTIyJUU1JThGJTk2JUU2JUI2JTg4JUU1JUE0JUIxJUU4JUI0JUE1JTIyJTJDJTIycmVxdG9rZW4lMjIlM0ElMjJiMjE1Y2FhNC1kOTBiLTRhZDgtOGUwMi0zYzIyZjhjYmJiYTYlMjIlMkMlMjJxb3JkZXJpZCUyMiUzQSUyMlRHVF9TNkNGMThCODU1MzFFN0FCMDIyMTklMjIlMkMlMjJ0cmFuc2FjdGlvbmlkJTIyJTNBJTIyNjFjZTZmN2NhZWVkNiUyMiUyQyUyMm9yZGVyc3VjY2VzcyUyMiUzQSUyMkZhbHNlJTIyJTJDJTIyY2hlY2klMjIlM0ElMjI2MDc0JTJDSzk2NjQlMjIlMkMlMjJmcm9tX3N0YXRpb25fbmFtZSUyMiUzQSUyMiVFNSVCOSVCMyVFNSU4NyU4OSUyMiUyQyUyMnRvX3N0YXRpb25fbmFtZSUyMiUzQSUyMiVFNSVCOSVCMyVFNSU4NyU4OSVFNSU4RCU5NyUyMiUyQyUyMnRyYWluX2RhdGUlMjIlM0ElMjIyMDIyLTAxLTE2JTIyJTJDJTIyc3RhcnRfdGltZSUyMiUzQSUyMjEzJTNBMTAlMjIlMkMlMjJhcnJpdmVfdGltZSUyMiUzQSUyMjEzJTNBMjUlMjIlMkMlMjJwYXNzZW5nZXJzJTIyJTNBJTVCJTdCJTIycGFzc2VuZ2VyaWQlMjIlM0ExMzEzMTc1NiUyQyUyMnBhc3NlbmdlcnNlbmFtZSUyMiUzQSUyMiVFNiU5RCU4RSVFNyVCRCU5MSVFOCU5OSVCOSUyMiUyQyUyMnBhc3Nwb3J0c2VubyUyMiUzQSUyMjMyMTAyMzE5OTUwODE0NTAyNiUyMiUyQyUyMnBhc3Nwb3J0dHlwZXNlaWQlMjIlM0ElMjIxJTIyJTJDJTIycGFzc3BvcnR0eXBlc2VpZG5hbWUlMjIlM0ElMjIlRTQlQkElOEMlRTQlQkIlQTMlRTglQkElQUIlRTQlQkIlQkQlRTglQUYlODElMjIlMkMlMjJwaWFvdHlwZSUyMiUzQSUyMjElMjIlMkMlMjJwaWFvdHlwZW5hbWUlMjIlM0ElMjIlRTYlODglOTAlRTQlQkElQkElRTclQTUlQTglMjIlMkMlMjJib3JuRGF0ZSUyMiUzQSUyMjE5OTUtMDgtMTQlMjIlMkMlMjJleHBpcnlEYXRlJTIyJTNBJTIyMTkwMC0wMS0wMSUyMiUyQyUyMmNvdW50cnlDb2RlJTIyJTNBJTIyQ04lMjIlMkMlMjJlbmNNb2JpbGVObyUyMiUzQSUyMjZGMjY2QjQ3MjE5MTYyMERFODMwMzA3OEY3MzlGMzhCJTIyJTdEJTVEJTJDJTIyYWN0aW9uX3Jlc3VsdF90eXBlJTIyJTNBMiUyQyUyMnBhcnRuZXJpZCUyMiUzQSUyMmdzdHJhaW4lMjIlMkMlMjJyZXF0aW1lJTIyJTNBJTIyMjAyMTEyMzExMDQ5MTIwMDAlMjIlMkMlMjJzaWduJTIyJTNBJTIyZmQ4ZmFhODE1ZmY2ZGUxNmMxMTNlYjA1NmQ4ZmIxMjAlMjIlMkMlMjJzdWNjZXNzT3BlVHlwZSUyMiUzQSUyMlklMjIlMkMlMjJtZXRob2QlMjIlM0ElMjJxdHJhaW5fb3JkZXIlMjIlN0Q=")
	fmt.Println(r.decodeBody(&msg))
}