package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const PORT = ":3000"

// Possible characters for random name
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Our database
var db *bolt.DB

// This is used to display pastes in our html template
type Paste struct {
	Name  string
	Paste string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var err error
	db, err = bolt.Open("pastes.db", 0600, nil)
	if err != nil {
		log.Println(err)
		return
	}

	standardRouter := mux.NewRouter()

	standardRouter.HandleFunc("/", PasteHandler).Methods("GET")
	standardRouter.HandleFunc("/n", PasteNewHandler).Methods("POST")
	standardRouter.HandleFunc("/{key}", PasteServeHandler).Methods("GET")
	standardRouter.HandleFunc("/r/{key}", PasteServeRawHandler).Methods("GET")

	http.Handle("/", standardRouter)

	log.Println("Listening at localhost" + PORT)
	err = http.ListenAndServe(PORT, nil)
	if err != nil {
		log.Println(err)
		db.Close()
		return
	}
	db.Close()
}

// Serves paste creation page
func PasteHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("views/new.html")
	if err != nil {
		log.Println(err)
	}
	t.Execute(w, nil)
}

// Handles creation of new paste
func PasteNewHandler(w http.ResponseWriter, r *http.Request) {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("pastes"))
		if err != nil {
			return err
		}

		// Generate string for filename and concat with given filetype
		key := randSeq(10) + "." + r.FormValue("filetype")
		paste := []byte(r.FormValue("paste"))

		err = bucket.Put([]byte(key), paste)
		if err != nil {
			return err
		}

		// Redirect user to view created paste
		http.Redirect(w, r, "/"+key, 302)
		return nil
	})
	if err != nil {
		// Something went wrong...
		log.Println(err)
		return
	}

}

// Serve contents of paste in html viewer
func PasteServeHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["key"]
	paste, err := readPaste(name)
	if err != nil {
		log.Println(err)
		return
	}

	p := Paste{
		Name:  name,
		Paste: string(paste),
	}

	t, err := template.ParseFiles("views/view.html")
	if err != nil {
		log.Println(err)
	}
	t.Execute(w, p)
}

// Serve contents of paste as plaintext with filename of `randSeq + filetype`
func PasteServeRawHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["key"]
	paste, err := readPaste(name)
	if err != nil {
		log.Println(err)
		return
	}

	w.Header()["Content-Type"] = []string{"text/plain; charset=utf-8"}
	fmt.Fprint(w, string(paste))
}

func readPaste(name string) (string, error) {
	var paste []byte

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("pastes"))
		if bucket == nil {
			return fmt.Errorf("Bucket pastes not found!")
		}

		paste = bucket.Get([]byte(name))
		if paste == nil {
			return fmt.Errorf("Paste not found!")
		}

		return nil
	})

	return string(paste), err
}

// Generates random string of length n
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
