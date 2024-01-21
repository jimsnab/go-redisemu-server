package main

import (
	"context"
	"testing"
	"time"

	"github.com/jimsnab/go-lane"
	"github.com/jimsnab/go-redisemu"
	"github.com/redis/go-redis/v9"
)

type (
	redisTestServer struct {
		tl  lane.TestingLane
		srv *redisemu.RedisEmu
	}
)

func testServer(t *testing.T) *redisTestServer {
	tl := lane.NewTestingLane(context.Background())
	tl.AddTee(lane.NewLogLane(tl))

	redisServer, err := redisemu.NewEmulator(
		tl,
		7379,
		"localhost",
		"",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	redisServer.Start()

	t.Cleanup(func() {
		redisServer.RequestTermination()
		redisServer.WaitForTermination()
	})

	return &redisTestServer{
		tl:  tl,
		srv: redisServer,
	}
}

func TestPipelined(t *testing.T) {
	ts := testServer(t)

	var incr *redis.IntCmd

	opt := redis.Options{
		Addr: "localhost:7379",
	}
	rdcli := redis.NewClient(&opt)

	_, err := rdcli.Pipelined(ts.tl, func(p redis.Pipeliner) error {
		incr = p.Incr(ts.tl, "test")
		p.PExpire(ts.tl, "test", time.Millisecond*50)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if incr.Val() != 1 {
		t.Error("expected value of 1")
	}

	time.Sleep(time.Millisecond * 100)
	n, err := rdcli.Incr(ts.tl, "test").Result()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("expected value of 1 again")
	}

	rdcli.Del(ts.tl, "test")
}

func TestTxPipelined(t *testing.T) {
	ts := testServer(t)

	var incr *redis.IntCmd

	opt := redis.Options{
		Addr: "localhost:7379",
	}
	rdcli := redis.NewClient(&opt)

	_, err := rdcli.TxPipelined(ts.tl, func(p redis.Pipeliner) error {
		incr = p.Incr(ts.tl, "test")
		p.PExpire(ts.tl, "test", time.Millisecond*50)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if incr.Val() != 1 {
		t.Error("expected value of 1")
	}

	time.Sleep(time.Millisecond * 100)
	n, err := rdcli.Incr(ts.tl, "test").Result()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("expected value of 1 again")
	}

	rdcli.Del(ts.tl, "test")
}
