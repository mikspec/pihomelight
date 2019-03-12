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

	flag.Parse()

	// Start robot
	robot := GetRobot(PIRPIN, RELAYPIN, *duration)
	robot.Start()
}
