package main

import (
	"fmt"
	"sync"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"
)

// GetRobot returns configured raspi robot - PIR sensor + relay
func GetRobot(pirPin, relayPin string, delay int) *gobot.Robot {
	r := raspi.NewAdaptor()
	sensor := gpio.NewPIRMotionDriver(r, pirPin)
	relay := gpio.NewRelayDriver(r, relayPin)
	mutex := sync.Mutex{}
	cnt := 0

	work := func() {

		sensor.On(gpio.MotionDetected, func(data interface{}) {
			fmt.Println(gpio.MotionDetected)

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
