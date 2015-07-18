package main

//go:generate go-bindata -o=./ui.go -nomemcopy=true -pkg main -prefix=../ui/public $GOBINDATA_OPTS ../ui/public/...

import (
	"encoding/gob"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"

	"github.com/gin-gonic/gin"
	"github.com/zserge/incr"
)

var (
	mutex sync.RWMutex
	db    = map[string]*incr.Event{}
)

// Load events into data store and sync it back periodically
func persistEvents() {
	if f, err := os.Open("events.gob"); err != nil {
		log.Println(err)
	} else {
		if err := gob.NewDecoder(f).Decode(&db); err != nil {
			log.Println(err)
		}
		f.Close()
	}

	go func() {
		for {
			time.Sleep(time.Second)
			if f, err := os.Create("events.gob.tmp"); err != nil {
				log.Println(err)
			} else {
				mutex.Lock()
				if err := gob.NewEncoder(f).Encode(&db); err != nil {
					log.Println(err)
				} else {
					// Os rename
				}
				mutex.Unlock()
				f.Close()
				if err := os.Rename("events.gob.tmp", "events.gob"); err != nil {
					log.Println(err)
				}
			}
		}
	}()
}

func corsHandler(c *gin.Context) {
	c.Writer.Header().Add("Access-Control-Allow-Origin",
		c.Request.Header.Get("Origin"))
	c.Writer.Header().Add("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Add("Access-Control-Allow-Headers",
		c.Request.Header.Get("Access-Control-Request-Headers"))
	c.Writer.Header().Add("Access-Control-Allow-Methods",
		c.Request.Header.Get("Access-Control-Request-Method"))
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(200)
	} else {
		c.Next()
	}
}

func staticHandler(c *gin.Context) {
	path := c.Request.URL.Path[1:]
	if path == "" {
		path = "index.html"
	}
	if buf, err := Asset(path); err == nil {
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		c.Data(200, mimeType, buf)
	} else {
		c.Next()
	}
}

// curl http://www.google-analytics.com/__utm.gif | xxd -i
var minimalGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0xff,
	0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
}

func main() {
	httpPort := os.Getenv("PORT")
	if len(httpPort) == 0 {
		httpPort = "8080"
	}
	r := gin.Default()

	// Limit all requests to 10 per second
	//limitHandler := tollbooth_gin.LimitHandler(tollbooth.NewLimiter(600, 60*time.Second))

	limitHandler := func(c *gin.Context) {
		c.Next()
	}

	persistEvents()

	r.Use(staticHandler)
	r.Use(corsHandler)

	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()
		latency := end.Sub(start)

		mutex.Lock()
		if _, ok := db["incr:request"]; !ok {
			db["incr:request"] = incr.NewEvent()
		}
		db["incr:request"].Add(incr.Time(time.Now().Unix()), incr.Value(latency), "")
		mutex.Unlock()
	})

	go func() {
		for _ = range time.Tick(time.Second) {
			mutex.Lock()
			if _, ok := db["incr:goalloc"]; !ok {
				db["incr:goalloc"] = incr.NewEvent()
				db["incr:memused"] = incr.NewEvent()
				db["incr:load:1"] = incr.NewEvent()
				db["incr:load:5"] = incr.NewEvent()
				db["incr:load:15"] = incr.NewEvent()
			}
			now := incr.Time(time.Now().Unix())

			var memstats runtime.MemStats
			runtime.ReadMemStats(&memstats)
			db["incr:goalloc"].Add(now, incr.Value(memstats.Alloc), "")

			if vmem, err := mem.VirtualMemory(); err == nil {
				db["incr:memused"].Add(now, incr.Value(vmem.Used), "")
			}

			if load, err := load.LoadAvg(); err == nil {
				db["incr:load:1"].Add(now, incr.Value(load.Load1), "")
				db["incr:load:5"].Add(now, incr.Value(load.Load5), "")
				db["incr:load:15"].Add(now, incr.Value(load.Load15), "")
			}
			mutex.Unlock()
		}
	}()

	r.Any("/:name/:value", limitHandler, func(c *gin.Context) {
		name := c.Params.ByName("name")
		value, _ := strconv.ParseFloat(c.Params.ByName("value"), 64)
		sender := ""

		if c, err := c.Request.Cookie("sender"); err == nil {
			sender = c.Value
		}

		mutex.Lock()
		defer mutex.Unlock()

		if _, ok := db[name]; !ok {
			// TODO: Limit requests to 10 per day
			db[name] = incr.NewEvent()
		}

		if date := c.Request.Header.Get("Date"); date != "" {
			if d, err := time.Parse(http.TimeFormat, date); err == nil {
				if diff := d.Sub(time.Now()); math.Abs(diff.Hours()) < 24 {
					db[name].Add(incr.Time(d.Unix()), incr.Value(value), sender)
					return
				}
			}
		}
		db[name].Add(incr.Time(time.Now().Unix()), incr.Value(value), sender)
		if c.Request.Method == "GET" {
			c.Data(200, "image/gif", minimalGIF)
		}
	})

	r.GET("/:name", limitHandler, func(c *gin.Context) {
		mutex.RLock()
		defer mutex.RUnlock()
		e, ok := db[c.Params.ByName("name")]
		if !ok {
			e = incr.NewEvent()
		}

		if c.Request.URL.Query().Get("live") != "" {
			c.JSON(200, e.Data(incr.LiveDuration))
		} else if c.Request.URL.Query().Get("hourly") != "" {
			c.JSON(200, e.Data(incr.HourlyDuration))
		} else if c.Request.URL.Query().Get("daily") != "" {
			c.JSON(200, e.Data(incr.DailyDuration))
		} else if c.Request.URL.Query().Get("weekly") != "" {
			c.JSON(200, e.Data(incr.WeeklyDuration))
		} else {
			c.JSON(200, e.Data(0))
		}
	})

	r.Run(":" + httpPort)
}
