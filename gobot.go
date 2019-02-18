package main

import (
	"fmt"
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

	work := func() {

		sensor.On(gpio.MotionDetected, func(data interface{}) {
			fmt.Println(gpio.MotionDetected)
			// Light on
			relay.Off()
			timer := time.NewTimer(time.Duration(delay) * time.Second)
			go func() {
				<-timer.C
				// Light off
				relay.On()
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
