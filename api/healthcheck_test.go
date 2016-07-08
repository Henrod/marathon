package api_test

//
// import (
// 	"net/http"
// 	"testing"
//
// 	. "github.com/franela/goblin"
// 	. "github.com/onsi/gomega"
// )
//
// func TestHealthcheckHandler(t *testing.T) {
// 	g := Goblin(t)
//
// 	// special hook for gomega
// 	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })
//
// 	g.Describe("Healthcheck Handler", func() {
// 		g.It("Should respond with default WORKING string", func() {
// 			a := GetDefaultTestApplication()
// 			res := Get(a, "/healthcheck", t)
//
// 			res.Status(http.StatusOK).Body().Equal("WORKING")
// 		})
//
// 		g.It("Should respond with customized WORKING string", func() {
// 			a := GetDefaultTestApplication()
// 			a.Config.SetDefault("healthcheck.workingText", "OTHERWORKING")
// 			res := Get(a, "/healthcheck", t)
//
// 			res.Status(http.StatusOK).Body().Equal("OTHERWORKING")
// 		})
// 	})
// }
