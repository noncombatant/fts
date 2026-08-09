// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"time"
)

type Something struct {
	ID   int64
	Data []byte
}

type getSomethingResult struct {
	Something *Something
	Error     error
}

func GetSomething(id int64) (*Something, error) {
	result := make(chan getSomethingResult, 1)
	go func() {
		result <- doGetSomething(id)
	}()
	select {
	case <-time.After(5 * time.Second):
		return nil, errors.New("timed out")
	case result := <-result:
		return result.Something, result.Error
	}
}

func doGetSomething(id int64) getSomethingResult {
	fmt.Println("sleepin")
	time.Sleep(6 * time.Second)
	fmt.Println("slept")
	something := &Something{
		ID:   int64(id),
		Data: []byte("hello, peeps"),
	}
	fmt.Println("returnin")
	return getSomethingResult{
		Something: something,
		Error:     nil,
	}
}

func main() {
	s, e := GetSomething(33)
	if e != nil {
		fmt.Println(e)
	} else {
		fmt.Println(s.ID, string(s.Data), e)
	}
	fmt.Println("waiting...")
	time.Sleep(4 * time.Second)
	fmt.Println("done waiting")
}
