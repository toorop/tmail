package main

import (
	"time"
)

// deliver delivers deliveries (Oh wait !! that's amazing)
func deliver(delivery *delivery, cCountDeliveries *chan int) {
	defer decrementsDeliveryCounter(cCountDeliveries)

}

// Helpers

// decrementsDeliveryCounter
func decrementsDeliveryCounter(cCountDeliveries *chan int) {
	*cCountDeliveries <- -1
}
