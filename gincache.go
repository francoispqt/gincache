package gincache

import (
	"bytes"

	adapters "github.com/francoispqt/gincache/adapters"
	"github.com/gin-gonic/gin"
)

const (
	// DefaultCachedResponseStatusCode is the default status code for the cached response
	DefaultCachedResponseStatusCode = 200
	// DefaultCachedResponseContentType is the default Content-Type header for the cached response
	DefaultCachedResponseContentType = "application/json"
)

// DefaultAdapter is the default adapter used if no StorageAdapter is provided in the Options
var DefaultAdapter StorageAdapter

// KeyFunc is a custom type for the function generating cache keys
type KeyFunc func(*gin.Context) (string, error)

// StorageAdapter is the interface representing a storage adapter for the cache
// Add your own storage adapters by implementing this interface
type StorageAdapter interface {
	Get(string) (bool, string, error)
	Set(string, string, int) error
	Clear(string) error
}

// Options is the structure passed to NewMiddleware factory function
type Options struct {
	TTL                 int
	Key                 string
	KeyFunc             KeyFunc
	Adapter             StorageAdapter
	DisableSet          bool
	ResponseStatusCode  int
	ResponseContentType string
	KeyAsETag           bool
	Headers             map[string]string
}

type bodyWriter struct {
	gin.ResponseWriter
	body         *bytes.Buffer
	CacheOptions *Options
	context      *gin.Context
	CacheKey     string
}

func isErrorResponse(c *gin.Context, options *Options) bool {
	statusCode := c.Writer.Status()
	return statusCode > 399
}

func (w bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	c := w.context
	options := w.CacheOptions
	// check if response is not error code
	// if context is not aborted and if DisableSet is not done
	// and cache
	if !isErrorResponse(c, options) && !options.DisableSet && !c.IsAborted() {
		// check if we know the content type
		if options.ResponseContentType == "" {
			options.ResponseContentType = w.ResponseWriter.Header().Get("Content-Type")
		}

		err := options.Adapter.Set(w.CacheKey, w.body.String(), options.TTL)
		if err != nil {
			c.Set("CacheSetError", err)
			c.Error(err)
		}
	}
	return w.ResponseWriter.Write(b)
}

// NewMiddleware is the factory function returning a gin.HandlerFunc closure as a middleware
func NewMiddleware(options *Options) gin.HandlerFunc {
	// check options
	// if no adapter assign default adapter (memory)
	if options.Adapter == nil {
		if DefaultAdapter == nil {
			DefaultAdapter = &adapters.MemoryAdapter{}
		}
		options.Adapter = DefaultAdapter
	}
	key := options.Key
	return func(c *gin.Context) {
		c.Set("CacheOptions", options)
		// if TTL is 0 do not cache
		if options.TTL == 0 {
			c.Next()
			return
		}

		// getKey
		// if it errors, keep the normal flow without cache
		var err error
		if options.KeyFunc != nil {
			key, err = options.KeyFunc(c)
			// if error while retrieving key
			if err != nil {
				c.Abort()
				return
			}
		}
		c.Set("CacheKey", key)

		// check if key exists in adapter
		exists, result, err := options.Adapter.Get(key)
		// if result found, return it using optional ResponseCode from options
		// and Content Type, if no Content-type option, use from response
		// here we use a boolean to avoid checking empty string on result
		if exists && err == nil {
			// send response from cache
			// get ResponseCode according to options
			statusCode := options.ResponseStatusCode
			if statusCode == 0 {
				statusCode = DefaultCachedResponseStatusCode
			}
			// get ContentType
			contentType := options.ResponseContentType
			if contentType == "" {
				contentType = DefaultCachedResponseContentType
			}
			headers := c.Writer.Header()
			if len(options.Headers) > 0 {
				for k, header := range options.Headers {
					headers.Add(k, header)
				}
			}
			if options.KeyAsETag {
				headers.Add("ETag", key)
			}
			headers.Add("Content-Type", contentType)
			c.Writer.WriteHeader(statusCode)
			c.Writer.Write([]byte(result))
			c.Abort()
			return
		}

		// overwrite the writer to intercept response content
		bWriter := &bodyWriter{
			body:           bytes.NewBufferString(""),
			context:        c,
			ResponseWriter: c.Writer,
			CacheOptions:   options,
			CacheKey:       key,
		}
		c.Writer = bWriter
		c.Next()
	}
}
