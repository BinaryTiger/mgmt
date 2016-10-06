// Mgmt
// Copyright (C) 2013-2016+ James Shubin and the project contributors
// Written by James Shubin <james@shubin.ca> and the project contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/gob"
	"log"
)

func init() {
	gob.Register(&CronRes{})
}

// CronRes is a timer resource for time based events based on systemd Cron.
type CronRes struct {
	BaseRes  `yaml:",inline"`
	Interval int `yaml:"interval"` // Interval : Interval between runs
}

// CronUUID is the UUID struct for CronRes.
type CronUUID struct {
	BaseUUID
	name string
}

// NewCronRes is a constructor for this resource. It also calls Init() for you.
func NewCronRes(name string, interval int) *CronRes {
	obj := &CronRes{
		BaseRes: BaseRes{
			Name: name,
		},
		Interval: interval,
	}
	obj.Init()
	return obj
}

// Init runs some startup code for this resource.
func (obj *CronRes) Init() {
	obj.BaseRes.kind = "Cron"
	obj.BaseRes.Init() // call base init, b/c we're overrriding
}

// Validate the params that are passed to CronRes.
// Currently we are getting only an interval in seconds
// which gets validated by go compiler
func (obj *CronRes) Validate() bool {
	return true
}

// Watch is the primary listener for this resource and it outputs events.
func (obj *CronRes) Watch(processChan chan Event) error {
	if obj.IsWatching() {
		return nil
	}
	obj.SetWatching(true)
	defer obj.SetWatching(false)
	cuuid := obj.converger.Register()
	defer cuuid.Unregister()

	var startup bool
	Startup := func(block bool) <-chan time.Time {
		if block {
			return nil // blocks forever
			//return make(chan time.Time) // blocks forever
		}
		return time.After(time.Duration(500) * time.Millisecond) // 1/2 the resolution of converged timeout
	}

	// Create a time.Ticker for the given interval
	ticker := time.NewTicker(time.Duration(obj.Interval) * time.Second)
	defer ticker.Stop()

	var send = false

	for {
		obj.SetState(resStateWatching)
		select {
		case <-ticker.C: // received the timer event
			send = true
			log.Printf("%v[%v]: received tick", obj.Kind(), obj.GetName())
		case event := <-obj.events:
			cuuid.SetConverged(false)
			if exit, _ := obj.ReadEvent(&event); exit {
				return nil
			}
		case <-cuuid.ConvergedCron():
			cuuid.SetConverged(true)
			continue

		case <-Startup(startup):
			cuuid.SetConverged(false)
			send = true
		}
		if send {
			startup = true // startup finished
			send = false
			obj.isStateOK = false
			if exit, err := obj.DoSend(processChan, "timer ticked"); exit || err != nil {
				return err // we exit or bubble up a NACK...
			}
		}
	}
}

// GetUUIDs includes all params to make a unique identification of this object.
// Most resources only return one, although some resources can return multiple.
func (obj *CronRes) GetUUIDs() []ResUUID {
	x := &CronUUID{
		BaseUUID: BaseUUID{
			name: obj.GetName(),
			kind: obj.Kind(),
		},
		name: obj.Name,
	}
	return []ResUUID{x}
}

// The AutoEdges method returns the AutoEdges. In this case none are used.
func (obj *CronRes) AutoEdges() AutoEdge {
	return nil
}

// Compare two resources and return if they are equivalent.
func (obj *CronRes) Compare(res Res) bool {
	switch res.(type) {
	case *CronRes:
		res := res.(*CronRes)
		if !obj.BaseRes.Compare(res) {
			return false
		}
		if obj.Name != res.Name {
			return false
		}
		if obj.Interval != res.Interval {
			return false
		}
	default:
		return false
	}
	return true
}

// CheckApply method for Cron resource. Does nothing, returns happy!
func (obj *CronRes) CheckApply(apply bool) (bool, error) {
	log.Printf("%v[%v]: CheckApply(%t)", obj.Kind(), obj.GetName(), apply)
	return true, nil // state is always okay
}
