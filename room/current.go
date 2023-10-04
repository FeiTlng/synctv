package room

import (
	"sync"
	"time"

	json "github.com/json-iterator/go"
)

type Current struct {
	movie  MovieInfo
	status Status
	lock   *sync.RWMutex
}

func newCurrent() *Current {
	return &Current{
		movie:  MovieInfo{},
		status: newStatus(),
		lock:   new(sync.RWMutex),
	}
}

type Status struct {
	Seek           float64 `json:"seek"`
	Rate           float64 `json:"rate"`
	Playing        bool    `json:"playing"`
	lastUpdateTime time.Time
}

func newStatus() Status {
	return Status{
		Seek:           0,
		Rate:           1.0,
		lastUpdateTime: time.Now(),
	}
}

func (c *Current) MarshalJSON() ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.updateSeek()
	return json.Marshal(map[string]interface{}{
		"movie":  c.movie,
		"status": c.status,
	})
}

func (c *Current) Movie() MovieInfo {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.movie
}

func (c *Current) SetMovie(movie MovieInfo) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.movie = movie
	c.status.Seek = 0
}

func (c *Current) Status() Status {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.updateSeek()
	return c.status
}

func (c *Current) updateSeek() {
	if c.status.Playing {
		c.status.Seek += time.Since(c.status.lastUpdateTime).Seconds() * c.status.Rate
	}
	c.status.lastUpdateTime = time.Now()
}

func (c *Current) SetStatus(playing bool, seek, rate, timeDiff float64) Status {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.status.Playing = playing
	c.status.Rate = rate
	if playing {
		c.status.Seek = seek + (timeDiff * rate)
	} else {
		c.status.Seek = seek
	}
	c.status.lastUpdateTime = time.Now()
	return c.status
}

func (c *Current) SetSeekRate(seek, rate, timeDiff float64) Status {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.status.Playing {
		c.status.Seek = seek + (timeDiff * rate)
	} else {
		c.status.Seek = seek
	}
	c.status.Rate = rate
	c.status.lastUpdateTime = time.Now()
	return c.status
}