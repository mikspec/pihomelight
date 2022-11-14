package main

import (
	"bytes"
	"encoding/json"
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
	halloweenDivider int,
	halloweenLoop int,
) *gobot.Master {
	r := raspi.NewAdaptor()
	sensor := gpio.NewPIRMotionDriver(r, pirPin)
	relay := gpio.NewRelayDriver(r, relayPin)
	relay.Inverted = !lightOnState
	relay.Off()
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

	// Light on and off function with mutex protection
	lightSwitch := func(lightOnDuration time.Duration) {
		// Protect counter
		mutex.Lock()
		relay.On()
		cnt++
		timer := time.NewTimer(lightOnDuration)
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
					relay.Off()
				}
			}
			mutex.Unlock()
		}()
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

		// Light On request out of the darkness window
		if err != nil || ((sunriseTime <= startTime) && (sunsetTime >= endTime)) {
			return
		}

		// Light Off time adjustment for sunrise - endtime after sunrise
		if (startTime < sunriseTime) && (sunriseTime < endTime) && (startTime != endTime) {
			duration = (sunriseTime - startTime) * 60
		}

		// Light On time adjustment for sunset - start time before sunset
		if (startTime < sunsetTime) && (sunsetTime < endTime) && (startTime != endTime) {
			sunsetTimer := time.NewTimer(time.Duration(sunsetTime-startTime) * time.Minute)

			go func() {
				<-sunsetTimer.C
				lightSwitch(time.Duration(endTime-sunsetTime) * time.Minute)
			}()
			return
		}

		lightSwitch(time.Duration(duration) * time.Second)
	}

	halloweenLight := func(divider int, loop int) {
		go func() {
			if remoteRelayIP != "" {
				log.Println("Halloween child start")
				postBody, _ := json.Marshal(map[string]int{
					"divider": divider,
					"loop":    loop,
				})
				reqBody := bytes.NewBuffer(postBody)
				resp, err := http.Post("http://"+remoteRelayIP+":"+port+"/api/robots/pilight/commands/halloween", "application/json", reqBody)
				if err != nil {
					log.Println(err)
				} else if resp != nil && resp.Body != nil {
					defer resp.Body.Close()
				}
				log.Println("Halloween child end")
			}
		}()

		log.Printf("Halloween main start: %d, %d\n", divider, loop)
		d := time.Duration(1) * time.Second / time.Duration(divider)
		for i := 0; i < loop; i++ {
			relay.Toggle()
			time.Sleep(d)
			relay.Toggle()
			time.Sleep(d)
		}
		log.Println("Halloween main end")
	}

	work := func() {

		if pirSensorOn {
			sensor.On(gpio.MotionDetected, func(data interface{}) {
				log.Println(gpio.MotionDetected)
				lightOn(delay)
				if remoteRelayIP != "" {
					resp, err := http.Get("http://" + remoteRelayIP + ":" + port + "/api/robots/pilight/commands/light_on")
					if err != nil {
						log.Println(err)
					} else if resp != nil && resp.Body != nil {
						defer resp.Body.Close()
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

	robot.AddCommand("halloween",
		func(params map[string]interface{}) interface{} {
			divider := halloweenDivider
			if paramDivider, ok := params["divider"]; ok {
				if val, ok := paramDivider.(float64); ok {
					divider = int(val)
				}
			}
			loop := halloweenLoop
			if paramLoop, ok := params["loop"]; ok {
				if val, ok := paramLoop.(float64); ok {
					loop = int(val)
				}
			}
			halloweenLight(divider, loop)
			return "Halloween"
		})

	mbot := gobot.NewMaster()
	mbot.AddRobot(robot)
	server := api.NewAPI(mbot)
	server.Port = port
	server.Start()
	return mbot
}
