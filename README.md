# Gin Cache Middleware
[![Build Status](https://travis-ci.org/francoispqt/gincache.svg?branch=master)](https://travis-ci.org/francoispqt/gincache)
[![codecov](https://codecov.io/gh/francoispqt/gincache/branch/master/graph/badge.svg)](https://codecov.io/gh/francoispqt/gincache)


## Get started
Import the package

```go
import "github.com/francoispqt/gincache"
```

Then use it

```go
r := gin.Default()

// creating a global middleware
r.Use(gincache.NewMiddleware(&Options{
    TTL: 3600,
    KeyFunc: func(c *gin.Context) (string, error){
        return "CACHEKEY", nil
    },
}))

// creating a route based middleware
r.GET(
    "/",
    gincache.NewMiddleware(&Options{
        TTL: 3600,
        Key: "GET/",
    }),
    func(c *fin.Context) {

    }
)

r.Run(":8080")
```