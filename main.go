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
	GroupInfo []GROUPINFO `bson:"GROUPINFO" json:"GROUPINFO"`
}

type BOSSINFO struct {
    KingOfName  string `bson:"kingofname" json:"kingofname"`
	RefreshTick string `bson:"refreshtick" json:"refreshtick"`
	Die string `bson:"die" json:"die"`
    Resurrection string `bson:"resurrection" json:"resurrection"`
}

type GROUPINFO struct {
    Id  string `bson:"id" json:"id"`
	Type string `bson:"type" json:"type"`
	Active int `bson:"active" json:"active"`
}

func convertTimetoMinute(orgTime int) int {
	H := orgTime/100
	M := orgTime - (H*100)
	A := H*60+M
	
	return A
}

func convertMinutetoTime(orgMinute int) int {
	H := orgMinute/60
	HH := H%24
	M := orgMinute%60
	T := (HH*100)+M
	
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
				if message.Text == "STOP" {
					if groupID != ""{
						if _, err := bot.PushMessage(groupID, linebot.NewTextMessage("STOP CALL ATTENTION TO BOSS RESURRECTION !! ")).Do(); err != nil {
							log.Print(err)
						}
					}
					doneChan <- true
				}
				/*if message.Text == "STOP" {
					checkBossTimer.Stop()
				}*/
				 if message.Text == "START" {
					log.Println("START CALL ATTENTION TO BOSS RESURRECTION !! ")
					checkBossTimer := time.NewTicker(time.Second*60).C
					if groupID != ""{
						if _, err := bot.PushMessage(groupID, linebot.NewTextMessage("START CALL ATTENTION TO BOSS RESURRECTION !! ")).Do(); err != nil {
							log.Print(err)
						}
					}
					go func() {
						for {
						select {
							case <- checkBossTimer:
								log.Println("CHECK BOSS RESURRECTION !! ")
								NOWTIME := time.Now().In(local).Hour()*60+time.Now().In(local).Minute()+10
								log.Println("NOWTIME-"+strconv.Itoa(NOWTIME))

								//log.Println("CONNECT DB....")
								//[CONNECT DB]
								session, err := mgo.Dial("mongodb://heroku_xzzlp7s1:heroku_xzzlp7s1@ds111598.mlab.com:11598/heroku_xzzlp7s1")
								if err != nil {
								   panic(err)
								}
								defer session.Close()

								// Optional. Switch the session to a monotonic behavior.
								session.SetMode(mgo.Monotonic, true)

								c := session.DB("heroku_xzzlp7s1").C("bossinfo")
								//log.Println("find data...")
								var dbResult []JSONDATA
								err = c.Find(nil).All(&dbResult)
								if err != nil {
								   log.Fatal(err)
								}
								
								for _, bossinfo := range dbResult[0].BossInfo {
								//log.Println("bossinfo.Resurrection "+bossinfo.Resurrection)
								bossinfo_Resurrection, err := strconv.Atoi(bossinfo.Resurrection)
								if err != nil {
									log.Print(err)
								}
								ResurrectionA := convertTimetoMinute(bossinfo_Resurrection)
								//log.Println("ResurrectionA "+strconv.Itoa(ResurrectionA))

								JetLag := NOWTIME - ResurrectionA
								//log.Println("JetLag "+strconv.Itoa(JetLag))
								//if JetLag < 0 {
								//	JetLag = -JetLag
								//}
								//log.Println("UJetLag "+strconv.Itoa(JetLag))

								if JetLag <= 10 && JetLag > 0 {
									if groupID != ""{
										if _, err := bot.PushMessage(groupID, linebot.NewTextMessage("BOSS APPEARANCE: --"+bossinfo.KingOfName +"--")).Do(); err != nil {
											log.Print(err)
										}
									}
									if userID != ""{
										if _, err := bot.PushMessage(userID, linebot.NewTextMessage("BOSS APPEARANCE: --"+bossinfo.KingOfName +"--")).Do(); err != nil {
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
				 if string(message.Text[0]) == "!" {
					result := strings.Split(message.Text," ")
					log.Println("result[0]-"+result[0])
					if result[0] == "!BOSS" {
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
								   log.Println(err)
								}
								isFound := false
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
										
										// Update
										colQuerier := bson.M{"BOSSINFO.kingofname": dbResult[0].BossInfo[i].KingOfName}
										change := bson.M{"$set": bson.M{"BOSSINFO.$.die": dbResult[0].BossInfo[i].Die, "BOSSINFO.$.resurrection": dbResult[0].BossInfo[i].Resurrection}}
										//id := bson.ObjectIdHex("5a69a0718d0d213fd88abd92")
										err = c.Update(colQuerier, change)
										if err != nil {
											log.Println(err)
										}
										if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("UPDATE BOSS:"+dbResult[0].BossInfo[i].KingOfName+" INFO SUCCESS.")).Do(); err != nil {
											log.Print(err)
										}
										isFound = true
										break
									}
								}
								
								if isFound == false {
									if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("HAS NO BOSS:"+ result[1])).Do(); err != nil {
										log.Print(err)
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
					}// ==!BOSS
    
				}// ==!
				
				
			
				
			}
			
		}

		if event.Type == linebot.EventTypeJoin {
			//[CONNECT DB]
			session, err := mgo.Dial("mongodb://heroku_xzzlp7s1:heroku_xzzlp7s1@ds111598.mlab.com:11598/heroku_xzzlp7s1")
			if err != nil {
				panic(err)
			}
			defer session.Close()
			// Optional. Switch the session to a monotonic behavior.
			session.SetMode(mgo.Monotonic, true)
			c := session.DB("heroku_xzzlp7s1").C("bossinfo")
			
			var dbResult []JSONDATA
			err = c.Find(nil).All(&dbResult)
			if err != nil {
				log.Println(err)
			}
			// Upsert
			/*index := len(dbResult[0].GroupInfo)
			log.Print("index ...............   ")
			log.Println(index)
			
			colQuerier := bson.M{"GROUPINFO.id": event.Source.GroupID}
			change := bson.M{"$set": bson.M{"GROUPINFO.$index.id": event.Source.GroupID, "GROUPINFO.$index.type": "group", "GROUPINFO.$index.active":0}}
			info, err := c.Upsert(colQuerier, change)
			if err != nil {
				log.Println(err)
			}
			log.Println(info)*/
			/*
			// Find
			var dbResult bson.M
			err = c.Find(bson.M{"GROUPINFO.id": event.Source.GroupID}).One(&dbResult)
			if err != nil {
				if err == mgo.ErrNotFound {
					//Insert
					insertData := bson.M{"GROUPINFO.id": event.Source.GroupID, "GROUPINFO.type":event.Source.Type, "GROUPINFO.active": 0}
					err = c.Insert(insertData)
					if err != nil {
						panic(err)
					}
				}
			}
			*/
			if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(event.Source.GroupID)).Do(); err != nil {
					log.Print(err)
			}
		}

		if event.Type == linebot.EventTypeFollow {
			

			//[CONNECT DB]
			session, err := mgo.Dial("mongodb://heroku_xzzlp7s1:heroku_xzzlp7s1@ds111598.mlab.com:11598/heroku_xzzlp7s1")
			if err != nil {
				panic(err)
			}
			defer session.Close()
			// Optional. Switch the session to a monotonic behavior.
			session.SetMode(mgo.Monotonic, true)
			c := session.DB("heroku_xzzlp7s1").C("bossinfo")

			// Find
			/*var dbResult []JSONDATA
			err = c.Find(nil).All(&dbResult)
			if err != nil {
				log.Println(err)
			}
			// Upsert
			index := len(dbResult[0].GroupInfo)
			log.Print("index ...............   ")
			log.Println(index)*/
			usertData := JSONDATA{}
			usertData.GroupInfo.Id := event.Source.GroupID
			usertData.GroupInfo.Type := "group"
			usertData.GroupInfo.Active := 0
			colQuerier := bson.M{"GROUPINFO.id": event.Source.GroupID}
			//change := bson.M{"$set": bson.M{"GROUPINFO.$.id": event.Source.GroupID, "GROUPINFO.$.type": "group", "GROUPINFO.$.active":0}}
			info, err := c.Upsert(colQuerier, &usertData)
			if err != nil {
				log.Println(err)
			}
			log.Println(info)
		}
	}

	
}
