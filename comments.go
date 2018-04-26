package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var tmpl = template.Must(template.ParseFiles("comments.html"))

func getLatestComments(w http.ResponseWriter, r *http.Request) {
	resp, err := yt.CommentThreads.List("snippet,replies").AllThreadsRelatedToChannelId(*chanID).MaxResults(100).Context(r.Context()).Do()
	if err != nil {
		fmt.Fprintf(w, "Unable to retrieve data from youtube: %v", err)
		return
	}

	log.Print("API list request complete")
	//log.Printf("%# v", pretty.Formatter(resp))

	data := CommentsPage{
		RequestedTime: time.Now().UTC(),
		NextPageToken: resp.NextPageToken,
	}

	for _, i := range resp.Items {
		c := &Comment{
			ID:             i.Id,
			VideoID:        i.Snippet.VideoId,
			AuthorName:     i.Snippet.TopLevelComment.Snippet.AuthorDisplayName,
			CommentHTML:    template.HTML(i.Snippet.TopLevelComment.Snippet.TextDisplay),
			CommentText:    i.Snippet.TopLevelComment.Snippet.TextOriginal,
			LastUpdateTime: i.Snippet.TopLevelComment.Snippet.UpdatedAt,
			ParentID:       i.Snippet.TopLevelComment.Snippet.ParentId,
		}
		data.Comments = append(data.Comments, c)

		t, err := time.Parse(time.RFC3339Nano, c.LastUpdateTime)
		if err != nil {
			continue
		}

		c.LastUpdateTime = data.RequestedTime.Sub(t).Truncate(time.Second).String()
	}
	//comment ID: Ugy8exLtYQ89HMMnFmd4AaABAg
	//video ID: tlkfNhvsW6I

	//http://youtube.com/watch?v=k2U7MdhioYw&lc=UgwFX-MnbhTCqMqWwwd4AaABAg
	//http://youtube.com/watch?v=k2U7MdhioYw&lc=UgxlYKTh-ruHKufc78F4AaABAg

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Unable to execute template: %v", err)
	}
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
	RequestedTime time.Time
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
	LastUpdateTime string
	ParentID       string
}
