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
	"strings"
	"bytes"

	"github.com/line/line-bot-sdk-go/linebot"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type JSONDATA struct {
    BossInfo []BOSSINFO `bson:"BOSSINFO" json:"BOSSINFO"`
	BossInfo_ []BOSSINFO `bson:"_BOSSINFO" json:"_BOSSINFO"`
	GroupInfo []GROUPINFO `bson:"GROUPINFO" json:"GROUPINFO"`
}

type BOSSINFO struct {
    KingOfName  string `bson:"kingofname" json:"kingofname"`
	RefreshTick string `bson:"refreshtick" json:"refreshtick"`
	Die string `bson:"die" json:"die"`
    Resurrection string `bson:"resurrection" json:"resurrection"`
	Map string `bson:"map" json:"map"`
    UpdateDate string `bson:"updatedate" json:"updatedate"`
	Author string `bson:"author" json:"author"`
	Category string `bson:"category" json:"category"`
}

type GROUPINFO struct {
    Id  string `bson:"id" json:"id"`
	Type string `bson:"type" json:"type"`
	Active int `bson:"active" json:"active"`
	License int `bson:"license" json:"license"`
	CreateTime string `bson:"createtime" json:"createtime"`
}

type AIRINFO struct {
	CO string `bson:"CO" json:"CO"`
	County string `bson:"County" json:"County"`
	FPMI string `bson:"FPMI" json:"FPMI"`
	MajorPollutant string `bson:"MajorPollutant" json:"MajorPollutant"`
	NO string `bson:"NO" json:"NO"`
	NO2 string `bson:"NO2" json:"NO2"`
	NOx string `bson:"NOx" json:"NOx"`
	O3 string `bson:"O3" json:"O3"`
	PM10 string `bson:"PM10" json:"PM10"`
	PM2_5 string `bson:"PM2.5" json:"PM2.5"`
	PSI string `bson:"PSI" json:"PSI"`
	PublishTime string `bson:"PublishTime" json:"PublishTime"`
	SiteName string `bson:"SiteName" json:"SiteName"`
	SO2 string `bson:"SO2" json:"SO2"`
	Status string `bson:"Status" json:"Status"`
	WindDirec string `bson:"WindDirec" json:"WindDirec"`
	WindSpeed string `bson:"WindSpeed" json:"WindSpeed"`
}

type STOCKINFO struct {
	TS string `bson:"ts" json:"ts"`
	FV string `bson:"fv" json:"fv"`
	TK0 string `bson:"tk0" json:"tk0"`
	TK1 string `bson:"tk1" json:"tk1"`
	OA string `bson:"oa" json:"oa"`
	OB string `bson:"ob" json:"ob"`
	TLONG string `bson:"tlong" json:"tlong"`
	OT string `bson:"ot" json:"ot"`
	F string `bson:"f" json:"f"`
	EX string `bson:"ex" json:"ex"`
	G string `bson:"g" json:"g"`
	OV string `bson:"ov" json:"ov"`
	D string `bson:"d" json:"d"`
	IT string `bson:"it" json:"it"`
	B string `bson:"b" json:"b"`
	C string `bson:"c" json:"c"`
	MT string `bson:"mt" json:"mt"`
	A string `bson:"a" json:"a"`
	N string `bson:"n" json:"n"`
	O string `bson:"o" json:"o"`
	L string `bson:"l" json:"l"`
	OZ string `bson:"oz" json:"oz"`
	H string `bson:"h" json:"h"`
	IP string `bson:"ip" json:"ip"`
	I string `bson:"i" json:"i"`
	W string `bson:"w" json:"w"`
	V string `bson:"v" json:"v"`
	U string `bson:"u" json:"u"`
	T string `bson:"t" json:"t"`
	S string `bson:"s" json:"s"`
	PZ string `bson:"pz" json:"pz"`
	TV string `bson:"tv" json:"tv"`
	P string `bson:"p" json:"p"` 
	NF string `bson:"nf" json:"nf"`
	CH string `bson:"ch" json:"ch"`
	Z string `bson:"z" json:"z"`
	Y string `bson:"y" json:"y"`
	PS string `bson:"ps" json:"ps"`
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

func getJson(url string, result interface{}) error {
	//url := "http://opendata2.epa.gov.tw/AQI.json"
	resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("cannot fetch URL %q: %v", url, err)
    }
    defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected http GET status: %s", resp.Status)
    }
    // We could check the resulting content type
    // here if desired.
    err = json.NewDecoder(resp.Body).Decode(&result)
    if err != nil {
        return fmt.Errorf("cannot decode JSON: %v", err)
    }
    return nil

   /*raw, err := ioutil.ReadFile("http://opendata.epa.gov.tw/ws/Data/REWIQA/?$orderby=SiteName&amp;$skip=0&amp;$top=1000&amp;format=json")
    if err != nil {
        log.Println(err.Error())
        //os.Exit(1)
    }
    var c AIRINFO
    json.Unmarshal(raw, &c)
    return c*/
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
					if JetLag == 5/* || JetLag == 7*/ {
						
						for _, groupinfo := range dbResult[0].GroupInfo {
							if groupinfo.Active == 1 /*&& groupinfo.License == 1*/{
								if _, err := bot.PushMessage(groupinfo.Id, linebot.NewTextMessage("BOSS : "+bossinfo.KingOfName +"將在"+bossinfo.Resurrection+"重生! Map: "+ bossinfo.Map)).Do(); err != nil {
									log.Print(err)
								}
							}
						}
					}
				}	
				for _, bossinfo := range dbResult[0].BossInfo_ {
					bossinfo_Resurrection, err := strconv.Atoi(bossinfo.Resurrection)
					if err != nil {
						log.Print(err)
					}
					ResurrectionA := convertTimetoMinute(bossinfo_Resurrection)
					JetLag := NOWTIME - ResurrectionA
					if JetLag == 5 {
						
						if _, err := bot.PushMessage("C7e98ffa1fcb2d0a399cd27aca6bffca0", linebot.NewTextMessage("[革]BOSS : "+bossinfo.KingOfName +"將在"+bossinfo.Resurrection+"重生! Map: "+ bossinfo.Map)).Do(); err != nil {
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
						upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.active":0}}
						err = c.Update(colQuerier, upsertData)
						if err != nil {
							log.Println(err)
						}

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
						if result[2] != "" {
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

							for _, groupinfo := range dbResult[0].GroupInfo {
								if groupinfo.Id == event.Source.GroupID {
									//if groupinfo.License == 1 {
										isFound := false
										for i, _ := range dbResult[0].BossInfo {
											if result[1] == dbResult[0].BossInfo[i].KingOfName {										
												dbResult[0].BossInfo[i].Die = result[2]
												intNewDie, err := strconv.Atoi(result[2])
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
												lens := len(strNewDieTime)
												var list_buf bytes.Buffer
												for  i := 0 ; i < 4-lens ; i++ {
													list_buf.WriteString("0")
												}
												list_buf.WriteString(strNewDieTime)		
												dbResult[0].BossInfo[i].Resurrection = list_buf.String()
									
												var local *time.Location
												local, ok := time.LoadLocation("Asia/Taipei")
												log.Print(ok)
												_NowTime := time.Now().In(local)
												dbResult[0].BossInfo[i].UpdateDate = _NowTime.Format("2006-01-02 15:04:05")

												profile, err := bot.GetProfile(event.Source.UserID).Do();
												if err != nil {
													log.Println(err)
												}
												dbResult[0].BossInfo[i].Author = profile.DisplayName
												// Update
												colQuerier := bson.M{"BOSSINFO.kingofname": dbResult[0].BossInfo[i].KingOfName}
												change := bson.M{"$set": bson.M{"BOSSINFO.$.die": dbResult[0].BossInfo[i].Die, "BOSSINFO.$.resurrection": dbResult[0].BossInfo[i].Resurrection,"BOSSINFO.$.updatedate": dbResult[0].BossInfo[i].UpdateDate,"BOSSINFO.$.author":dbResult[0].BossInfo[i].Author}}
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
									//}
								}
							}
							
						}
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
							list_buf.WriteString("   Map: ")
							list_buf.WriteString(bossinfo.Map)
							list_buf.WriteString("   Last Upate: ")
							list_buf.WriteString(bossinfo.UpdateDate)
							list_buf.WriteString(" from: ")
							list_buf.WriteString(bossinfo.Author)
							list_buf.WriteString("\n")							
						}

						for _, groupinfo := range dbResult[0].GroupInfo {
							if groupinfo.Id == event.Source.GroupID {
								//if groupinfo.License == 1 {
									if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(list_buf.String())).Do(); err != nil {
										log.Print(err)
									}
								//}
							}							
						}
						
					}//!LIST
					if result[0] == "!PM" {
						if len(result) > 1 {
							var airJson []AIRINFO
							var airresult interface{}
							getJson("http://opendata2.epa.gov.tw/AQI.json", &airresult)
							airByte, _ := json.Marshal(airresult)
							json.Unmarshal(airByte, &airJson)
							//err = json.NewDecoder(airresult).Decode(&airJson)
							//if err != nil {
							//	log.Println("cannot decode JSON: %v", err)
							//}
							log.Println(airJson)
							isFound := false
							for _, airinfo := range airJson {
								if airinfo.SiteName == result[1] {
									var airinfo_buf bytes.Buffer
									airinfo_buf.WriteString(airinfo.SiteName)
									airinfo_buf.WriteString("的 PM2.5 數值為 ")
									airinfo_buf.WriteString(airinfo.PM2_5)
									airinfo_buf.WriteString("狀態 ")
									airinfo_buf.WriteString(airinfo.Status)

									if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(airinfo_buf.String())).Do(); err != nil {
										log.Print(err)
									}
									isFound = true
								}
							}
							if isFound == false {
								var airinfo_buf bytes.Buffer
								airinfo_buf.WriteString("沒有")
								airinfo_buf.WriteString(result[1])
								airinfo_buf.WriteString("的 PM2.5 資料")
								if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(airinfo_buf.String())).Do(); err != nil {
										log.Print(err)
								}
							}
						} else {
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("請輸入欲查詢PM2.5資料的地點")).Do(); err != nil {
									log.Print(err)
							}
						}
					}//!PM

    
				}// ==!
				//if event.Source.GroupID == "C7e98ffa1fcb2d0a399cd27aca6bffca0" {
					if string(message.Text[0]) == "#" {
						result := strings.Split(message.Text," ")
					
						if result[0] == "#BOSS" {
							if result[2] != "" {
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

								//for _, groupinfo := range dbResult[0].GroupInfo {
									//if groupinfo.Id == event.Source.GroupID {
										//if groupinfo.License == 1 {
											isFound := false
											for i, _ := range dbResult[0].BossInfo_ {
												if result[1] == dbResult[0].BossInfo_[i].KingOfName {										
													dbResult[0].BossInfo_[i].Die = result[2]
													intNewDie, err := strconv.Atoi(result[2])
													if err != nil {
														log.Print(err)
													}
													intNewDieMinute := convertTimetoMinute(intNewDie)
													intRefreshTick, err := strconv.Atoi(dbResult[0].BossInfo_[i].RefreshTick)
													if err != nil {
														log.Print(err)
													}
													intNewDieTime := convertMinutetoTime(intNewDieMinute + intRefreshTick)
													strNewDieTime := strconv.Itoa(intNewDieTime)
													lens := len(strNewDieTime)
													var list_buf bytes.Buffer
													for  i := 0 ; i < 4-lens ; i++ {
														list_buf.WriteString("0")
													}
													list_buf.WriteString(strNewDieTime)		
													dbResult[0].BossInfo_[i].Resurrection = list_buf.String()
									
													var local *time.Location
													local, ok := time.LoadLocation("Asia/Taipei")
													log.Print(ok)
													_NowTime := time.Now().In(local)
													dbResult[0].BossInfo_[i].UpdateDate = _NowTime.Format("2006-01-02 15:04:05")

													profile, err := bot.GetProfile(event.Source.UserID).Do();
													if err != nil {
														log.Println(err)
													}
													dbResult[0].BossInfo_[i].Author = profile.DisplayName
													// Update
													colQuerier := bson.M{"_BOSSINFO.kingofname": dbResult[0].BossInfo_[i].KingOfName}
													change := bson.M{"$set": bson.M{"_BOSSINFO.$.die": dbResult[0].BossInfo_[i].Die, "_BOSSINFO.$.resurrection": dbResult[0].BossInfo_[i].Resurrection,"_BOSSINFO.$.updatedate": dbResult[0].BossInfo_[i].UpdateDate,"_BOSSINFO.$.author":dbResult[0].BossInfo_[i].Author}}
													//id := bson.ObjectIdHex("5a69a0718d0d213fd88abd92")
													err = c.Update(colQuerier, change)
													if err != nil {
														log.Println(err)
													}
													if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("UPDATE BOSS:"+dbResult[0].BossInfo_[i].KingOfName+" INFO SUCCESS.")).Do(); err != nil {
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
										//}
									//}
								//}
							
							}
						}// ==#BOSS
						if message.Text == "#LIST" {
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
							log.Print(dbResult)
							//log.Print(dbResult[0].BossInfo)
							var list_buf bytes.Buffer
							list_buf.WriteString("[革命]")
							list_buf.WriteString("\n")
							for i, bossinfo := range dbResult[0].BossInfo_ {
								list_buf.WriteString(strconv.Itoa(i+1))
								list_buf.WriteString(". ")
								list_buf.WriteString(bossinfo.KingOfName)
								list_buf.WriteString(" : ")
								list_buf.WriteString(bossinfo.Resurrection)
								list_buf.WriteString("   Map: ")
								list_buf.WriteString(bossinfo.Map)
								list_buf.WriteString("   Last Upate: ")
								list_buf.WriteString(bossinfo.UpdateDate)
								list_buf.WriteString(" from: ")
								list_buf.WriteString(bossinfo.Author)
								list_buf.WriteString("\n")							
							}
							
							/*for _, groupinfo := range dbResult[0].GroupInfo {
								if groupinfo.Id == event.Source.GroupID {
									if groupinfo.License == 1 {
										if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(list_buf.String())).Do(); err != nil {
											log.Print(err)
										}
									}
								}							
							}*/
							/*if _, err := bot.PushMessage("C7e98ffa1fcb2d0a399cd27aca6bffca0", linebot.NewTextMessage(list_buf.String())).Do(); err != nil {
								log.Print(err)
							}*/
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(list_buf.String())).Do(); err != nil {
								log.Print(err)
							}
						
						}//#LIST
					

    
					}// ==#
				//}

				if string(message.Text[0]) == "P" {
					var local *time.Location
					local, _ = time.LoadLocation("Asia/Taipei")
					_Year,_Month,_Day := time.Now().In(local).Date()
					_strMonth := strconv.Itoa(int(_Month))
					_strDay := strconv.Itoa(int(_Day))
					
					lens := len(_strMonth)
					var month_buf bytes.Buffer
					for  i := 0 ; i < 2-lens ; i++ {
						month_buf.WriteString("0")
					}
					month_buf.WriteString(_strMonth)

					lens = len(_strDay)
					var day_buf bytes.Buffer
					for  i := 0 ; i < 2-lens ; i++ {
						day_buf.WriteString("0")
					}
					day_buf.WriteString(_strDay)

					var date_buf bytes.Buffer
					date_buf.WriteString(strconv.Itoa(_Year))
					date_buf.WriteString(month_buf.String())
					date_buf.WriteString(day_buf.String())
					
					CURRENT_NANO := time.Now().In(local).UnixNano()/int64(time.Millisecond)
					CURRENT_MILLIS := strconv.FormatInt(CURRENT_NANO, 10)
					
					stocknum := message.Text[1:]
					var stockJson []STOCKINFO
					var stockresult interface{}

					//u, err := url.Parse("http://mis.twse.com.tw/stock/fibest.jsp?stock=1101")
					
					req, err := http.NewRequest("GET", "http://mis.twse.com.tw/stock/index.jsp", nil)
					//cookie := req.Cookie{Name: "JSESSIONID", Value: "748555A2619B2CB7A12250B0C5B42267", Path: "http://mis.twse.com.tw/", MaxAge: 86400}
					/*http.SetCookie(w, &cookie)
					req, err := http.NewRequest("GET", "http://mis.twse.com.tw", nil)
					*/if err != nil {
						log.Println(err)
					}
					log.Println(req)

					URL := "http://mis.twse.com.tw/stock/api/getStockInfo.jsp?ex_ch=tse_"+stocknum+".tw&json=1&delay=0&d="+date_buf.String()+"&_="+CURRENT_MILLIS
					log.Print("URL")
					log.Print(URL)
					getJson(URL, &stockresult)
					stockByte, _ := json.Marshal(stockresult)
					log.Print("stockByte")
					log.Print(string(stockByte))
					json.Unmarshal(stockByte, &stockJson)
					log.Print("stockJson")
					log.Print(stockJson)
					
				}// == P
				
				/*
				if string(message.Text[0]) != "!" && string(message.Text[0]) != "P"{
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
					profile, err := bot.GetGroupMemberProfile( event.Source.GroupID, event.Source.UserID).Do();
					if err != nil {
						log.Println(err)
					}
					dbResult[0].BossInfo[i].Author = profile.DisplayName

					// Upsert
					colQuerier := bson.M{"GROUPINFO.id" : event.Source.GroupID}
					upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.id": event.Source.GroupID, "GROUPINFO.$.type": "group", "GROUPINFO.$.active":0, "GROUPINFO.$.license":0}}
					//upsertData := bson.M{"$set": bson.M{"GROUPINFO": bson.M{ "id": event.Source.GroupID, "type": "user", "active":0}}}
					info, err := c.Upsert(colQuerier, upsertData)
					if err != nil {
						log.Println(err)
						//if err == mgp.findAndModifyFailed {
							upsertData := bson.M{"$push": bson.M{"GROUPINFO": bson.M{"id": event.Source.GroupID, "type": "group", "active":0, "license":0}}}
							info, err := c.UpsertId(bson.ObjectIdHex("5a69aa488d0d213fd88abd95"), upsertData)
							if err != nil {
								log.Println(err)
							}
							log.Println(info)
						//}
					}
				}*/
			
				
			}//evens == message
			
		}

		if event.Type == linebot.EventTypeJoin {
			var local *time.Location
			local, _ = time.LoadLocation("Asia/Taipei")
			_NowTime := time.Now().In(local)
			_NowTimeStr := _NowTime.Format("2006-01-02 15:04:05")

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
			colQuerier := bson.M{"GROUPINFO.id" : event.Source.GroupID}
			upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.id": event.Source.GroupID, "GROUPINFO.$.type": "group", "GROUPINFO.$.active":0, "GROUPINFO.$.license":0, "GROUPINFO.$.createtime":_NowTimeStr}}
			//upsertData := bson.M{"$set": bson.M{"GROUPINFO": bson.M{ "id": event.Source.GroupID, "type": "user", "active":0}}}
			info, err := c.Upsert(colQuerier, upsertData)
			if err != nil {
				log.Println(err)
				//if err == mgp.findAndModifyFailed {
					upsertData := bson.M{"$push": bson.M{"GROUPINFO": bson.M{"id": event.Source.GroupID, "type": "group", "active":0, "license":0 , "createtime": _NowTimeStr }}}
					info, err := c.UpsertId(bson.ObjectIdHex("5a69aa488d0d213fd88abd95"), upsertData)
					if err != nil {
						log.Println(err)
					}
					log.Println(info)
				//}
			}
			log.Println(info)
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
			colQuerier := bson.M{"GROUPINFO.id" : event.Source.UserID}
			upsertData := bson.M{"$set": bson.M{"GROUPINFO.$.id": event.Source.UserID, "GROUPINFO.$.type": "user", "GROUPINFO.$.active":0, "GROUPINFO.$.license":0}}
			//upsertData := bson.M{"$set": bson.M{"GROUPINFO": bson.M{ "id": event.Source.UserID, "type": "user", "active":0}}}
			info, err := c.Upsert(colQuerier, upsertData)
			if err != nil {
				log.Println(err)
				//if err == mgp.findAndModifyFailed {
					upsertData := bson.M{"$push": bson.M{"GROUPINFO": bson.M{"id": event.Source.UserID, "type": "user", "active":0, "license":0}}}
					info, err := c.UpsertId(bson.ObjectIdHex("5a69aa488d0d213fd88abd95"), upsertData)
					if err != nil {
						log.Println(err)
					}
					log.Println(info)
				//}
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
