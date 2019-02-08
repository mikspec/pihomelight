package main

// PIRPIN - gpio pin for PIR sensor
const PIRPIN = "11"

// RELAYPIN - gpio pin for relay
const RELAYPIN = "3"

func main() {
	// Start robot
	robot := GetRobot(PIRPIN, RELAYPIN, 60)
	robot.Start()
}
