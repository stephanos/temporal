// Command ringpop is the fn-224 multi-node scenario guest: each node joins a
// ringpop ring over the modeled network, must have dropped the peers it is
// partitioned from, and must see a stable full ring after the heal and again
// at its deadline.
package main

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/temporalio/ringpop-go"
	"github.com/temporalio/ringpop-go/discovery/statichosts"
	"github.com/temporalio/ringpop-go/swim"
	"github.com/temporalio/tchannel-go"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func seconds(value string) time.Duration {
	parsed, err := strconv.Atoi(value)
	check(err)
	return time.Duration(parsed) * time.Second
}

// Usage: ringpop <self host:port> <comma-separated ring host:ports>
// <partitioned-view second> <heal second> <deadline second>
// <comma-separated peers unreachable at the view, or ->
func main() {
	if len(os.Args) != 7 {
		panic("usage: ringpop self hosts view heal deadline partitioned")
	}
	start := time.Now()
	self, hosts := os.Args[1], strings.Split(os.Args[2], ",")
	view, heal, deadline := seconds(os.Args[3]), seconds(os.Args[4]), seconds(os.Args[5])
	var partitioned []string
	if os.Args[6] != "-" {
		partitioned = strings.Split(os.Args[6], ",")
	}
	slices.Sort(hosts)
	at := func(offset time.Duration) { time.Sleep(time.Until(start.Add(offset))) }

	listener, err := net.Listen("tcp", self)
	check(err)
	channel, err := tchannel.NewChannel("gomad-ringpop", &tchannel.ChannelOptions{})
	check(err)
	check(channel.Serve(listener))
	ring, err := ringpop.New("gomad-ringpop", ringpop.Channel(channel), ringpop.Address(self))
	check(err)
	for {
		_, err = ring.Bootstrap(&swim.BootstrapOptions{JoinSize: 1, MaxJoinDuration: 10 * time.Second, DiscoverProvider: statichosts.New(hosts...)})
		if err == nil {
			break
		}
		if time.Since(start) >= view {
			panic(fmt.Sprintf("bootstrap %s: %v", self, err))
		}
		time.Sleep(time.Second)
	}
	members := func() []string {
		reachable, err := ring.GetReachableMembers()
		check(err)
		slices.Sort(reachable)
		return reachable
	}
	fmt.Printf("bootstrapped %s at %s\n", self, time.Since(start).Truncate(time.Millisecond))

	at(view)
	current := members()
	fmt.Printf("partitioned view %s %v\n", self, current)
	for _, peer := range partitioned {
		if slices.Contains(current, peer) {
			panic(fmt.Sprintf("%s still reaches partitioned peer %s: %v", self, peer, current))
		}
	}

	at(heal)
	stable := 0
	for time.Since(start) < deadline {
		if slices.Equal(members(), hosts) {
			stable++
		} else {
			stable = 0
		}
		if stable == 3 {
			checksum, err := ring.Checksum()
			check(err)
			fmt.Printf("converged %s after heal %s checksum %d\n", self, (time.Since(start) - heal).Truncate(time.Second), checksum)
			at(deadline)
			if final := members(); !slices.Equal(final, hosts) {
				panic(fmt.Sprintf("membership of %s lost the full ring before its deadline: %v", self, final))
			}
			return
		}
		time.Sleep(time.Second)
	}
	panic(fmt.Sprintf("membership of %s did not converge after heal: %v", self, members()))
}
