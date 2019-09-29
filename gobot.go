package main

import (
	"fmt"
	"sync"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"
	"github.com/kelvins/sunrisesunset"
)


// GetRobot returns configured raspi robot - PIR sensor + relay
func GetRobot(pirPin, relayPin string, delay int, longitude float64, latitude float64) *gobot.Robot {
	r := raspi.NewAdaptor()
	sensor := gpio.NewPIRMotionDriver(r, pirPin)
	relay := gpio.NewRelayDriver(r, relayPin)
	// Switch off the light 
	relay.On()
	mutex := sync.Mutex{}
	cnt := 0
	// Set sun clock 
	t := time.Now()
	_, offset := t.Zone()
	year, month, day := t.Date()
	p := sunrisesunset.Parameters{
		Latitude:  latitude,
		Longitude: longitude,
		UtcOffset: float64(offset) / 3600,
		Date:      time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
	  }
  
	work := func() {

		sensor.On(gpio.MotionDetected, func(data interface{}) {
			fmt.Println(gpio.MotionDetected)

			t = time.Now()
			year, month, day = t.Date()
			today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			if !p.Date.Equal(today) {
				p.Date = today
				_, offset = t.Zone()
				p.UtcOffset = float64(offset) / 3600
			}
			
			sunrise, sunset, err :=  p.GetSunriseSunset()
			if err != nil || ((sunrise.Hour() * 60 + sunrise.Minute()) < (t.Hour() * 60 + t.Minute()) && (sunset.Hour() * 60 + sunset.Minute()) > (t.Hour() * 60 + t.Minute())) {
				return 
			}

			// Protect counter
			mutex.Lock()
			// Light on - low state switch on the relay
			relay.Off()
			cnt++
			timer := time.NewTimer(time.Duration(delay) * time.Second)
			mutex.Unlock()

			go func() {
				<-timer.C
				// Protect counter
				mutex.Lock()
				if cnt > 0 {
					cnt--
					// Light off only for last call
					if cnt == 0 {
						// Light off - high state switch off the relay
						relay.On()
					}
				}
				mutex.Unlock()
			}()
		})
	}

	robot := gobot.NewRobot("pilight",
		[]gobot.Connection{r},
		[]gobot.Device{sensor, relay},
		work,
	)

	return robot
}
