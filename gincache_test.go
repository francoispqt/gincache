package gincache

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/francoispqt/gincache/adapters"
	"github.com/go-siris/siris/core/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createGinRouter() *gin.Engine {
	r := gin.Default()
	return r
}

type MockAdapter struct {
	mock.Mock
	expectedGetReturn     string
	expectedGetReturnBool bool
	expectedSetReturnErr  error
}

func (m *MockAdapter) Get(key string) (bool, string, error) {
	_ = m.Called(key)
	return m.expectedGetReturnBool, m.expectedGetReturn, nil
}

func (m *MockAdapter) Set(key string, value string, TTL int) error {
	_ = m.Called(key, value, TTL)
	return m.expectedSetReturnErr
}

func (m *MockAdapter) Clear(key string) error {
	_ = m.Called(key)
	return nil
}

func TestGinCacheGetCacheExists(t *testing.T) {
	r := createGinRouter()
	cacheKey := "TESTCACHEKEY"
	expectedTestHeader1 := "test"
	expectedTestHeader2 := "test2"

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturn = "{ \"foo\":\"bar\" }"
	mockAdapter.expectedGetReturnBool = true

	// setup expectations
	// we expect get to be called with our key
	// and stub it to return true
	mockAdapter.
		On("Get", cacheKey).
		Return(true, mockAdapter.expectedGetReturn, nil).
		Once()

	mockAdapter.AssertNotCalled(t, "Set")

	r.GET(
		"/",
		NewMiddleware(&Options{
			TTL: 3600,
			KeyFunc: func(c *gin.Context) (string, error) {
				return cacheKey, nil
			},
			Adapter: mockAdapter,
			Headers: map[string]string{
				"test":  "test",
				"test2": "test2",
			},
		}),
		func(c *gin.Context) {
			// adding false assertion because should never be called
			assert.Equal(t, true, false)
			c.JSON(200, map[string]string{
				"not": "the response",
			})
			return
		},
	)

	var req *http.Request

	req, _ = http.NewRequest("GET", "/", nil)

	req.Header.Set("Content-Type", "application/json")

	// Sets the response recorder
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)
	resBody := res.Body.String()

	// assert response body and response status code
	assert.Equal(t, mockAdapter.expectedGetReturn, resBody, "Expect body to equal the returned value from adapter")
	assert.Equal(t, DefaultCachedResponseStatusCode, res.Code, "Expect status code to be equal to default cache response status code")
	// assert headers are set
	testHeader1 := res.Header().Get("test")
	testHeader2 := res.Header().Get("test2")
	assert.Equal(t, expectedTestHeader1, testHeader1, "Header \"test\" should be set in response")
	assert.Equal(t, expectedTestHeader2, testHeader2, "Header \"test2\" should be set in response")

	mockAdapter.AssertExpectations(t)
}

func TestGinCacheSetNewCache(t *testing.T) {

	r := createGinRouter()

	expectedResponse := "{\"foo\":\"bar\"}"
	cacheKey := "TESTCACHEKEY"
	TTL := 3600

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturnBool = false

	// setup expectations
	mockAdapter.
		On("Get", cacheKey).
		Return(false, "", nil).
		Once()

	mockAdapter.
		On("Set", cacheKey, expectedResponse, TTL).
		Return(nil).
		Once()

	r.GET(
		"/",
		NewMiddleware(&Options{
			TTL: 3600,
			KeyFunc: func(c *gin.Context) (string, error) {
				return cacheKey, nil
			},
			Adapter: mockAdapter,
		}),
		func(c *gin.Context) {
			c.JSON(200, map[string]string{
				"foo": "bar",
			})
			return
		},
	)

	var req *http.Request

	req, _ = http.NewRequest("GET", "/", nil)

	req.Header.Set("Content-Type", "application/json")

	// Sets the response recorder
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	resBody := res.Body.String()

	// assert response body and response status code
	assert.Equal(t, expectedResponse, resBody, "Expect body to equal the returned value from adapter")

	mockAdapter.AssertExpectations(t)

}

func TestGinCacheSetNewCacheTTL0(t *testing.T) {

	r := createGinRouter()
	cacheKey := "TESTCACHEKEY"
	TTL := 0

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturnBool = false

	// setup expectations
	// assert Get and Set are not called because TTL is 0
	mockAdapter.AssertNotCalled(t, "Get")
	mockAdapter.AssertNotCalled(t, "Set")

	r.GET(
		"/",
		NewMiddleware(&Options{
			TTL: TTL,
			KeyFunc: func(c *gin.Context) (string, error) {
				return cacheKey, nil
			},
			Adapter: mockAdapter,
		}),
		func(c *gin.Context) {
			c.JSON(200, map[string]string{
				"foo": "bar",
			})
			return
		},
	)

	var req *http.Request

	req, _ = http.NewRequest("GET", "/", nil)

	req.Header.Set("Content-Type", "application/json")

	// Sets the response recorder
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)
	mockAdapter.AssertExpectations(t)
}

func TestGinCacheSetNewCacheDefaultAdapter(t *testing.T) {

	r := createGinRouter()
	cacheKey := "TESTCACHEKEY"
	TTL := 0

	// options
	opts := &Options{
		TTL: TTL,
		KeyFunc: func(c *gin.Context) (string, error) {
			return cacheKey, nil
		},
	}

	r.GET(
		"/",
		NewMiddleware(opts),
		func(c *gin.Context) {
			c.JSON(200, map[string]string{
				"foo": "bar",
			})
			return
		},
	)

	assert.IsType(t, &adapters.MemoryAdapter{}, opts.Adapter, "Adapter should be default adapter")
}

func TestGinCacheAbortContext(t *testing.T) {

	r := createGinRouter()
	cacheKey := "TESTCACHEKEY"
	TTL := 3600

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturnBool = false

	// setup expectations
	// assert Get is called but set is not called because
	// context is aborted
	mockAdapter.
		On("Get", cacheKey).
		Return(true, mockAdapter.expectedGetReturn, nil).
		Once()
	mockAdapter.AssertNotCalled(t, "Set")

	// options
	opts := &Options{
		TTL: TTL,
		KeyFunc: func(c *gin.Context) (string, error) {
			return cacheKey, nil
		},
		Adapter: mockAdapter,
	}

	r.GET(
		"/",
		NewMiddleware(opts),
		func(c *gin.Context) {
			c.Abort()
			return
		},
	)

	var req *http.Request
	req, _ = http.NewRequest("GET", "/", nil)

	// Sets the response recorder
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	mockAdapter.AssertExpectations(t)
}

func TestGinCacheErrorKeyFunc(t *testing.T) {

	r := createGinRouter()
	TTL := 3600

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturnBool = false

	// setup expectations
	// assert Get and Set are not called because
	// KeyFunc returns an error
	mockAdapter.AssertNotCalled(t, "Get")
	mockAdapter.AssertNotCalled(t, "Set")

	// options
	opts := &Options{
		TTL: TTL,
		KeyFunc: func(c *gin.Context) (string, error) {
			return "", errors.New("TestError")
		},
		Adapter: mockAdapter,
	}

	r.GET(
		"/",
		NewMiddleware(opts),
		func(c *gin.Context) {
			c.JSON(200, "OK")
			return
		},
	)

	var req *http.Request
	req, _ = http.NewRequest("GET", "/", nil)

	// Sets the response recorder
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	mockAdapter.AssertExpectations(t)
}

func TestGinCacheKeyAsETag(t *testing.T) {

	r := createGinRouter()
	cacheKey := "TESTCACHEKEY"
	TTL := 3600

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturnBool = true

	// setup expectations
	// assert Get is called but set is not called because
	// context is aborted
	mockAdapter.
		On("Get", cacheKey).
		Return(true, mockAdapter.expectedGetReturn, nil).
		Once()
	mockAdapter.AssertNotCalled(t, "Set")

	// options
	opts := &Options{
		TTL: TTL,
		KeyFunc: func(c *gin.Context) (string, error) {
			return cacheKey, nil
		},
		KeyAsETag: true,
		Adapter:   mockAdapter,
	}

	r.GET(
		"/",
		NewMiddleware(opts),
		func(c *gin.Context) {
			c.JSON(200, "OK")
			return
		},
	)

	var req *http.Request
	req, _ = http.NewRequest("GET", "/", nil)

	// Sets the response recorder
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	mockAdapter.AssertExpectations(t)

	testHeaderETag := res.Header().Get("ETag")
	assert.Equal(t, cacheKey, testHeaderETag, "ETag header should equal cache key")

}

func TestGinCacheSetError(t *testing.T) {

	r := createGinRouter()
	cacheKey := "TESTCACHEKEY"
	TTL := 3600

	// create an instance of our test object
	mockAdapter := new(MockAdapter)
	mockAdapter.expectedGetReturnBool = false
	mockAdapter.expectedSetReturnErr = errors.New("TestError")

	// setup expectations
	// assert Get is called but set is not called because
	// context is aborted
	mockAdapter.
		On("Get", cacheKey).
		Return(mockAdapter.expectedGetReturnBool, mockAdapter.expectedGetReturn, nil).
		Once()
	mockAdapter.
		On("Set", cacheKey, "\"OK\"", 3600).
		Return(mockAdapter.expectedSetReturnErr).
		Once()

	// options
	opts := &Options{
		TTL: TTL,
		KeyFunc: func(c *gin.Context) (string, error) {
			return cacheKey, nil
		},
		Adapter: mockAdapter,
	}

	r.GET(
		"/",
		NewMiddleware(opts),
		func(c *gin.Context) {
			c.JSON(200, "OK")
			return
		},
	)

	var req *http.Request
	req, _ = http.NewRequest("GET", "/", nil)

	// Sets the response recorder
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	mockAdapter.AssertExpectations(t)
}
