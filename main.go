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

type Page struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
    Url   string `json:"url"`
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

    url := "https://github.com/hippopuppet/LineBot-Bot-Boss/blob/master/BossRefreshInfo.json"

	spaceClient := http.Client{
        Timeout: time.Second * 2, // Maximum of 2 secs
    }

    res, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        log.Print(err)
        os.Exit(1)
    }

	req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := spaceClient.Do(req)
    if getErr != nil {
        log.Fatal(getErr)
    }
	body, readErr := ioutil.ReadAll(res.Body)
    if readErr != nil {
        log.Fatal(readErr)
    }


    var c []Page
    jsonErr := json.Unmarshal(body, &c)
	if jsonErr != nil {
        log.Fatal(jsonErr)
    }

    return c
}

var bot *linebot.Client
var userID string
var groupID string

func main() {
	var err error
	bot, err = linebot.New(os.Getenv("ChannelSecret"), os.Getenv("ChannelAccessToken"))
	log.Println("Bot:", bot, " err:", err)
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
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text+"--"+ strconv.Itoa( time.Now().In(local).Hour() )+"-"+strconv.Itoa( time.Now().In(local).Minute() )+"-"+strconv.Itoa( time.Now().In(local).Second() ) )).Do(); err != nil {
					log.Print(err)
				}
			}
			pages := getPages()
			for _, p := range pages {
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("JASON-"p.toString() )).Do(); err != nil {
					log.Print(err)
				}
			}
		}

		if event.Type == linebot.EventTypeJoin {
			//userID := event.Source.UserID
			groupID := event.Source.GroupID
			if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("GroupID"+groupID)).Do(); err != nil {
					log.Print(err)
			}
		}

		if event.Type == linebot.EventTypeFollow {
			userID := event.Source.UserID
			//groupID := event.Source.GroupID
			if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("UserID:"+userID)).Do(); err != nil {
					log.Print(err)
			}
		}
	}

	
}
