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

type JSONDATA struct {
    BossInfo []BOSSINFO `bson:"BOSSINFO" json:"BOSSINFO"`
}

type BOSSINFO struct {
    KingOfName  string `bson:"kingofname" json:"kingofname"`
	RefreshTick string `bson:"refreshtick" json:"refreshtick"`
	Die string `bson:"die" json:"die"`
    Resurrection string `bson:"resurrection" json:"resurrection"`
}

func convertTimetoMinute(orgTime int) int {
	H := orgTime/100
	M := orgTime - (H*100)
	A := H*60+M
	
	return A
}

func convertMinutetoTime(orgMinute int) int {
	H := orgMinute/60
	M := orgMinute%60
	T := (H*100)+M
	
	return T
}

func (p JSONDATA) toString() string {
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

func getBossJson() JSONDATA {
   raw, err := ioutil.ReadFile("./BossRefreshInfo.json")
    if err != nil {
        log.Println(err.Error())
        os.Exit(1)
    }

    var c JSONDATA
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

								log.Println("CONNECT DB....")
								//[CONNECT DB]
								session, err := mgo.Dial("mongodb://heroku_xzzlp7s1:heroku_xzzlp7s1@ds111598.mlab.com:11598/heroku_xzzlp7s1")
								if err != nil {
								   panic(err)
								}
								defer session.Close()

								// Optional. Switch the session to a monotonic behavior.
								session.SetMode(mgo.Monotonic, true)

								c := session.DB("heroku_xzzlp7s1").C("bossinfo")
								log.Println("find data...")
								var dbResult []JSONDATA
								err = c.Find(nil).All(&dbResult)
								if err != nil {
								   log.Fatal(err)
								}
								
								for _, bossinfo := range dbResult[0].BossInfo {
								log.Println("bossinfo.Resurrection "+bossinfo.Resurrection)
								bossinfo_Resurrection, err := strconv.Atoi(bossinfo.Resurrection)
								if err != nil {
									log.Print(err)
								}
								ResurrectionA := convertTimetoMinute(bossinfo_Resurrection)
								log.Println("ResurrectionA "+strconv.Itoa(ResurrectionA))

								JetLag := NOWTIME - ResurrectionA
								log.Println("JetLag "+strconv.Itoa(JetLag))
								if JetLag < 0 {
									JetLag = -JetLag
								}
								log.Println("UJetLag "+strconv.Itoa(JetLag))

								if JetLag <= 10 {
									if groupID != ""{
										if _, err := bot.PushMessage(groupID, linebot.NewTextMessage("BOSS APPEARANCE: "+bossinfo.KingOfName )).Do(); err != nil {
											log.Print(err)
										}
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
								log.Println("CONNECT DB....")
								//[CONNECT DB]
								session, err := mgo.Dial("mongodb://heroku_xzzlp7s1:heroku_xzzlp7s1@ds111598.mlab.com:11598/heroku_xzzlp7s1")
								if err != nil {
								   panic(err)
								}
								defer session.Close()

								// Optional. Switch the session to a monotonic behavior.
								session.SetMode(mgo.Monotonic, true)

								c := session.DB("heroku_xzzlp7s1").C("bossinfo")
								log.Println("find data...")
								var dbResult []JSONDATA
								err = c.Find(nil).All(&dbResult)
								if err != nil {
								   log.Fatal(err)
								}
								
								for i, _ := range dbResult[0].BossInfo {
									log.Println("p.KingOfName-"+dbResult[0].BossInfo[i].KingOfName)
									log.Println("compare ...."+ result[1])
									if result[1] == dbResult[0].BossInfo[i].KingOfName {
										log.Println("assiagn die time ...."+ dbResult[0].BossInfo[i].Die)
										dbResult[0].BossInfo[i].Die = result[3]
										intNewDie, err := strconv.Atoi(result[3])
										if err != nil {
											log.Print(err)
										}
										intNewDieMinute := convertTimetoMinute(intNewDie)
										intRefreshTick, err := strconv.Atoi(dbResult[0].BossInfo[i].RefreshTick)
										if err != nil {
											log.Print(err)
										}
										intNewDieTime := convertMinutetoTime(intNewDieMinute + intRefreshTick)
										strNewDieTime := strconv.Itoa(intNewDieTime)
										
										dbResult[0].BossInfo[i].Resurrection = strNewDieTime
										log.Println("calaculate resurrection .... "+ dbResult[0].BossInfo[i].Resurrection)

										var dbM []bson.M
										err = c.Find(nil).All(&dbM)
										if err != nil {
										   log.Fatal(err)
										}
										log.Println("dbM ...")
										log.Println(dbM)

										// Update
										//colQuerier := bson.M{"kingofname": dbResult[0].BossInfo[i].KingOfName} 
										colQuerier := bson.M{"_id": ObjectIdHex("5a688511d1bd33c6d81b1abb") } 
										log.Print("kingofname: ")
										log.Println(colQuerier)

										change := bson.M{"$set": bson.M{"die": dbResult[0].BossInfo[i].Die, "resurrection": dbResult[0].BossInfo[i].Resurrection}}
										err = c.Update(colQuerier, change)
										if err != nil {
											panic(err)
										}
										
										break
									}
								}
								
								/*JsonData, err := json.Marshal(dbResult)
								if err != nil {
									log.Print(err)
								}
								log.Println("Marshal result: ...")
								log.Println(string(JsonData))*/
							}
						}// ==Die
					}// ==@BOSS
    
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text+"--"+ strconv.Itoa( time.Now().In(local).Hour() )+"-"+strconv.Itoa( time.Now().In(local).Minute() )+"-"+strconv.Itoa( time.Now().In(local).Second() ) )).Do(); err != nil {
						log.Print(err)
					}
				}// ==@
				
				
			
				
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
