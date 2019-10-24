package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kelvins/sunrisesunset"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/api"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"
)

// GetRobot returns configured raspi robot - PIR sensor + relay + API command
func GetRobot(
	pirPin,
	relayPin string,
	delay int,
	longitude float64,
	latitude float64,
	lightOnState bool,
	port string,
	pirSensorOn bool,
	remoteRelayIP string,
) *gobot.Master {
	r := raspi.NewAdaptor()
	sensor := gpio.NewPIRMotionDriver(r, pirPin)
	relay := gpio.NewRelayDriver(r, relayPin)
	// Switch off the light - lightOnState keeps logic value when light is on - some relays are activated on high or low signal state
	if lightOnState {
		relay.Off()
	} else {
		relay.On()
	}
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
	sunrise, sunset, err := p.GetSunriseSunset()
	log.Println("Sunrise:", sunrise.Format("15:04:05"), sunrise) // Sunrise: 06:11:44
	log.Println("Sunset:", sunset.Format("15:04:05"), sunset)    // Sunset: 18:14:27
	if err != nil {
		return nil
	}

	// Function called by API command and pir sensor, checks wheather there is a dark outside
	lightOn := func(duration int) {

		t = time.Now()
		year, month, day = t.Date()
		today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if !p.Date.Equal(today) {
			p.Date = today
			_, offset = t.Zone()
			p.UtcOffset = float64(offset) / 3600
			sunrise, sunset, err = p.GetSunriseSunset()
		}
		startTime := t.Hour()*60 + t.Minute()
		endTime := startTime + duration/60
		sunriseTime := sunrise.Hour()*60 + sunrise.Minute()
		sunsetTime := sunset.Hour()*60 + sunset.Minute()

		// Light on request out of the darkness window
		if err != nil || ((sunriseTime < startTime) && (sunsetTime > endTime)) {
			return
		}

		// Light Off time adjustment for sunrise - endtime after sunrise
		if endTime > sunriseTime {
			duration = (sunriseTime - startTime) * 60
		}

		// Protect counter
		mutex.Lock()
		// Light on - lightOnState keeps logic value when light is on
		if lightOnState {
			relay.On()
		} else {
			relay.Off()
		}
		cnt++
		timer := time.NewTimer(time.Duration(duration) * time.Second)
		mutex.Unlock()

		// Light off after configured period of time
		go func() {
			<-timer.C
			// Protect counter
			mutex.Lock()
			if cnt > 0 {
				cnt--
				// Light off only for last call
				if cnt == 0 {
					// Light off
					if lightOnState {
						relay.Off()
					} else {
						relay.On()
					}
				}
			}
			mutex.Unlock()
		}()
	}

	work := func() {

		if pirSensorOn {
			sensor.On(gpio.MotionDetected, func(data interface{}) {
				log.Println(gpio.MotionDetected)
				lightOn(delay)
				if remoteRelayIP != "" {
					_, err := http.Get("http://" + remoteRelayIP + ":" + port + "/api/robots/pilight/commands/light_on")
					if err != nil {
						log.Println(err)
					}

				}
			})
		}
	}

	robot := gobot.NewRobot("pilight",
		[]gobot.Connection{r},
		[]gobot.Device{sensor, relay},
		work,
	)

	robot.AddCommand("light_on",
		func(params map[string]interface{}) interface{} {
			duration := delay
			if paramDuration, ok := params["duration"]; ok {
				if val, ok := paramDuration.(float64); ok {
					duration = int(val)
				}
			}
			lightOn(duration)
			return fmt.Sprintf("Light On - %d", duration)
		})

	mbot := gobot.NewMaster()
	mbot.AddRobot(robot)
	server := api.NewAPI(mbot)
	server.Port = port
	server.Start()
	return mbot
}
