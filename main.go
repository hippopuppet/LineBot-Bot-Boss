// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"strconv"
	"encoding/json"
	"io/ioutil"

	"github.com/line/line-bot-sdk-go/linebot"
)

type Time time.Time

const (
    timeFormart = "2006-01-02 15:04:05"
)

func (t *Time) UnmarshalJSON(data []byte) (err error) {
    now, err := time.ParseInLocation(`"`+timeFormart+`"`, string(data), time.Local)
    *t = Time(now)
    return
}

func (t Time) MarshalJSON() ([]byte, error) {
    b := make([]byte, 0, len(timeFormart)+2)
    b = append(b, '"')
    b = time.Time(t).AppendFormat(b, timeFormart)
    b = append(b, '"')
    return b, nil
}

func (t Time) String() string {
    return time.Time(t).Format(timeFormart)
}

type Page struct {
    KingOfName  string `json:"KingOfName"`
	RefreshTick int `json:"RefreshTick"`
	Die Time `json:"Die"`
    Resurrection Time `json:"Resurrection"`
}

func (p Page) toString() string {
    return toJson(p)
}

func toJson(p interface{}) string {
    bytes, err := json.Marshal(p)
    if err != nil {
        log.Print(err)
        os.Exit(1)
    }

    return string(bytes)
}

func getPages() []Page {
   raw, err := ioutil.ReadFile("./BossRefreshInfo.json")
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }

    var c []Page
    json.Unmarshal(raw, &c)
    return c
}

var bot *linebot.Client
var userID string
var groupID string
var doneChan = make(chan bool)
var checkBossTimer time.Ticker

func main() {
	checkBossTimer := time.NewTicker(time.Second*10).C
	
	var err error
	bot, err = linebot.New(os.Getenv("ChannelSecret"), os.Getenv("ChannelAccessToken"))
	log.Println("Bot:", bot, " err:", err)

	go func() {
		for {
        select {
        case <- checkBossTimer:
            log.Println("checkBossTimer expired")
			pages := getPages()
			for _, p := range pages {
				if _, err := bot.PushMessage(userID, linebot.NewTextMessage("JASON-"+p.toString())).Do(); err != nil {
					log.Print(err)
				}
			}
        case <- doneChan:
            log.Println("Done")
            return
			}
		}
	}()

	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		
		if event.Type == linebot.EventTypeMessage {
			var local *time.Location
			local, ok := time.LoadLocation("Asia/Taipei")
			log.Print(ok)
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if message.Text == "DONE" {
					doneChan <- true
				}
				if message.Text == "STOP" {
					checkBossTimer.Stop()
				}
				if message.Text == "START" {
					checkBossTimer = time.NewTicker(time.Second*10).C
				}
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text+"--"+ strconv.Itoa( time.Now().In(local).Hour() )+"-"+strconv.Itoa( time.Now().In(local).Minute() )+"-"+strconv.Itoa( time.Now().In(local).Second() ) )).Do(); err != nil {
					log.Print(err)
				}
			
				
			}
			
		}

		if event.Type == linebot.EventTypeJoin {
			//userID := event.Source.UserID
			groupID = event.Source.GroupID
			if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("GroupID"+groupID)).Do(); err != nil {
					log.Print(err)
			}
		}

		if event.Type == linebot.EventTypeFollow {
			userID = event.Source.UserID
			//groupID := event.Source.GroupID
			if _, err := bot.PushMessage(userID, linebot.NewTextMessage("UserID:"+userID)).Do(); err != nil {
				log.Print(err)
			}
		}
	}

	
}
