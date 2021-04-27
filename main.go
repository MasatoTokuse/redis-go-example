package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/MasatoTokuse/exectime"
	"github.com/go-redis/redis"
	_ "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis"
)

const maxUser int = 3000

func main() {
	execTime := exectime.Measure(func() {
		client := NewRedisClient()
		err := client.Set("countCurrentUser", 0, 0).Err()
		panicIf(err)
		loopN(RedisMutualExclusionExample, 4000)
	})
	fmt.Println(execTime.Seconds())
}

func RedisMutualExclusionExample(userIdInt int) {
	start := time.Now()
	userId := fmt.Sprint(userIdInt)
	client := NewRedisClient()
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)
	mutexname := "my-global-mutex"
	mutex := rs.NewMutex(mutexname)
	if err := mutex.Lock(); err != nil {
		panic(err)
	}

	val, err := client.Get("countCurrentUser").Result()
	panicIf(err)
	countCurrentUser, err := strconv.Atoi(val)
	panicIf(err)

	if countCurrentUser >= maxUser {
		log.Fatalln(fmt.Sprintf("Can't enter user anymore. maxUser=%d, countCurrentUser=%d", maxUser, countCurrentUser))
	}
	err = client.Set(userId, true, 0).Err()
	panicIf(err)
	err = client.Set("countCurrentUser", countCurrentUser+1, 0).Err()
	panicIf(err)
	_, err = client.Get(userId).Result()
	panicIf(err)
	if err == redis.Nil {
		fmt.Println("key does not exist", userId)
	} else {
		panicIf(err)
		// fmt.Println(userId, val)
	}
	if ok, err := mutex.Unlock(); !ok || err != nil {
		panic("unlock failed")
	}
	fmt.Printf("userId=%s, lockTime=%v, endedTime=%v\n", userId, time.Since(start), time.Now().Format("05.999999999"))
}

func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	client.WithContext(context.Background())
	return client
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func loopN(fn func(i int), n int) {
	for i := 0; i < n; i++ {
		go fn(i)
	}
	time.Sleep(20 * time.Second)
}
