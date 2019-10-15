package main

import "flag"

// PIRPIN - gpio pin for PIR sensor
const PIRPIN = "13"

// RELAYPIN - gpio pin for relay
const RELAYPIN = "3"

func main() {

	duration := flag.Int(
		"duration",
		120,
		"Light on duration",
	)

	longitude := flag.Float64(
		"longitude",
		17.0,
		"Longitude of property",
	)

	latitude := flag.Float64(
		"latitude",
		51.0,
		"Latitude of property",
	)

	lightOnState := flag.Bool(
		"lightOnState",
		false,
		"Light on state",
	)

	port := flag.String(
		"port",
		"3000",
		"API server port",
	)

	pirSensorOn := flag.Bool(
		"pirSensorOn",
		true,
		"Pir sensor on",
	)

	remoteRelayIP := flag.String(
		"remoteRelay",
		"",
		"Remote relay IP",
	)

	flag.Parse()

	// Start robot
	robot := GetRobot(PIRPIN, RELAYPIN, *duration, *longitude, *latitude, *lightOnState, *port, *pirSensorOn, *remoteRelayIP)
	robot.Start()
}
