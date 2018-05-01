package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/api/youtube/v3"

	"github.com/gorilla/mux"
)

const MAX_PAGES = 5

var tmpl = template.Must(template.ParseFiles("comments.html"))

func getLatestComments(w http.ResponseWriter, r *http.Request) {

	lastSeen := getLastSeen(r)

	// lets keep getting pages until we find our comment or slip past our oldest time
	stTime := time.Now()

	data := CommentsPage{
		RequestedTime: stTime.Truncate(time.Second).String(),
	}

	// keep getting pages until we're done
	for done, page := false, 1; !done; page++ {
		resp, err := yt.CommentThreads.List("snippet,replies").
			AllThreadsRelatedToChannelId(*chanID).
			MaxResults(100).
			Context(r.Context()).
			PageToken(data.NextPageToken).
			Do()

		if err != nil {
			fmt.Fprintf(w, "Unable to retrieve data from youtube: %v", err)
			return
		}

		log.Printf("API list page %v request complete", page)
		//log.Printf("%# v", pretty.Formatter(resp))

		data.NextPageToken = resp.NextPageToken

		for _, i := range resp.Items {
			c := newComment(i.Id, stTime, i.Snippet.TopLevelComment.Snippet)
			data.Comments = append(data.Comments, c)

			if !done && lastSeen != nil {
				// check the ID, but also the time in case the ID was removed
				if lastSeen.ID == c.ID || c.LastUpdateTime.Before(lastSeen.LastUpdateTime) {
					c.MarkLineBefore = true
					done = true
				}
			}

			// now add the replies of the comment
			if i.Replies != nil {
				for _, ir := range i.Replies.Comments {
					rc := newComment(ir.Id, stTime, ir.Snippet)
					data.Comments = append(data.Comments, rc)

					if !done && lastSeen != nil {
						// check the ID, only check IDs for replies
						if lastSeen.ID == rc.ID {
							rc.MarkLineBefore = true
							done = true
						}
					}
				}
			}
		}

		if page > MAX_PAGES {
			// only get 5 pages max...that's 500 comments, maybe check the app more often
			log.Printf("Hit limit of %v pages, maybe check the app more often.", MAX_PAGES)
			done = true
		} else if lastSeen == nil {
			done = true
		}
	}

	if len(data.Comments) > 0 {
		// save our last seen
		saveLastSeen(data.Comments[0])
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Unable to execute template: %v", err)
	}
}

func newComment(id string, stTime time.Time, s *youtube.CommentSnippet) *Comment {
	c := &Comment{
		ID:          id,
		VideoID:     s.VideoId,
		AuthorName:  s.AuthorDisplayName,
		CommentHTML: template.HTML(s.TextDisplay),
		CommentText: s.TextOriginal,
		ParentID:    s.ParentId,
	}

	t, err := time.Parse(time.RFC3339Nano, s.UpdatedAt)
	if err != nil {
		return c
	}

	c.LastUpdateTime = t
	c.UpdatedSince = stTime.Sub(t).Truncate(time.Second).String()

	return c
}

func getLastSeen(r *http.Request) *LastSeen {
	data, err := ioutil.ReadFile("lastseen")
	if err != nil {

		if err != os.ErrNotExist {
			log.Printf("Cannot read last seen file: %v", err)
		}
		return nil
	}

	ls := &LastSeen{}
	if err := json.Unmarshal(data, ls); err != nil {
		log.Printf("Cannot unmarshal last seen file: %v", err)
		return nil
	}

	return ls
}

func saveLastSeen(c *Comment) {
	data, err := json.Marshal(&LastSeen{ID: c.ID, LastUpdateTime: c.LastUpdateTime})
	if err != nil {
		log.Printf("Error making json for last seen: %v", err)
	}
	ioutil.WriteFile("lastseen", data, os.ModePerm)
}

func removeComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	log.Printf("Delete: %v", id)

	// delete the comment
	err := yt.Comments.SetModerationStatus(id, "heldForReview").Context(r.Context()).Do()

	if err != nil {
		w.WriteHeader(http.StatusFailedDependency)
		fmt.Fprintf(w, "Unable to delete: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type CommentsPage struct {
	RequestedTime string
	NextPageToken string
	Comments      []*Comment
}

// Comment for our template
type Comment struct {
	ID             string
	VideoID        string
	AuthorName     string
	CommentHTML    template.HTML
	CommentText    string
	LastUpdateTime time.Time
	UpdatedSince   string
	ParentID       string
	MarkLineBefore bool
}

// LastSeen stores metadata about the last comment viewed in the app
type LastSeen struct {
	ID             string
	LastUpdateTime time.Time
}
