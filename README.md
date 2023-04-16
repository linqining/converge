# converge

Package converge not just like singleflight providing a duplicate batch function call suppression mechanism, but also 
merge multiple single resource request into a batch function call. 

## Installation

    $go get github.com/linqining/converge

## Example
```go
package main

import (
	"errors"
	"github.com/linqining/converge"
	"log"
	"math/rand"
	"sync"
	"time"
)

type AssetAmount int64

func batchQuery(userIDs []int64) (map[int64]AssetAmount, error) {
	ret := make(map[int64]AssetAmount)
	for _, id := range userIDs {
		ret[id] = AssetAmount(1000 + rand.Int63n(1000))
	}
	return ret, nil
}

var userConverge *converge.Converge[int64, AssetAmount]

func init() {
	var err error
	userConverge, err = converge.New[int64, AssetAmount](batchQuery, converge.NewConfig(10, time.Millisecond))
	if err != nil {
		panic(err)
	}
}

func GetUserMoney(uid int64) (AssetAmount, error) {
	ret, err := userConverge.Do([]int64{uid})
	if err != nil {
		return 0, err
	}
	result := ret[uid]
	if !result.Exist {
		return 0, errors.New("can not find user asset")
	}
	return result.Val, nil
}

func BatchGetUserMoney(uids []int64) (map[int64]AssetAmount, error) {
	ret, err := userConverge.Do(uids)
	if err != nil {
		return nil, err
	}
	userAssetMap := make(map[int64]AssetAmount)
	for k, v := range ret {
		if v.Exist {
			userAssetMap[k] = v.Val
		} else {
			userAssetMap[k] = 0
		}
	}
	return userAssetMap, nil
}

func main() {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		assAmount, err := GetUserMoney(1001)
		if err != nil {
			log.Println(err)
		}
		log.Println("1001 GetUserMoney.assetAmount", assAmount)
	}()

	go func() {
		defer wg.Done()
		amountMap, err := BatchGetUserMoney([]int64{1001, 1002, 1003})
		if err != nil {
			log.Println(err)
		}
		log.Println("1001 BatchGetUserMoney.assetAmount", amountMap[1001])
		log.Println("1002 BatchGetUserMoney.assetAmount", amountMap[1002])
		log.Println("1003 BatchGetUserMoney.assetAmount", amountMap[1003])
	}()
	wg.Wait()
}
```
## Config
converge.NewConfig has two parameters:

The first one represent maxInFlight, total BatchFunction call will limit to at most maxInFlight call

The second is waitDuration, there is a gap between and putting the elements to the pending call list and taking the 
elements from the pending call list, it's useful when a call takes long time,such as 1 second,it should less than
the time spend each call. You can see the TestNewConverge testcase.
