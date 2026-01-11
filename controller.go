package openfand

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mdouchement/logger"
	"github.com/mdouchement/openfand/openfan"
)

type Controller struct {
	fan      OpenFan
	sensor   Sensor
	shaper   Shaper
	events   chan event
	listener net.Listener
	ticker   *time.Ticker
	fans     map[openfan.Fan]Fan
	active   map[openfan.Fan]Evaluation
	pending  map[openfan.Fan]Evaluation
}

func New(cfg Config, fan OpenFan, sensor Sensor, shaper Shaper, polling time.Duration) (*Controller, error) {
	c := &Controller{
		fan:     fan,
		sensor:  sensor,
		shaper:  shaper,
		events:  make(chan event, 10),
		ticker:  time.NewTicker(polling),
		fans:    make(map[openfan.Fan]Fan),
		active:  make(map[openfan.Fan]Evaluation),
		pending: make(map[openfan.Fan]Evaluation),
	}

	for _, fan := range cfg.FanSettings {
		c.fans[fan.ID] = *fan
	}

	err := os.MkdirAll(filepath.Dir(cfg.Socket), 0o755)
	if err != nil {
		return nil, fmt.Errorf("socket: %w", err)
	}

	if _, err := os.Stat(cfg.Socket); err == nil {
		fmt.Printf("Removing existing %s\n", cfg.Socket)
		os.Remove(cfg.Socket)
	}
	c.listener, err = net.Listen("unix", cfg.Socket)
	if err != nil {
		return nil, fmt.Errorf("socket: %w", err)
	}

	return c, nil
}

func (c *Controller) Launch(ctx context.Context) {
	log := logger.LogWith(ctx)

	go c.eventLoop(ctx)

	http.HandleFunc("/monitor", c.monitor(log))
	go func() {
		for {
			log.Info("Staring HTTP server on", c.listener.Addr().String())
			err := http.Serve(c.listener, nil)
			if err != nil {
				log.WithError(err).Error("Could not serve HTTP")
			}
			time.Sleep(2 * time.Second)
		}
	}()

	evalCh := make(chan map[openfan.Fan]Evaluation, 1)
	refreshCh := make(chan refresh, 1)
	go c.gatherTemperatures(log, evalCh)
	go c.eval(log, evalCh, refreshCh)

	refreshCh <- refresh{interval: 10 * time.Second} // At least RPMs are refreshed every this interval

	go func() {
		for {
			select {
			case e := <-refreshCh:
				rpms, err := c.fan.RPMs()
				if err != nil {
					log.WithError(err).Error("Could not read RPMs")
					continue
				}

				c.events <- event{name: eventUpdateRPMs, rpms: rpms}

				// Prepare next iteration.
				if e.until == 0 || e.current < e.until {
					e.current++
					time.AfterFunc(e.interval, func() {
						refreshCh <- e
					})
				}

			case <-ctx.Done():
				c.ticker.Stop()
				close(evalCh)
				close(refreshCh)
				if err := c.listener.Close(); err != nil {
					log.WithError(err).Error("Could not close socket listener")
				}
				if err := os.Remove(c.listener.Addr().String()); err != nil && err != os.ErrNotExist {
					// listener.Close() should close the socket but ceinture et bretelles!
					log.WithError(err).Errorf("Could not remove socket %s", c.listener.Addr().String())
				}

				close(c.events)
				return
			}
		}
	}()
}

func (c *Controller) eventLoop(ctx context.Context) {
	log := logger.LogWith(ctx)
	watchers := map[int64]chan<- []byte{}

	for e := range c.events {
		switch e.name {
		case eventUpdateEval:
			c.active[e.eval.ID] = e.eval
		case eventUpdateRPMs:
			var change bool

			for fid, rpm := range e.rpms {
				eval := c.active[fid]

				const tolerance = 5
				if eval.RPM != 0 && (rpm < eval.RPM-tolerance || rpm > eval.RPM+tolerance) {
					// Only log if RPMs changed to avoid flooding the logs.
					change = true
				}

				eval.RPM = rpm
				c.active[fid] = eval
			}

			if change {
				var speeds []string
				for _, fid := range slices.Sorted(maps.Keys(e.rpms)) {
					rpm := e.rpms[fid]
					if rpm == 0 {
						continue
					}
					speeds = append(speeds, fmt.Sprintf("fan%d(%s): %d", fid+1, c.fans[fid].Label, rpm))
				}
				log.Info(strings.Join(speeds, " - "))
			}

			c.events <- event{name: eventRefreshWatchers}

		case eventRefreshWatchers:
			payload, err := json.Marshal(slices.Collect(maps.Values(c.active)))
			if err != nil {
				log.WithError(err).Error("Could not serialize metrics") // Should never happen
				continue
			}

			for _, watcher := range watchers {
				watcher <- payload
			}
		case eventWatch:
			watchers[e.monitorID] = e.monitor
			c.events <- event{name: eventRefreshWatchers}
		case eventUnwatch:
			close(watchers[e.monitorID])
			delete(watchers, e.monitorID)
		}
	}
}

func (c *Controller) gatherTemperatures(log logger.Logger, ch chan<- map[openfan.Fan]Evaluation) {
	for range c.ticker.C {
		temps, err := c.sensor.Temperatures()
		if err != nil {
			log.WithError(err).Error("Could not read temperature sensors")
			continue
		}

		ch <- c.shaper.Eval(temps)
	}
}

func (c *Controller) eval(log logger.Logger, ch <-chan map[openfan.Fan]Evaluation, refreshCh chan<- refresh) {
	for evals := range ch {
		var toRefresh bool

		for fid, eval := range evals {
			sa, ok := c.active[fid]
			if ok {
				if eval.PWM == sa.PWM {
					// No change, just reset everything.
					delete(c.pending, fid)
					continue
				}

				// Setup base variables for delay computing.
				diff := eval.PWM - sa.PWM
				d := c.fans[fid].FanSetUp.Duration
				if diff < 0 {
					d = c.fans[fid].FanSetDown.Duration
				}

				// Do we need to await certain time before updating PWM?
				if d > 0 {
					sp, ok := c.pending[fid]
					if !ok {
						// First change, store for later
						c.pending[fid] = eval
						continue
					}

					if eval.EvaluedAt.Sub(sp.EvaluedAt) < d {
						// Still awaiting the sepcified delay, await next iteration.
						continue
					}

					// Delay reached, reset the map and update PWM.
					delete(c.pending, fid)
				}
			}

			toRefresh = true
			c.events <- event{name: eventUpdateEval, eval: eval}

			log.Infof("Set PWM %d for fan%d(%s) on %s of %.0f°C", eval.PWM, eval.ID+1, eval.Label, strconv.Quote(eval.TemperatureName), eval.Temperature)
			_, err := c.fan.SetPWM(fid, eval.PWM)
			if err != nil {
				log.WithError(err).Errorf("Could not set PWN for fan%d", fid)
				continue
			}
		}

		if !toRefresh {
			continue
		}

		refreshCh <- refresh{interval: 500 * time.Millisecond, until: 8} // 8 events over 4s should be enough for Fans to change their speed.
	}
}

func (c *Controller) monitor(log logger.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connected")

		// Set http headers required for SSE.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		disconnected := r.Context().Done()

		id := genID()
		ch := make(chan []byte, 20)
		c.events <- event{name: eventWatch, monitorID: id, monitor: ch}

		rc := http.NewResponseController(w)
		for {
			select {
			case <-disconnected:
				log.Info("Client disconnected")
				c.events <- event{name: eventUnwatch, monitorID: id}
				return
			case payload := <-ch:
				_, err := w.Write(append(payload, '\n', '\n'))
				if err != nil {
					log.WithError(err).Error("Could not write monitor SSE payload")
					return
				}

				err = rc.Flush()
				if err != nil {
					log.WithError(err).Error("Could not flush monitor SSE payload")
					return
				}
			}
		}
	}
}
