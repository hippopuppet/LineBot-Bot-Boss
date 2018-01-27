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
	"bytes"

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
	checkBossTimer := time.NewTicker(time.Second*60).C
	var local *time.Location
	local, ok := time.LoadLocation("Asia/Taipei")
	log.Print(ok)
	go func() {
		for {
		select {
			case <- checkBossTimer:
				log.Println("CHECK BOSS RESURRECTION !! ")
				NOWTIME := time.Now().In(local).Hour()*60+time.Now().In(local).Minute()+10
				log.Println("NOWTIME-"+strconv.Itoa(NOWTIME))

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
					log.Fatal(err)
				}
								
				for _, bossinfo := range dbResult[0].BossInfo {
					bossinfo_Resurrection, err := strconv.Atoi(bossinfo.Resurrection)
					if err != nil {
						log.Print(err)
					}
					ResurrectionA := convertTimetoMinute(bossinfo_Resurrection)
					JetLag := NOWTIME - ResurrectionA
					if JetLag <= 10 && JetLag > 0 {
						for _, groupinfo := range dbResult[0].GroupInfo {
							if groupinfo.Active == 1 {
								if _, err := bot.PushMessage(groupinfo.Id, linebot.NewTextMessage("BOSS APPEARANCE: --"+bossinfo.KingOfName +"--")).Do(); err != nil {
									log.Print(err)
								}
							}
						}
					}//if JetLag <= 10 && JetLag > 0
				}	
					
			case <- doneChan:
				log.Println("Done")
				return
			}
		}
	}()

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
			
			switch message := event.Message.(type) {
			case *linebot.TextMessage:

				 if string(message.Text[0]) == "!" {
					result := strings.Split(message.Text," ")
					if message.Text == "!STOP" {
						if _, err := bot.PushMessage(event.Source.GroupID, linebot.NewTextMessage("STOP CALL ATTENTION TO BOSS RESURRECTION !! ")).Do(); err != nil {
							log.Print(err)
						}
					//doneChan <- true
					}
					if message.Text == "!START" {
						//[CONNECT DB]
						session, err := mgo.Dial("mongodb://heroku_xzzlp7s1:heroku_xzzlp7s1@ds111598.mlab.com:11598/heroku_xzzlp7s1")
						if err != nil {
							panic(err)
						}
						defer session.Close()
						// Optional. Switch the session to a monotonic behavior.
						session.SetMode(mgo.Monotonic, true)
						c := session.DB("heroku_xzzlp7s1").C("bossinfo")

						colQuerier := bson.M{"GROUPINFO.id" : event.Source.GroupID}
						upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.active":1}}
						err = c.Update(colQuerier, upsertData)
						if err != nil {
							log.Println(err)
						}
						
						if _, err := bot.PushMessage(event.Source.GroupID, linebot.NewTextMessage("START CALL ATTENTION TO BOSS RESURRECTION !! ")).Do(); err != nil {
							log.Print(err)
						}
					}
					if result[0] == "!BOSS" {
						if result[2] == "Die" {
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
									if result[1] == dbResult[0].BossInfo[i].KingOfName {										
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
					if message.Text == "!LIST" {
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
							log.Fatal(err)
						}
						var list_buf bytes.Buffer
						for i, bossinfo := range dbResult[0].BossInfo {
							list_buf.WriteString(strconv.Itoa(i+1))
							list_buf.WriteString(". ")
							list_buf.WriteString(bossinfo.KingOfName)
							list_buf.WriteString(" : ")
							list_buf.WriteString(bossinfo.Resurrection)
							list_buf.WriteString("\n")							
						}
     
						if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(list_buf.String())).Do(); err != nil {
							log.Print(err)
						}
					}//!LIST
    
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

			// Find
			var dbResult []JSONDATA
			err = c.Find(nil).All(&dbResult)
			if err != nil {
				log.Println(err)
			}
			// Upsert
			index := len(dbResult[0].GroupInfo)
			if index <= 0 {
				log.Println("index <= 0 ")
				upsertData := bson.M{"$push": bson.M{"GROUPINFO": bson.M{"id": event.Source.GroupID, "type": "group", "active":0}}}
				info, err := c.UpsertId(bson.ObjectIdHex("5a69aa488d0d213fd88abd95"), upsertData)
				if err != nil {
					log.Println(err)
				}
				log.Println(info)
			}
			if index > 0 {
				log.Println("index > 0 ")
				colQuerier := bson.M{"GROUPINFO.id" : event.Source.GroupID}
				upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.id": event.Source.GroupID, "GROUPINFO.$.type": "group", "GROUPINFO.$.active":0}}
				info, err := c.Upsert(colQuerier, upsertData)
				if err != nil {
					log.Println(err)
				}
				log.Println(info)
			}
			originalContentURL := "https://i.imgur.com/Qr2DKSG.jpg"
			previewImageURL := "https://i.imgur.com/Qr2DKSG.jpg"
			message := linebot.NewImageMessage(originalContentURL, previewImageURL)
			if _, err = bot.ReplyMessage(event.ReplyToken, message).Do(); err != nil {
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
			var dbResult []JSONDATA
			err = c.Find(nil).All(&dbResult)
			if err != nil {
				log.Println(err)
			}
			// Upsert
			index := len(dbResult[0].GroupInfo)
			/*if index <= 0 {
				log.Println("index <= 0 ")
				upsertData := bson.M{"$push": bson.M{"GROUPINFO": bson.M{"id": event.Source.UserID, "type": "user", "active":0}}}
				//upsertData := bson.M{"$push": bson.M{"GROUPINFO.$.id": event.Source.UserID, "GROUPINFO.$.type": "user", "GROUPINFO.$.active":0}}
				info, err := c.UpsertId(bson.ObjectIdHex("5a69aa488d0d213fd88abd95"), upsertData)
				if err != nil {
					log.Println(err)
				}
				log.Println(info)
			}*/
			
			log.Println("index > 0 ")
			colQuerier := bson.M{"GROUPINFO.id" : event.Source.UserID}
			upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.id": event.Source.UserID, "GROUPINFO.$.type": "user", "GROUPINFO.$.active":0}}
			//upsertData := bson.M{"$set": bson.M{"GROUPINFO": bson.M{ "id": event.Source.UserID, "type": "user", "active":0}}}
			info, err := c.Upsert(colQuerier, upsertData)
			if err != nil {
				log.Println(err)
				if err == mgo.ErrNotFound {
					upsertData := bson.M{"$push": bson.M{"GROUPINFO": bson.M{"id": event.Source.UserID, "type": "user", "active":0}}}
					info, err := c.UpsertId(bson.ObjectIdHex("5a69aa488d0d213fd88abd95"), upsertData)
					if err != nil {
						log.Println(err)
					}
					log.Println(info)
				}
			}
			log.Println(info)
			
		
		originalContentURL := "https://i.imgur.com/Qr2DKSG.jpg"
		previewImageURL := "https://i.imgur.com/Qr2DKSG.jpg"
		message := linebot.NewImageMessage(originalContentURL, previewImageURL)
		if _, err = bot.ReplyMessage(event.ReplyToken, message).Do(); err != nil {
				log.Print(err)
		}
		}//
	}

	
}
