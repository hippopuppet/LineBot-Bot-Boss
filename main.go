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
	"cloud.google.com/go/storage"
	"google.golang.org/appengine"
    "google.golang.org/appengine/file"
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

// demo struct holds information needed to run the various demo functions.
type demo struct {
	client     *storage.Client
	bucketName string
	bucket     *storage.BucketHandle

	w   io.Writer
	ctx context.Context
	// cleanUp is a list of filenames that need cleaning up at the end of the demo.
	cleanUp []string
	// failed indicates that one or more of the demo steps failed.
	failed bool
}

func (d *demo) errorf(format string, args ...interface{}) {
	d.failed = true
	fmt.Fprintln(d.w, fmt.Sprintf(format, args...))
	log.Errorf(d.ctx, format, args...)
}

//[START write]
// createFile creates a file in Google Cloud Storage.
func (d *demo) createFile(fileName string) {
	fmt.Fprintf(d.w, "Creating file /%v/%v\n", d.bucketName, fileName)

	wc := d.bucket.Object(fileName).NewWriter(d.ctx)
	wc.ContentType = "text/plain"
	wc.Metadata = map[string]string{
		"x-goog-meta-foo": "foo",
		"x-goog-meta-bar": "bar",
	}
	d.cleanUp = append(d.cleanUp, fileName)

	if _, err := wc.Write([]byte("abcde\n")); err != nil {
		d.errorf("createFile: unable to write data to bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	if _, err := wc.Write([]byte(strings.Repeat("f", 1024*4) + "\n")); err != nil {
		d.errorf("createFile: unable to write data to bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	if err := wc.Close(); err != nil {
		d.errorf("createFile: unable to close bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
}

//[END write]

//[START read]
// readFile reads the named file in Google Cloud Storage.
func (d *demo) readFile(fileName string) {
	io.WriteString(d.w, "\nAbbreviated file content (first line and last 1K):\n")

	rc, err := d.bucket.Object(fileName).NewReader(d.ctx)
	if err != nil {
		d.errorf("readFile: unable to open file from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		d.errorf("readFile: unable to read data from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}

	fmt.Fprintf(d.w, "%s\n", bytes.SplitN(slurp, []byte("\n"), 2)[0])
	if len(slurp) > 1024 {
		fmt.Fprintf(d.w, "...%s\n", slurp[len(slurp)-1024:])
	} else {
		fmt.Fprintf(d.w, "%s\n", slurp)
	}
}

//[END read]

//[START copy]
// copyFile copies a file in Google Cloud Storage.
func (d *demo) copyFile(fileName string) {
	copyName := fileName + "-copy"
	fmt.Fprintf(d.w, "Copying file /%v/%v to /%v/%v:\n", d.bucketName, fileName, d.bucketName, copyName)

	obj, err := d.bucket.Object(copyName).CopierFrom(d.bucket.Object(fileName)).Run(d.ctx)
	if err != nil {
		d.errorf("copyFile: unable to copy /%v/%v to bucket %q, file %q: %v", d.bucketName, fileName, d.bucketName, copyName, err)
		return
	}
	d.cleanUp = append(d.cleanUp, copyName)

	d.dumpStats(obj)
}

//[END copy]

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
								//[START get_default_bucket]
								// Use `dev_appserver.py --default_gcs_bucket_name GCS_BUCKET_NAME`
								// when running locally.
								bucket, err := file.DefaultBucketName(ctx)
								if err != nil {
									log.Errorf(ctx, "failed to get default GCS bucket name: %v", err)
								}
								//[END get_default_bucket]
								client, err := storage.NewClient(ctx)
								if err != nil {
									log.Errorf(ctx, "failed to create client: %v", err)
									return
								}
								defer client.Close()
								
								buf := &bytes.Buffer{}
								d := &demo{
									w:          buf,
									ctx:        ctx,
									client:     client,
									bucket:     client.Bucket(bucket),
									bucketName: bucket,
								}

								n := "demo-testfile-go"
								d.createFile(n)
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
