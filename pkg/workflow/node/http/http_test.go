package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/port"
)

func startTestServer(t *testing.T) (testServer *http.Server, freePort int, success bool) {
	router := gin.Default()
	router.GET("/happyPass", func(c *gin.Context) {
		for k := range c.Request.Header {
			if strings.ToLower(k) == "content-length" {
				continue
			}
			c.Writer.Header().Set(k, c.Request.Header.Get(k))
		}
		c.JSON(200, map[string]any{
			"data":  "pong",
			"query": buildQuery(c.Request),
		})
		c.Abort()
	})
	router.GET("/contentType", func(c *gin.Context) {
		c.String(200, c.Request.Header.Get("Content-Type"))
	})
	router.GET("/sleep1", func(c *gin.Context) {
		time.Sleep(1 * time.Second)
		c.String(200, "OK")
	})
	router.GET("/sizedBody", func(c *gin.Context) {
		sizeStr := c.DefaultQuery("size", "10")
		size, _ := strconv.Atoi(sizeStr)
		sizedBody := bytes.Repeat([]byte("a"), size)
		c.Writer.WriteString(`{"data": "`)
		c.Writer.Write(sizedBody)
		c.Writer.WriteString(`"}`)
		c.AbortWithStatus(200)
	})
	router.GET("/empty", func(c *gin.Context) {
		c.AbortWithStatus(200)
	})

	freePort, err := port.GetFreePort()
	if err != nil {
		return
	}
	testServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", freePort),
		Handler: router,
	}
	go func() {
		testServer.ListenAndServe()
	}()
	err = port.WaitPort(freePort, 2*time.Second)
	if err != nil {
		return
	}

	return testServer, freePort, true
}

func Test_makeRequest_run(t *testing.T) {
	testServer, port, success := startTestServer(t)
	if !success {
		t.Skip()
	}
	defer testServer.Shutdown(context.Background())

	t.Run("happy pass", func(t *testing.T) {
		mr := makeRequest{
			Method: "GET",
			URL:    fmt.Sprintf("http://localhost:%d/happyPass", port),
			Headers: []kv{
				{
					Key:   "foo",
					Value: "bar",
				},
			},
			Queries: []kv{
				{
					Key:   "foo",
					Value: "bar",
				},
			},
			BodyType:    "raw",
			ContentType: "json",
			RawContent:  "ping",
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 200, output.StatusCode)
		assert.Equal(t, "bar", output.Header["Foo"])
		assert.Equal(t, map[string]any{
			"data":  "pong",
			"query": map[string]interface{}{"foo": "bar"},
		}, output.Body)
	})

	t.Run("add content-type json to request header automatically", func(t *testing.T) {
		mr := makeRequest{
			Method:      "GET",
			URL:         fmt.Sprintf("http://localhost:%d/contentType", port),
			BodyType:    "raw",
			ContentType: "json",
			RawContent:  `{"foo": "bar"}`,
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "application/json", output.Body)
	})

	t.Run("add content-type text/plain to request header automatically", func(t *testing.T) {
		mr := makeRequest{
			Method:      "GET",
			URL:         fmt.Sprintf("http://localhost:%d/contentType", port),
			BodyType:    "raw",
			ContentType: "text",
			RawContent:  `123`,
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "text/plain", output.Body)
	})

	t.Run("add content-type form-urlencoded to request header automatically", func(t *testing.T) {
		mr := makeRequest{
			Method:   "GET",
			URL:      fmt.Sprintf("http://localhost:%d/contentType", port),
			BodyType: "formUrlencoded",
			FormContent: []kv{
				{
					Key:   "foo",
					Value: "bar",
				},
			},
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "application/x-www-form-urlencoded", output.Body)
	})

	t.Run("add content-type form-data to request header automatically", func(t *testing.T) {
		mr := makeRequest{
			Method:   "GET",
			URL:      fmt.Sprintf("http://localhost:%d/contentType", port),
			BodyType: "formData",
			FormContent: []kv{
				{
					Key:   "foo",
					Value: "bar",
				},
			},
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(output.Body.(string), "multipart/form-data"))
	})

	t.Run("context timeout", func(t *testing.T) {
		mr := makeRequest{
			Method:   "GET",
			URL:      fmt.Sprintf("http://localhost:%d/sleep1", port),
			BodyType: "empty",
		}
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, err := mr.run(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("response normal body", func(t *testing.T) {
		mr := makeRequest{
			Method:   "GET",
			URL:      fmt.Sprintf("http://localhost:%d/sizedBody", port),
			BodyType: "empty",
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.IsType(t, map[string]any{}, output.Body)
		assert.Equal(t, map[string]any{
			"data": "aaaaaaaaaa",
		}, output.Body)
	})

	t.Run("response body exceeded", func(t *testing.T) {
		mr := makeRequest{
			Method:   "GET",
			URL:      fmt.Sprintf("http://localhost:%d/sizedBody?size=%d", port, 1<<22),
			BodyType: "empty",
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		// body exceeded, so can't unmarshal to map
		assert.IsType(t, "string", output.Body)
	})

	t.Run("empty body", func(t *testing.T) {
		mr := makeRequest{
			Method:   "GET",
			URL:      fmt.Sprintf("http://localhost:%d/empty", port),
			BodyType: "empty",
		}
		output, err := mr.run(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 200, output.StatusCode)
		assert.Equal(t, "", output.Body)
	})
}

func buildQuery(req *http.Request) map[string]string {
	query := make(map[string]string, len(req.URL.Query()))
	for k := range req.URL.Query() {
		query[k] = req.URL.Query().Get(k)
	}
	return query
}
