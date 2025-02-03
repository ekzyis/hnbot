package main

import (
	"errors"
	"log"
	"time"

	"gitlab.com/ekzyis/hnbot/db"
	"gitlab.com/ekzyis/hnbot/hn"
	sn "gitlab.com/ekzyis/hnbot/sn"
)

func SyncHnItemsToDb() {
	for {
		now := time.Now()
		dur := now.Truncate(time.Minute).Add(time.Minute).Sub(now)
		log.Println("[hn] sleeping for", dur.Round(time.Second))
		time.Sleep(dur)

		stories, err := hn.FetchTopItems()
		if err != nil {
			log.Println(err)
			continue
		}
		if err := db.SaveHnItems(&stories); err != nil {
			log.Println(err)
			continue
		}
	}
}

func main() {
	// fetch HN front page every minute in the background and store state in db
	go SyncHnItemsToDb()

	// check every 15 minutes if there is now a HN item that is worth posting to SN
	for {
		var (
			filtered *[]hn.Item
			err      error
		)

		now := time.Now()
		dur := now.Truncate(time.Minute).Add(15 * time.Minute).Sub(now)
		log.Println("[sn] sleeping for", dur.Round(time.Second))
		time.Sleep(dur)

		if filtered, err = sn.CurateContent(); err != nil {
			log.Println(err)
			continue
		}

		log.Printf("[sn] found %d item(s) to post\n", len(*filtered))

		for _, item := range *filtered {
			_, err := sn.Post(&item, sn.PostOptions{SkipDupes: false})
			if err != nil {
				var dupesErr *sn.DupesError
				if errors.As(err, &dupesErr) {
					log.Println(dupesErr)
					parentId := dupesErr.Dupes[0].Id
					if err := db.SaveSnItem(parentId, item.ID); err != nil {
						log.Println(err)
					}
					continue
				}
				log.Println(err)
				continue
			}
		}
	}
}
