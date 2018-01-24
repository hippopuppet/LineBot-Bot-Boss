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
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
	"gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)
type Page struct {
    KingOfName  string `json:"kingofname"`
	RefreshTick string `json:"refreshtick"`
	Die string `json:"die"`
    Resurrection string `json:"resurrection"`
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
        log.Println(err.Error())
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
	var err error
	
	session, err := mgo.Dial("ds111598.mlab.com:11598/heroku_xzzlp7s1")
    if err != nil {
       panic(err)
    }
    defer session.Close()

    // Optional. Switch the session to a monotonic behavior.
    session.SetMode(mgo.Monotonic, true)

    c := session.DB("heroku_xzzlp7s1").C("bossinfo")

	result := Page{}
    err = c.Find(bson.M{"refreshtick": "120"}).One(&result)
    if err != nil {
       log.Fatal(err)
    }
	log.Println("KingOfName: "+ result.KingOfName)
    
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
				if message.Text == "DONE" {
					doneChan <- true
				}
				 if message.Text == "STOP" {
					checkBossTimer.Stop()
				}
				 if message.Text == "START" {
					checkBossTimer := time.NewTicker(time.Second*10).C
					go func() {
						for {
						select {
						case <- checkBossTimer:
							log.Println("checkBossTimer expired")
							NOWTIME := time.Now().In(local).Hour()*60+time.Now().In(local).Minute()+10
							log.Println("NOWTIME-"+strconv.Itoa(NOWTIME))
							pages := getPages()
							for _, p := range pages {
								log.Println("p.Resurrection-"+p.Resurrection)
								p_Resurrection, err := strconv.Atoi(p.Resurrection)
								if err != nil {
									log.Print(err)
								}
								ResurrectionH := p_Resurrection/100
								ResurrectionM := p_Resurrection - (ResurrectionH*100)
								ResurrectionA := ResurrectionH*60+ResurrectionM
								log.Println("ResurrectionA-"+strconv.Itoa(ResurrectionA))

								if NOWTIME - ResurrectionA <=  10 {
									if _, err := bot.PushMessage(userID, linebot.NewTextMessage("BOSS APPEARANCE-"+p.KingOfName )).Do(); err != nil {
										log.Print(err)
									}
								}
								
							}
						case <- doneChan:
							log.Println("Done")
							return
							}
						}
					}()
				}
				 if string(message.Text[0]) == "@" {
					result := strings.Split(message.Text," ")
					log.Println("result[0]-"+result[0])
					if result[0] == "@BOSS" {
						log.Println("result[2]-"+result[2])
						if result[2] == "Die" {
							log.Println("result[3]-"+result[3])
							if result[3] != "" {
								log.Println("load page....")
								pages := getPages()
								for i, _ := range pages {
									log.Println("p.KingOfName-"+pages[i].KingOfName)
									log.Println("compare ...."+ result[1])
									if result[1] == pages[i].KingOfName {
										pages[i].Die = result[3]
										log.Println("assiagn die time ...."+ pages[i].Die)
										break
									}
								}
								pagesJson, _ := json.Marshal(pages)
								err = ioutil.WriteFile("BossRefreshInfo.json", pagesJson, 0644)
								if err != nil {
									log.Println(err)
								}
								log.Println("WriteFile ...."+string(pagesJson) )
							}
						}
					}
					
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text+"--"+ strconv.Itoa( time.Now().In(local).Hour() )+"-"+strconv.Itoa( time.Now().In(local).Minute() )+"-"+strconv.Itoa( time.Now().In(local).Second() ) )).Do(); err != nil {
						log.Print(err)
					}
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
